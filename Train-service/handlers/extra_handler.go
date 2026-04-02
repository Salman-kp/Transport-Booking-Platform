package handlers

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/nabeel-mp/tripneo/train-service/db"
	"github.com/nabeel-mp/tripneo/train-service/kafka"
	"github.com/nabeel-mp/tripneo/train-service/models"
	"github.com/nabeel-mp/tripneo/train-service/repository"
	"github.com/nabeel-mp/tripneo/train-service/utils"
	goredis "github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// GetClasses handles GET /api/train/:id/classes
func GetClasses() fiber.Handler {
	return func(c fiber.Ctx) error {
		scheduleID := c.Params("id")

		var schedule models.TrainSchedule
		if err := db.DB.Preload("Train").First(&schedule, "id = ?", scheduleID).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "schedule not found"})
		}

		type ClassInfo struct {
			Class          string  `json:"class"`
			AvailableSeats int     `json:"available_seats"`
			AveragePrice   float64 `json:"average_price"`
		}

		classes := []struct {
			name  string
			avail int
		}{
			{"SL", schedule.AvailableSL},
			{"3AC", schedule.Available3AC},
			{"2AC", schedule.Available2AC},
			{"1AC", schedule.Available1AC},
		}

		result := make([]ClassInfo, 0)
		for _, cls := range classes {
			if cls.avail <= 0 {
				continue
			}
			var avgPrice float64
			db.DB.Model(&models.TrainInventory{}).
				Where("train_schedule_id = ? AND class = ? AND status = 'AVAILABLE'", scheduleID, cls.name).
				Select("COALESCE(AVG(price), 0)").Scan(&avgPrice)

			result = append(result, ClassInfo{
				Class:          cls.name,
				AvailableSeats: cls.avail,
				AveragePrice:   avgPrice,
			})
		}

		return c.Status(200).JSON(fiber.Map{
			"schedule_id": scheduleID,
			"train":       schedule.Train.TrainName,
			"date":        schedule.ScheduleDate.Format("2006-01-02"),
			"classes":     result,
		})
	}
}

// LockSeat handles POST /api/train/:scheduleId/seats/:seatId/lock
func LockSeat(rdb *goredis.Client) fiber.Handler {
	return func(c fiber.Ctx) error {
		scheduleID := c.Params("scheduleId")
		seatID := c.Params("seatId")
		userID, _ := c.Locals("userID").(string)

		acquired, err := utils.LockSeat(c.Context(), rdb, scheduleID, seatID, userID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to acquire lock: " + err.Error()})
		}
		if !acquired {
			return c.Status(409).JSON(fiber.Map{
				"locked": false,
				"error":  "seat is held by another user",
			})
		}
		return c.Status(200).JSON(fiber.Map{
			"locked":     true,
			"seat_id":    seatID,
			"expires_in": "10 minutes",
		})
	}
}

// UnlockSeat handles DELETE /api/train/:scheduleId/seats/:seatId/lock
func UnlockSeat(rdb *goredis.Client) fiber.Handler {
	return func(c fiber.Ctx) error {
		scheduleID := c.Params("scheduleId")
		seatID := c.Params("seatId")

		if err := utils.UnlockSeat(c.Context(), rdb, scheduleID, seatID); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to release lock: " + err.Error()})
		}
		return c.Status(200).JSON(fiber.Map{"unlocked": true, "seat_id": seatID})
	}
}

// PaymentCallback handles POST /api/train/internal/payment/callback
// Called directly by the Payment Service (no user auth — internal endpoint only).
func PaymentCallback(rdb *goredis.Client, producer *kafka.Producer) fiber.Handler {
	return func(c fiber.Ctx) error {
		var req struct {
			BookingID  string  `json:"booking_id"  validate:"required"`
			Status     string  `json:"status"      validate:"required,oneof=COMPLETED FAILED"`
			PaymentRef string  `json:"payment_ref"`
			Amount     float64 `json:"amount"`
		}
		if err := c.Bind().Body(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid callback body"})
		}
		if err := validate.Struct(req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		ctx := context.Background()
		booking, err := repository.GetBookingByID(req.BookingID)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "booking not found"})
		}

		if req.Status == "COMPLETED" {
			seatIDs, _ := repository.GetSeatIDsByBooking(req.BookingID)

			txErr := db.DB.Transaction(func(tx *gorm.DB) error {
				if err := repository.ConfirmBooking(tx, req.BookingID, req.PaymentRef); err != nil {
					return err
				}
				return repository.MarkSeatsBooked(tx, seatIDs)
			})
			if txErr != nil {
				return c.Status(500).JSON(fiber.Map{"error": "failed to confirm booking: " + txErr.Error()})
			}

			_ = utils.UnlockSeats(ctx, rdb, booking.ScheduleID.String(), seatIDs)

			// Issue ticket now that payment is confirmed
			ticketNumber := "TKT-" + booking.PNR + "-" + time.Now().Format("20060102")
			ticket := models.TrainTicket{
				BookingID:    booking.ID,
				TicketNumber: ticketNumber,
				QRCodeURL:    "/api/train/tickets/" + booking.ID.String() + "/qr",
				QRData:       utils.GenerateQRToken(booking.ID.String()),
				IssuedAt:     time.Now(),
			}
			if err := db.DB.Create(&ticket).Error; err != nil {
				log.Printf("[callback] Warning: ticket creation failed for booking %s: %v", req.BookingID, err)
			}

			producer.PublishBookingConfirmed(ctx, kafka.BookingConfirmedEvent{
				BookingID:   booking.ID.String(),
				PNR:         booking.PNR,
				UserID:      booking.UserID,
				TotalAmount: booking.TotalAmount,
			})
			producer.PublishNotification(ctx, kafka.NotificationEvent{
				Type:            "BOOKING_CONFIRMED",
				RecipientUserID: booking.UserID,
				Template:        "booking_confirmed",
				Data: map[string]interface{}{
					"pnr":          booking.PNR,
					"booking_id":   booking.ID.String(),
					"total_amount": booking.TotalAmount,
				},
			})

			log.Printf("[callback] Booking %s CONFIRMED via callback", req.BookingID)
			return c.Status(200).JSON(fiber.Map{"message": "booking confirmed"})
		}

		// --- FAILED branch ---
		seatIDs, _ := repository.GetSeatIDsByBooking(req.BookingID)
		db.DB.Transaction(func(tx *gorm.DB) error {
			repository.UpdateBookingStatus(tx, req.BookingID, "FAILED")
			repository.UpdateSeatStatuses(tx, seatIDs, "AVAILABLE")
			repository.IncrementAvailability(tx, booking.ScheduleID.String(), booking.SeatClass, len(seatIDs))
			return nil
		})
		_ = utils.UnlockSeats(ctx, rdb, booking.ScheduleID.String(), seatIDs)

		log.Printf("[callback] Booking %s FAILED via callback", req.BookingID)
		return c.Status(200).JSON(fiber.Map{"message": "booking marked failed"})
	}
}

// GetStations handles GET /api/train/stations
func GetStations() fiber.Handler {
	return func(c fiber.Ctx) error {
		var stations []models.Station
		if err := db.DB.Order("city ASC, name ASC").Find(&stations).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to fetch stations"})
		}
		return c.Status(200).JSON(fiber.Map{
			"count":    len(stations),
			"stations": stations,
		})
	}
}
