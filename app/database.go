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
}

func runMigrations() error {
	DB.Exec("SET FOREIGN_KEY_CHECKS = 0")

	DB.Exec("DROP TABLE IF EXISTS `news_category_associations`")
	
	// Migration theo thứ tự quan trọng - User phải được tạo trước
	migrationOrder := []interface{}{
		&model.User{},           // Tạo trước vì News cần reference
		&model.Category{},
		&model.Brand{},
		&model.Product{},
		&model.ProductImage{},
		&model.Review{},
		&model.Cart{},
		&model.CartItem{},
		&model.Order{},
		&model.OrderItem{},
		&model.Discount{},
		&model.DiscountProduct{},
		&model.DiscountCategory{},
		&model.UserDiscountUsage{},
		&model.Address{},
		&model.NewsCategory{},   // NewsCategory trước News
		&model.Tag{},            // Tag trước News
		&model.News{},           // News sau khi User, NewsCategory, Tag đã tồn tại
		&model.ProductTag{},
		&model.NewsTag{},
	}

	// Migrate từng model một cách tuần tự
	for _, modelPtr := range migrationOrder {
		// Xử lý đặc biệt cho Product model để fix slug duplicates
		if _, ok := modelPtr.(*model.Product); ok {
			if err := fixProductSlugDuplicates(); err != nil {
				log.Printf("⚠️  Warning: Failed to fix product slugs: %v", err)
			}
		}

		if err := DB.AutoMigrate(modelPtr); err != nil {
			return err
		}
		// log.Printf("✅ Migrated %T", modelPtr)
	}

	if err := DB.AutoMigrate(&model.NewsCategoryAssociation{}); err != nil {
		log.Printf("❌ Failed to migrate NewsCategoryAssociation: %v", err)
		return err
	}
	
	// Check if primary key exists before adding it
	var keyExists int
	DB.Raw(`SELECT COUNT(*) FROM information_schema.table_constraints 
			WHERE table_schema = DATABASE() 
			AND table_name = 'news_category_associations' 
			AND constraint_type = 'PRIMARY KEY'`).Scan(&keyExists)
	
	if keyExists == 0 {
		// Add composite primary key only if it doesn't exist
		if err := DB.Exec(`ALTER TABLE news_category_associations 
			ADD PRIMARY KEY (news_id, category_id)`).Error; err != nil {
		} 
	} 
	

	// Bật lại foreign key checks
	DB.Exec("SET FOREIGN_KEY_CHECKS = 1")

	// Tạo user mặc định nếu chưa có
	if err := createDefaultUser(); err != nil {
		log.Printf("⚠️  Warning: Failed to create default user: %v", err)
	}

	return nil
}

// fixProductSlugDuplicates sửa các slug duplicate trong products
func fixProductSlugDuplicates() error {
	// Kiểm tra xem bảng products có tồn tại không
	var count int64
	if err := DB.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'products'").Scan(&count).Error; err != nil {
		return nil // Bảng chưa tồn tại, không cần fix
	}
	if count == 0 {
		return nil // Bảng chưa tồn tại
	}

	// Fix empty slugs
	updateEmptySlugSQL := `
		UPDATE products 
		SET slug = LOWER(CONCAT(
			REPLACE(REPLACE(REPLACE(REPLACE(COALESCE(name, 'product'), ' ', '-'), '&', 'and'), '.', ''), '/', '-'),
			'-',
			SUBSTRING(id, 1, 8)
		))
		WHERE slug IS NULL OR slug = '' OR TRIM(slug) = ''
	`
	if err := DB.Exec(updateEmptySlugSQL).Error; err != nil {
		log.Printf("⚠️  Warning: Failed to update empty slugs: %v", err)
	}

	// Fix duplicate slugs
	fixDuplicateSlugSQL := `
		UPDATE products p1
		INNER JOIN (
			SELECT slug, MIN(id) as min_id
			FROM products 
			WHERE slug IS NOT NULL AND slug != ''
			GROUP BY slug 
			HAVING COUNT(*) > 1
		) p2 ON p1.slug = p2.slug AND p1.id != p2.min_id
		SET p1.slug = CONCAT(p1.slug, '-', SUBSTRING(p1.id, 1, 8))
	`
	if err := DB.Exec(fixDuplicateSlugSQL).Error; err != nil {
		log.Printf("⚠️  Warning: Failed to fix duplicate slugs: %v", err)
	}

	// Ensure all products have valid slugs
	ensureSlugSQL := `
		UPDATE products 
		SET slug = CONCAT('product-', SUBSTRING(id, 1, 8))
		WHERE slug IS NULL OR slug = '' OR TRIM(slug) = ''
	`
	if err := DB.Exec(ensureSlugSQL).Error; err != nil {
		log.Printf("⚠️  Warning: Failed to ensure valid slugs: %v", err)
	}

	return nil
}

// createDefaultUser tạo user mặc định để tránh lỗi foreign key
func createDefaultUser() error {
	var userCount int64
	DB.Model(&model.User{}).Count(&userCount)
	
	if userCount == 0 {
		log.Println("🔄 Creating default admin user...")
		
		defaultUser := model.User{
			Username:        "admin",
			FullName:        "Administrator", 
			Email:           "admin@example.com",
			Password:        "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", // password: password
			Role:            "admin",
			IsActive:        true,
			IsEmailVerified: true,
		}
		
		if err := DB.Create(&defaultUser).Error; err != nil {
			return fmt.Errorf("failed to create default user: %v", err)
		}
		
	}
	
	return nil
}

// ResetDatabase - CHỈ SỬ DỤNG CHO DEVELOPMENT KHI CẦN RESET TOÀN BỘ DATABASE!
// CÁCH SỬ DỤNG:
// 1. Uncomment toàn bộ function này
// 2. Trong main.go, thêm dòng: app.ResetDatabase() TRƯỚC app.Connect()
// 3. Chạy server 1 lần để reset
// 4. Comment lại function này và xóa dòng app.ResetDatabase() trong main.go
// 5. Restart server bình thường
/*
func ResetDatabase() error {
	if os.Getenv("GIN_MODE") == "release" {
		return fmt.Errorf("không thể reset database trong production mode")
	}
	
	log.Println("🚨 RESETTING DATABASE - ALL DATA WILL BE LOST!")
	
	// Tắt foreign key checks tạm thời
	DB.Exec("SET FOREIGN_KEY_CHECKS = 0")
	
	// Drop tất cả table
	DB.Migrator().DropTable(
		&model.OrderItem{},
		&model.Order{},
		&model.CartItem{},
		&model.Cart{},
		&model.Review{},
		&model.ProductImage{},
		&model.Product{},
		&model.Category{},
		&model.Brand{},
		&model.News{},
		&model.Address{},
		&model.Coupon{},
		&model.User{},
	)
	
	log.Println("✅ All tables dropped")
	
	// Migrate lại
	err := runMigrations()
	if err != nil {
		log.Printf("❌ Migration failed: %v", err)
		return err
	}
	
	// Bật lại foreign key checks
	DB.Exec("SET FOREIGN_KEY_CHECKS = 1")
	
	log.Println("✅ Database reset completed!")
	return nil
}
*/

func GetDB() *gorm.DB {
	return DB
}
