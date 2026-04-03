package handlers

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/lib/pq"
	"github.com/nabeel-mp/tripneo/train-service/db"
	"github.com/nabeel-mp/tripneo/train-service/jobs"
	"github.com/nabeel-mp/tripneo/train-service/models"
	"github.com/nabeel-mp/tripneo/train-service/repository"
)

// AdminCreateTrain handles POST /api/train/admin/trains
func AdminCreateTrain() fiber.Handler {
	return func(c fiber.Ctx) error {
		var req struct {
			TrainNumber string  `json:"train_number" validate:"required"`
			TrainName   string  `json:"train_name"   validate:"required"`
			DaysOfWeek  []int32 `json:"days_of_week" validate:"required"`
		}
		if err := c.Bind().Body(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
		}
		if err := validate.Struct(req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		train := models.Train{
			TrainNumber: req.TrainNumber,
			TrainName:   req.TrainName,
			DaysOfWeek:  pq.Int32Array(req.DaysOfWeek),
			IsActive:    true,
		}
		if err := db.DB.Create(&train).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to create train: " + err.Error()})
		}
		return c.Status(201).JSON(train)
	}
}

// AdminUpdateTrain handles PUT /api/train/admin/trains/:id
func AdminUpdateTrain() fiber.Handler {
	return func(c fiber.Ctx) error {
		trainID := c.Params("id")
		var req struct {
			TrainName string `json:"train_name"`
			IsActive  *bool  `json:"is_active"`
		}
		if err := c.Bind().Body(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
		}
		updates := map[string]interface{}{}
		if req.TrainName != "" {
			updates["train_name"] = req.TrainName
		}
		if req.IsActive != nil {
			updates["is_active"] = *req.IsActive
		}
		if len(updates) == 0 {
			return c.Status(400).JSON(fiber.Map{"error": "no fields to update"})
		}
		if err := db.DB.Model(&models.Train{}).Where("id = ?", trainID).Updates(updates).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "update failed: " + err.Error()})
		}
		return c.Status(200).JSON(fiber.Map{"message": "train updated"})
	}
}

// AdminGenerateSchedules handles POST /api/train/admin/schedules/generate
func AdminGenerateSchedules() fiber.Handler {
	return func(c fiber.Ctx) error {
		var req struct {
			Days int `json:"days"`
		}
		if err := c.Bind().Body(&req); err != nil || req.Days <= 0 {
			req.Days = 30
		}
		go jobs.GenerateUpcomingInventory(db.DB, req.Days)
		return c.Status(202).JSON(fiber.Map{
			"message": "schedule generation started",
			"days":    req.Days,
		})
	}
}

// AdminGetBookings handles GET /api/train/admin/bookings
func AdminGetBookings() fiber.Handler {
	return func(c fiber.Ctx) error {
		status := c.Query("status")
		page := 1
		if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
			page = p
		}
		limit := 50
		if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
			limit = l
		}
		offset := (page - 1) * limit

		query := db.DB.Model(&models.TrainBooking{}).
			Preload("TrainSchedule.Train").
			Order("created_at DESC").
			Limit(limit).
			Offset(offset)

		if status != "" {
			query = query.Where("status = ?", status)
		}

		var bookings []models.TrainBooking
		var total int64
		db.DB.Model(&models.TrainBooking{}).Count(&total)

		if err := query.Find(&bookings).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to fetch bookings"})
		}
		return c.Status(200).JSON(fiber.Map{
			"total":    total,
			"page":     page,
			"bookings": bookings,
		})
	}
}

// AdminGetRevenue handles GET /api/train/admin/analytics/revenue
func AdminGetRevenue() fiber.Handler {
	return func(c fiber.Ctx) error {
		fromStr := c.Query("from", time.Now().AddDate(0, -1, 0).Format("2006-01-02"))
		toStr := c.Query("to", time.Now().Format("2006-01-02"))

		from, _ := time.Parse("2006-01-02", fromStr)
		to, _ := time.Parse("2006-01-02", toStr)
		to = to.Add(24 * time.Hour) // include full end day

		var result struct {
			TotalRevenue   float64 `json:"total_revenue"`
			TotalBookings  int64   `json:"total_bookings"`
			ConfirmedCount int64   `json:"confirmed_count"`
			CancelledCount int64   `json:"cancelled_count"`
		}

		db.DB.Model(&models.TrainBooking{}).
			Where("created_at BETWEEN ? AND ? AND status = 'CONFIRMED'", from, to).
			Select("SUM(total_amount) as total_revenue, COUNT(*) as total_bookings").
			Scan(&result)

		db.DB.Model(&models.TrainBooking{}).
			Where("created_at BETWEEN ? AND ? AND status = 'CONFIRMED'", from, to).
			Count(&result.ConfirmedCount)

		db.DB.Model(&models.TrainBooking{}).
			Where("created_at BETWEEN ? AND ? AND status = 'CANCELLED'", from, to).
			Count(&result.CancelledCount)

		result.TotalBookings = result.ConfirmedCount + result.CancelledCount

		return c.Status(200).JSON(fiber.Map{
			"from":   fromStr,
			"to":     toStr,
			"result": result,
		})
	}
}

// AdminUpdatePricingRule handles PUT /api/train/admin/pricing-rules/:id
func AdminUpdatePricingRule() fiber.Handler {
	return func(c fiber.Ctx) error {
		ruleID := c.Params("id")
		var req struct {
			Multiplier *float64 `json:"multiplier"`
			IsActive   *bool    `json:"is_active"`
		}
		if err := c.Bind().Body(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
		}
		updates := map[string]interface{}{}
		if req.Multiplier != nil {
			updates["multiplier"] = *req.Multiplier
		}
		if req.IsActive != nil {
			updates["is_active"] = *req.IsActive
		}
		if len(updates) == 0 {
			return c.Status(400).JSON(fiber.Map{"error": "nothing to update"})
		}
		if err := db.DB.Model(&models.PricingRule{}).Where("id = ?", ruleID).Updates(updates).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "update failed: " + err.Error()})
		}
		return c.Status(200).JSON(fiber.Map{"message": "pricing rule updated"})
	}
}

// AdminPurchaseInventory handles POST /api/train/admin/inventory/purchase
// Marks existing BLOCKED seats as AVAILABLE (simulating a provider purchase).
func AdminPurchaseInventory() fiber.Handler {
	return func(c fiber.Ctx) error {
		var req struct {
			ScheduleID string   `json:"schedule_id" validate:"required,uuid"`
			Class      string   `json:"class"       validate:"required,oneof=SL 3AC 2AC 1AC"`
			SeatIDs    []string `json:"seat_ids"`
		}
		if err := c.Bind().Body(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
		}

		var ids []string
		if len(req.SeatIDs) > 0 {
			ids = req.SeatIDs
		} else {
			// If no specific seat IDs, mark all BLOCKED in this schedule+class
			var seats []models.TrainInventory
			db.DB.Where("train_schedule_id = ? AND class = ? AND status = 'BLOCKED'",
				req.ScheduleID, req.Class).Select("id").Find(&seats)
			for _, s := range seats {
				ids = append(ids, s.ID.String())
			}
		}

		if len(ids) == 0 {
			return c.Status(200).JSON(fiber.Map{"message": "no blocked seats found", "updated": 0})
		}

		if err := repository.MarkSeatsAvailable(db.DB, ids); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to update inventory: " + err.Error()})
		}

		return c.Status(200).JSON(fiber.Map{
			"message": "inventory marked AVAILABLE",
			"updated": len(ids),
		})
	}
}
