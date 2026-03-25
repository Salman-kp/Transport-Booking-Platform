package service

import (
	"fmt"

	domainerrors "github.com/nabeel-mp/tripneo/train-service/domain_errors"
	"github.com/nabeel-mp/tripneo/train-service/repository"
	"github.com/nabeel-mp/tripneo/train-service/utils"
)

func GetTicket(bookingID, userID string) (interface{}, error) {
	// First verify booking ownership
	booking, err := repository.GetBookingByID(bookingID)
	if err != nil {
		return nil, err
	}
	if booking.UserID != userID {
		return nil, domainerrors.ErrUnauthorized
	}
	if booking.Status != "CONFIRMED" {
		return nil, domainerrors.ErrBookingNotConfirmed
	}

	// Fetch ticket record
	ticket, err := repository.GetTicketByBookingID(bookingID)
	if err != nil {
		return nil, err
	}

	// Fetch passengers
	passengers, err := repository.GetPassengersByBookingID(bookingID)
	if err != nil {
		return nil, err
	}

	type TicketResponse struct {
		TicketNumber string      `json:"ticket_number"`
		PNR          string      `json:"pnr"`
		QRCodeURL    string      `json:"qr_code_url"`
		TrainName    string      `json:"train_name"`
		TrainNumber  string      `json:"train_number"`
		From         string      `json:"from"`
		To           string      `json:"to"`
		DepartureAt  interface{} `json:"departure_at"`
		ArrivalAt    interface{} `json:"arrival_at"`
		Class        string      `json:"class"`
		Passengers   interface{} `json:"passengers"`
		Status       string      `json:"status"`
	}

	return TicketResponse{
		TicketNumber: ticket.TicketNumber,
		PNR:          booking.PNR,
		QRCodeURL:    ticket.QRCodeURL,
		TrainName:    booking.TrainSchedule.Train.TrainName,
		TrainNumber:  booking.TrainSchedule.Train.TrainNumber,
		From:         booking.TrainSchedule.Train.OriginStation,
		To:           booking.TrainSchedule.Train.DestinationStation,
		DepartureAt:  booking.TrainSchedule.DepartureAt,
		ArrivalAt:    booking.TrainSchedule.ArrivalAt,
		Class:        booking.SeatClass,
		Passengers:   passengers,
		Status:       booking.Status,
	}, nil
}

func VerifyTicket(bookingID, token string) (interface{}, error) {
	booking, err := repository.GetBookingByID(bookingID)
	if err != nil {
		return nil, err
	}

	// Validate HMAC token
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
		"status":     booking.Status,
		"passengers": passengers,
	}, nil
}
