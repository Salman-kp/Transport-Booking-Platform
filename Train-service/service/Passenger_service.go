package service

import (
	"github.com/nabeel-mp/tripneo/train-service/models"
	"github.com/nabeel-mp/tripneo/train-service/repository"
)

// GetPassengers returns all passengers for a booking (ownership already verified by caller).
func GetPassengers(bookingID string) ([]models.Passenger, error) {
	return repository.GetPassengersByBookingID(bookingID)
}
