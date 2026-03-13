package repository

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/junaid9001/tripneo/auth-service/db"
	"github.com/junaid9001/tripneo/auth-service/models"
	"gorm.io/gorm"
)

var ErrEmailALreadyTaken = errors.New("email already taken")
var ErrEmailNotFound = errors.New("email not found")

func InsertUser(email, hashedPassword string) error {

	user := &models.User{
		Email:        email,
		PasswordHash: hashedPassword,
		CreatedAt:    time.Now(),
	}

	if err := db.DB.Create(&user).Error; err != nil {

		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrEmailALreadyTaken
		}

		return fmt.Errorf("Internal Server Error")
	}

	return nil
}

func FindUserByEmail(email string) (*models.User, error) {
	var user models.User
	if err := db.DB.Where("email=?", email).First(&user).Error; err != nil {
		log.Print(err.Error())
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEmailNotFound
		}

		return nil, fmt.Errorf("Internal Server Error")

	}
	return &user, nil
}

func UpdateUserVerified(email string) error {
	if err := db.DB.Model(&models.User{}).Where("email=?", email).Update("is_verified", true).Error; err != nil {
		log.Print(err)
		return fmt.Errorf("internal server error")
	}
	return nil
}
