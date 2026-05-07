package repository

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/junaid9001/tripneo/auth-service/db"
	domainerrors "github.com/junaid9001/tripneo/auth-service/domain_errors"
	"github.com/junaid9001/tripneo/auth-service/models"
	"gorm.io/gorm"
)

func InsertUser(name, email, hashedPassword string) error {

	user := &models.User{
		Name:         name,
		Email:        email,
		PasswordHash: hashedPassword,
		CreatedAt:    time.Now(),
	}

	if err := db.DB.Create(&user).Error; err != nil {

		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return domainerrors.EmailAlreadyTaken
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
			return nil, domainerrors.EmailNotFound
		}

		return nil, fmt.Errorf("Internal Server Error")

	}
	return &user, nil
}

func FindUserByID(id string) (*models.User, error) {
	var user models.User
	if err := db.DB.Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.EmailNotFound
		}
		return nil, fmt.Errorf("internal server error")
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

func AssignRole(email string, role string, permissions []string) error {
	if err := db.DB.Model(&models.User{}).Where("email=?", email).Updates(&models.User{
		Role:        role,
		Permissions: permissions,
	}).Error; err != nil {
		log.Print(err)
		return fmt.Errorf("internal server error")
	}
	return nil
}

func ListUsers() ([]models.User, error) {
	var users []models.User
	if err := db.DB.Find(&users).Error; err != nil {
		log.Print(err)
		return nil, fmt.Errorf("internal server error")
	}
	return users, nil
}
