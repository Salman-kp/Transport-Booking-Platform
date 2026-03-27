package service

import (
	"fmt"

	domainerrors "github.com/nabeel-mp/tripneo/train-service/domain_errors"
	"github.com/nabeel-mp/tripneo/train-service/repository"
	"github.com/nabeel-mp/tripneo/train-service/utils"
)

func GetTicket(bookingID, userID string) (interface{}, error) {
	booking, err := repository.GetBookingByID(bookingID)
	if err != nil {
		return nil, err
	}
	if booking.UserID != userID {
		return nil, domainerrors.ErrUnauthorized
	}

	ticket, err := repository.GetTicketByBookingID(bookingID)
	if err != nil {
		return nil, err
	}

	// Logic fix: Access boarding/destination stations directly from booking
	return map[string]interface{}{
		"ticket_number": ticket.TicketNumber,
		"pnr":           booking.PNR,
		"train_name":    booking.TrainSchedule.Train.TrainName,
		"train_number":  booking.TrainSchedule.Train.TrainNumber,
		"from":          booking.FromStation.Name, // Fixed
		"to":            booking.ToStation.Name,   // Fixed
		"departure_at":  booking.DepartureTime,
		"arrival_at":    booking.ArrivalTime,
		"status":        booking.Status,
	}, nil
}

// VerifyTicket validates a QR ticket's HMAC token.
func VerifyTicket(bookingID, token string) (interface{}, error) {
	booking, err := repository.GetBookingByID(bookingID)
	if err != nil {
		return nil, err
	}

	// Validate HMAC token to ensure the QR wasn't faked
	valid := utils.VerifyQRToken(bookingID, token)
	if !valid {
		return nil, fmt.Errorf("invalid or tampered QR token")
	}

	passengers, err := repository.GetPassengersByBookingID(bookingID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"valid":      true,
		"booking_id": bookingID,
		"pnr":        booking.PNR,
		"train":      booking.TrainSchedule.Train.TrainNumber,
		"route":      fmt.Sprintf("%s -> %s", booking.FromStation.Code, booking.ToStation.Code),
		"status":     booking.Status,
		"passengers": passengers,
	}, nil
}
