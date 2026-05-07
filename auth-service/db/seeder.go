package db

import (
	"log"
	"strings"
	"time"

	"github.com/junaid9001/tripneo/auth-service/config"
	"github.com/junaid9001/tripneo/auth-service/models"
	"github.com/junaid9001/tripneo/auth-service/utils"
	"gorm.io/gorm"
)

func SeedSuperAdmins(db *gorm.DB, cfg *config.Config) {
	if cfg.SUPER_ADMINS == "" {
		log.Println("No super admins to seed.")
		return
	}

	if cfg.SUPER_ADMIN_PASSWORD == "" {
		log.Println("SUPER_ADMIN_PASSWORD is not set. Skipping super admin seeding for security.")
		return
	}

	emails := strings.Split(cfg.SUPER_ADMINS, ",")
	for _, email := range emails {
		email = strings.TrimSpace(email)
		if email == "" {
			continue
		}

		var count int64
		db.Model(&models.User{}).Where("email = ?", email).Count(&count)
		if count > 0 {
			log.Printf("Super admin %s already exists. Skipping...", email)
			continue
		}

		// Create super admin
		hashedPass, err := utils.GenerateHashedPassword(cfg.SUPER_ADMIN_PASSWORD)
		if err != nil {
			log.Printf("Failed to hash password for super admin %s: %v", email, err)
			continue
		}

		user := &models.User{
			Name:         "Super Admin",
			Email:        email,
			PasswordHash: hashedPass,
			Role:         "superadmin",
			IsVerified:   true,
			Permissions:  []string{"*"}, // Full access
			CreatedAt:    time.Now(),
		}

		if err := db.Create(user).Error; err != nil {
			log.Printf("Failed to seed super admin %s: %v", email, err)
		} else {
			log.Printf("Successfully seeded super admin: %s", email)
		}
	}
}
