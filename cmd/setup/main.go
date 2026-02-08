package main

import (
	"backend/app"
	"backend/internal/model"
	"log"
	"os"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Connect to database
	app.Connect()
	db := app.GetDB()

	// Check if owner already exists
	var ownerCount int64
	db.Model(&model.User{}).Where("role = ?", "owner").Count(&ownerCount)

	if ownerCount > 0 {
		log.Println("Owner account already exists. Skipping creation.")
		return
	}

	// Create default owner account
	// Tài khoản dev: owner / owner123
	password := "owner123A@" // Default password - should be changed after first login
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}

	owner := model.User{
		Username: "owner",
		Email:    "leduytien202@gmail.com",
		Password: string(hashedPassword),
		FullName: "System Owner",
		Role:     "owner",
		IsActive: true,
		Avatar:   datatypes.JSON([]byte(`{}`)), // Set default empty JSON for avatar
	}

	if err := db.Create(&owner).Error; err != nil {
		log.Fatal("Failed to create owner account:", err)
	}

	log.Println("✅ Default owner account created successfully!")
	log.Println("Username: owner")
	log.Println("Password: owner123")
	log.Println("⚠️  Please change the password after first login!")

	os.Exit(0)
}
