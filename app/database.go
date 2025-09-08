package app

import (
	"backend/internal/model"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() {
	// Kết nối tới MySQL (chưa chọn DB) để tạo cơ sở dữ liệu nếu cần
	dsnWithoutDB := fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
	)

	// Cấu hình GORM giảm mức độ log
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn), // Chỉ hiển thị cảnh báo và lỗi
	}

	// Kết nối trước mà chưa chọn cơ sở dữ liệu
	tempDB, err := gorm.Open(mysql.Open(dsnWithoutDB), config)
	if err != nil {
		log.Fatal("❌ Failed to connect to MySQL server:", err)
	}

	// Tạo cơ sở dữ liệu nếu chưa tồn tại
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "backend"
	}
	
	createDBSQL := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", dbName)
	if err := tempDB.Exec(createDBSQL).Error; err != nil {
		log.Fatal("❌ Failed to create database:", err)
	}
	
	log.Printf("✅ Database '%s' ensured to exist", dbName)

	// Bây giờ kết nối tới cơ sở dữ liệu cụ thể
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		dbName,
	)

	// Kết nối tới cơ sở dữ liệu
	database, err := gorm.Open(mysql.Open(dsn), config)
	if err != nil {
		log.Fatal("❌ Failed to connect to database:", err)
	}

	// Cấu hình connection pool
	sqlDB, err := database.DB()
	if err != nil {
		log.Fatal("❌ Failed to get database instance:", err)
	}

	// Thiết lập tham số cho connection pool
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	DB = database

	// Chạy migration
	if err := runMigrations(); err != nil {
		log.Fatal("❌ Failed to run migrations:", err)
	}

	log.Println("✅ Database connected and migrated successfully!")
}

func runMigrations() error {
	// Tắt foreign key checks tạm thời
	DB.Exec("SET FOREIGN_KEY_CHECKS = 0")
	
	// Tự động migrate tất cả các model
	err := DB.AutoMigrate(
		&model.User{},
		&model.Category{},
		&model.Brand{},
		&model.Product{},
		&model.ProductImage{},
		&model.Review{},
		&model.Cart{},
		&model.CartItem{},
		&model.Order{},
		&model.OrderItem{},
		&model.Coupon{},
		&model.Address{},
		&model.News{},
	)
	
	// Bật lại foreign key checks
	DB.Exec("SET FOREIGN_KEY_CHECKS = 1")
	
	return err
}

func GetDB() *gorm.DB {
	return DB
}
