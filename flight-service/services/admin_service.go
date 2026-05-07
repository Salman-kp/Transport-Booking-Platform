package services

import (
	"errors"

	"github.com/google/uuid"
	"github.com/junaid9001/tripneo/flight-service/models"
	"github.com/junaid9001/tripneo/flight-service/repository"
	"gorm.io/gorm"
)

var (
	ErrInvalidAirlineID            = errors.New("invalid airline_id")
	ErrInvalidAircraftTypeID       = errors.New("invalid aircraft_type_id")
	ErrInvalidOriginAirportID      = errors.New("invalid origin_airport_id")
	ErrInvalidDestinationAirportID = errors.New("invalid destination_airport_id")
	ErrInvalidRoute                = errors.New("origin and destination cannot be the same airport")
)

type AdminService struct {
	repo *repository.BookingRepository
	db   *gorm.DB
}

func NewAdminService(repo *repository.BookingRepository, db *gorm.DB) *AdminService {
	return &AdminService{
		repo: repo,
		db:   db,
	}
}

// List all bookings in the system
func (s *AdminService) ListAllBookings() ([]models.Booking, error) {
	var bookings []models.Booking
	err := s.db.Preload("Passengers").
		Preload("FlightInstance").
		Preload("FlightInstance.Flight").
		Preload("FlightInstance.Flight.Airline").
		Preload("FlightInstance.Flight.OriginAirport").
		Preload("FlightInstance.Flight.DestinationAirport").
		Find(&bookings).Error
	return bookings, err
}

// List all flight templates
func (s *AdminService) ListFlightTemplates() ([]models.Flight, error) {
	var flights []models.Flight
	err := s.db.Preload("Airline").
		Preload("OriginAirport").
		Preload("DestinationAirport").
		Preload("AircraftType").
		Order("created_at DESC").
		Find(&flights).Error
	return flights, err
}

// Force update a booking status (Manual Override)
func (s *AdminService) ForceUpdateBookingStatus(bookingID uuid.UUID, status string) error {
	return s.db.Model(&models.Booking{}).Where("id = ?", bookingID).Update("status", status).Error
}

// Create a new Flight Template
func (s *AdminService) CreateFlight(flight *models.Flight) error {
	if flight.OriginAirportID == flight.DestinationAirportID {
		return ErrInvalidRoute
	}

	if err := s.ensureReferenceExists(&models.Airline{}, flight.AirlineID, ErrInvalidAirlineID); err != nil {
		return err
	}

	if err := s.ensureReferenceExists(&models.AircraftType{}, flight.AircraftTypeID, ErrInvalidAircraftTypeID); err != nil {
		return err
	}

	if err := s.ensureReferenceExists(&models.Airport{}, flight.OriginAirportID, ErrInvalidOriginAirportID); err != nil {
		return err
	}

	if err := s.ensureReferenceExists(&models.Airport{}, flight.DestinationAirportID, ErrInvalidDestinationAirportID); err != nil {
		return err
	}

	return s.db.Create(flight).Error
}

// Update an existing Flight Template
func (s *AdminService) UpdateFlight(flightID uuid.UUID, updates map[string]interface{}) error {
	return s.db.Model(&models.Flight{}).Where("id = ?", flightID).Updates(updates).Error
}

// Soft delete a flight
func (s *AdminService) SoftDeleteFlight(flightID uuid.UUID) error {
	return s.db.Model(&models.Flight{}).Where("id = ?", flightID).Update("is_active", false).Error
}

// Override prices on a specific flight instance
func (s *AdminService) OverridePrices(instanceID uuid.UUID, ecoPrice, busPrice float64) error {
	return s.db.Model(&models.FlightInstance{}).Where("id = ?", instanceID).Updates(map[string]interface{}{
		"current_price_economy":  ecoPrice,
		"current_price_business": busPrice,
	}).Error
}

func (s *AdminService) ensureReferenceExists(model interface{}, id uuid.UUID, notFoundError error) error {
	var count int64
	if err := s.db.Model(model).Where("id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return notFoundError
	}
	return nil
}

// CancelFlightInstance disables a specific flight instance (e.g. due to war, weather, ops issue).
// It cancels the instance and bulk-cancels all active bookings tied to it in a single transaction.
func (s *AdminService) CancelFlightInstance(instanceID uuid.UUID, reason string) (int64, error) {
	var cancelledBookings int64

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Verify instance exists and is not already cancelled
		var instance models.FlightInstance
		if err := tx.First(&instance, "id = ?", instanceID).Error; err != nil {
			return errors.New("flight instance not found")
		}
		if instance.Status == models.CANCELLED {
			return errors.New("flight instance is already cancelled")
		}

		// 2. Cancel the instance itself
		if err := tx.Model(&models.FlightInstance{}).
			Where("id = ?", instanceID).
			Update("status", models.CANCELLED).Error; err != nil {
			return err
		}

		// 3. Bulk-cancel all active bookings on this instance
		result := tx.Model(&models.Booking{}).
			Where("flight_instance_id = ? AND status IN ?", instanceID, []string{"CONFIRMED", "PENDING_PAYMENT"}).
			Update("status", "CANCELLED_BY_AIRLINE")
		if result.Error != nil {
			return result.Error
		}
		cancelledBookings = result.RowsAffected

		return nil
	})

	return cancelledBookings, err
}

