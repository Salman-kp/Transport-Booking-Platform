package domainerrors

import "errors"

var (
	ErrTrainNotFound           = errors.New("train not found")
	ErrScheduleNotFound        = errors.New("schedule not found")
	ErrNoSeatsAvailable        = errors.New("no confirmed seats available")
	ErrSeatAlreadyLocked       = errors.New("seat is temporarily held by another user")
	ErrSeatAlreadyBooked       = errors.New("seat is already booked")
	ErrBookingNotFound         = errors.New("booking not found")
	ErrBookingNotConfirmed     = errors.New("booking is not yet confirmed")
	ErrBookingAlreadyConfirmed = errors.New("booking is already confirmed")
	ErrBookingExpired          = errors.New("booking session has expired")
	ErrRefundNotEligible       = errors.New("not eligible for refund based on cancellation policy")
	ErrUnauthorized            = errors.New("you do not own this booking")
	ErrCannotCancel            = errors.New("only pending or confirmed bookings can be cancelled")
	ErrPNRNotFound             = errors.New("PNR not found")
)

