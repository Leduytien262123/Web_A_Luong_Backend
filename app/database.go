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
		log.Println("⚠️ Migration warning:", err)
	}

}

func runMigrations() error {
	log.Println("🔄 Running migrations...")

	DB.Exec("SET FOREIGN_KEY_CHECKS = 0")

	// Nếu bảng articles đã tồn tại, xóa các index/foreign key cũ trên tag_id
	// để tránh lỗi khi chuyển cột sang JSON (MySQL không cho phép index trực tiếp trên JSON).
	dropLegacyArticleTagIndexes()

	// QUAN TRỌNG: Chuyển đổi tag_id sang JSON TRƯỚC KHI chạy AutoMigrate
	// Nếu không, AutoMigrate sẽ cố gắng MODIFY cột với dữ liệu không hợp lệ
	if err := convertTagIDToJSON(); err != nil {
		log.Printf("⚠️  Warning during tag_id conversion: %v", err)
	}

	// Migration cho các bảng cần thiết
	migrationOrder := []interface{}{
		&model.User{},            // Tạo trước vì Article cần reference
		&model.Category{},        // Danh mục phân cấp 3 cấp
		&model.Tag{},             // Tag cho bài viết
		&model.Article{},         // Bài viết thuộc danh mục và tag
		&model.HomepageSection{}, // Các mục hiển thị ở trang chủ
	}

	// Migrate từng model một cách tuần tự
	for _, modelPtr := range migrationOrder {
		if err := DB.AutoMigrate(modelPtr); err != nil {
			return fmt.Errorf("failed to migrate %T: %v", modelPtr, err)
		}
		log.Printf("✅ Migrated %T", modelPtr)
	}

	// Chuẩn hóa giá trị mặc định cho cột is_active/is_hot sau khi thêm cột mới
	DB.Exec("UPDATE articles SET is_active = 1 WHERE is_active IS NULL")
	DB.Exec("UPDATE articles SET is_hot = 0 WHERE is_hot IS NULL")

	// Ensure all UUID columns have the same charset and collation
	log.Println("🔄 Fixing column charset and collation...")

	// Fix all UUID columns to have consistent charset/collation
	DB.Exec("ALTER TABLE users MODIFY COLUMN id CHAR(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci")
	DB.Exec("ALTER TABLE categories MODIFY COLUMN id CHAR(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci")
	DB.Exec("ALTER TABLE categories MODIFY COLUMN parent_id CHAR(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci")
	DB.Exec("ALTER TABLE tags MODIFY COLUMN id CHAR(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci")
	DB.Exec("ALTER TABLE articles MODIFY COLUMN id CHAR(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci")
	DB.Exec("ALTER TABLE articles MODIFY COLUMN category_id CHAR(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci")
	DB.Exec("ALTER TABLE articles MODIFY COLUMN author_id CHAR(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci")

	log.Println("🔄 Adding foreign key constraints...")

	// Drop existing constraints if they exist (sử dụng hàm helper để tránh warning)
	dropForeignKeyIfExists("articles", "fk_articles_author")
	dropForeignKeyIfExists("articles", "fk_articles_category")
	dropForeignKeyIfExists("articles", "fk_articles_tag")

	// Normalize legacy statuses to the new draft/post scheme
	DB.Exec("UPDATE articles SET status = 'post' WHERE status = 'published'")

	// Add foreign key constraints with proper ON DELETE/UPDATE actions
	if err := DB.Exec("ALTER TABLE articles ADD CONSTRAINT fk_articles_author FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE RESTRICT ON UPDATE CASCADE").Error; err != nil {
		log.Printf("⚠️  Warning: Could not add author foreign key (may already exist): %v", err)
	} else {
		log.Println("✅ Added author foreign key")
	}

	if err := DB.Exec("ALTER TABLE articles ADD CONSTRAINT fk_articles_category FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL ON UPDATE CASCADE").Error; err != nil {
		log.Printf("⚠️  Warning: Could not add category foreign key (may already exist): %v", err)
	} else {
		log.Println("✅ Added category foreign key")
	}

	// Fix cột order cũ thành display_order nếu tồn tại
	var orderColumnExists int
	DB.Raw(`SELECT COUNT(*) FROM information_schema.columns 
			WHERE table_schema = DATABASE() 
			AND table_name = 'categories' 
			AND column_name = 'order'`).Scan(&orderColumnExists)

	var displayOrderExists int
	DB.Raw(`SELECT COUNT(*) FROM information_schema.columns 
			WHERE table_schema = DATABASE() 
			AND table_name = 'categories' 
			AND column_name = 'display_order'`).Scan(&displayOrderExists)

	// Chỉ rename nếu cột order tồn tại VÀ display_order chưa tồn tại
	if orderColumnExists > 0 && displayOrderExists == 0 {
		log.Println("🔄 Renaming 'order' column to 'display_order'...")
		DB.Exec("ALTER TABLE categories CHANGE COLUMN `order` display_order INT DEFAULT 0")
		log.Println("✅ Column renamed successfully")
	} else if orderColumnExists > 0 && displayOrderExists > 0 {
		// Nếu cả 2 cột đều tồn tại, xóa cột order cũ
		log.Println("🔄 Dropping old 'order' column...")
		DB.Exec("ALTER TABLE categories DROP COLUMN `order`")
		log.Println("✅ Old column dropped")
	}

	// Bật lại foreign key checks
	DB.Exec("SET FOREIGN_KEY_CHECKS = 1")

	// Tạo user mặc định và danh mục mặc định
	if err := createDefaultData(); err != nil {
		log.Printf("⚠️  Warning: Failed to create default data: %v", err)
	}

	log.Println("✅ Migrations completed successfully")
	return nil
}

// convertTagIDToJSON chuyển đổi cột tag_id từ CHAR(36) sang JSON
// Hàm này PHẢI chạy TRƯỚC AutoMigrate để tránh lỗi dữ liệu không hợp lệ
func convertTagIDToJSON() error {
	// Kiểm tra bảng articles có tồn tại không
	var tableExists int
	DB.Raw(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'articles'`).Scan(&tableExists)
	if tableExists == 0 {
		return nil // Bảng chưa tồn tại, không cần chuyển đổi
	}

	// Kiểm tra kiểu dữ liệu hiện tại của cột tag_id
	var tagColumnType string
	DB.Raw(`SELECT DATA_TYPE FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'articles' AND column_name = 'tag_id'`).Scan(&tagColumnType)

	// Nếu đã là JSON hoặc cột không tồn tại, không cần làm gì
	if tagColumnType == "" || tagColumnType == "json" {
		return nil
	}

	log.Println("🔄 Converting articles.tag_id from", tagColumnType, "to JSON array...")

	// 1. Thêm cột tạm thời để lưu dữ liệu JSON
	if err := DB.Exec("ALTER TABLE articles ADD COLUMN tag_ids_temp JSON NULL").Error; err != nil {
		// Cột có thể đã tồn tại từ lần chạy trước bị lỗi
		log.Printf("⚠️  tag_ids_temp column may already exist: %v", err)
	}

	// 2. Chuyển đổi dữ liệu UUID đơn lẻ sang mảng JSON
	DB.Exec(`UPDATE articles SET tag_ids_temp = CASE 
		WHEN tag_id IS NULL OR TRIM(tag_id) = '' THEN JSON_ARRAY()
		ELSE JSON_ARRAY(tag_id)
		END`)

	// 3. Xóa cột cũ
	if err := DB.Exec("ALTER TABLE articles DROP COLUMN tag_id").Error; err != nil {
		log.Printf("⚠️  Could not drop old tag_id column: %v", err)
		return err
	}

	// 4. Đổi tên cột tạm thành tag_id
	if err := DB.Exec("ALTER TABLE articles CHANGE COLUMN tag_ids_temp tag_id JSON NULL").Error; err != nil {
		log.Printf("⚠️  Could not rename temp column: %v", err)
		return err
	}

	log.Println("✅ Converted tag_id to JSON array successfully")
	return nil
}

// dropLegacyArticleTagIndexes gỡ bỏ các index/foreign key cũ trên cột tag_id
// (kiểu cũ: CHAR/VARCHAR có index). Khi chuyển sang JSON, các index này khiến ALTER TABLE lỗi.
// Hàm an toàn: chỉ chạy nếu bảng articles tồn tại và index/FK thực sự tồn tại.
func dropLegacyArticleTagIndexes() {
	var tableExists int
	DB.Raw(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'articles'`).Scan(&tableExists)
	if tableExists == 0 {
		return
	}

	// Chỉ DROP nếu FK tồn tại
	var fkExists int
	DB.Raw(`SELECT COUNT(*) FROM information_schema.table_constraints 
			WHERE table_schema = DATABASE() AND table_name = 'articles' 
			AND constraint_name = 'fk_articles_tag' AND constraint_type = 'FOREIGN KEY'`).Scan(&fkExists)
	if fkExists > 0 {
		DB.Exec("ALTER TABLE articles DROP FOREIGN KEY fk_articles_tag")
	}

	// Chỉ DROP các index nếu tồn tại
	dropIndexIfExists("articles", "idx_articles_tag_id")
	dropIndexIfExists("articles", "tag_id")
	dropIndexIfExists("articles", "articles_tag_id_index")
}

// dropIndexIfExists xóa index nếu nó tồn tại
func dropIndexIfExists(tableName, indexName string) {
	var indexExists int
	DB.Raw(`SELECT COUNT(*) FROM information_schema.statistics 
			WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?`, tableName, indexName).Scan(&indexExists)
	if indexExists > 0 {
		DB.Exec(fmt.Sprintf("ALTER TABLE %s DROP INDEX %s", tableName, indexName))
	}
}

// dropForeignKeyIfExists xóa foreign key nếu nó tồn tại
func dropForeignKeyIfExists(tableName, constraintName string) {
	var fkExists int
	DB.Raw(`SELECT COUNT(*) FROM information_schema.table_constraints 
			WHERE table_schema = DATABASE() AND table_name = ? 
			AND constraint_name = ? AND constraint_type = 'FOREIGN KEY'`, tableName, constraintName).Scan(&fkExists)
	if fkExists > 0 {
		DB.Exec(fmt.Sprintf("ALTER TABLE %s DROP FOREIGN KEY %s", tableName, constraintName))
	}
}

// createDefaultData tạo dữ liệu mặc định: super admin và 6 danh mục chính
func createDefaultData() error {
	// Tạo super admin mặc định
	var userCount int64
	DB.Model(&model.User{}).Count(&userCount)

	if userCount == 0 {
		log.Println("🔄 Creating default super admin user...")

		superAdmin := model.User{
			Username:        "superadmin",
			FullName:        "Super Administrator",
			Email:           "leduytien202@gmail.com",
			Password:        "$2a$10$tv5wb.7uuGly2Fb5AoAh/e4B4DK23Qw8hubsaKdym/4wCJ5JLxLp6", // password: owner123A@
			Role:            "super_admin",
			IsActive:        true,
			IsEmailVerified: true,
		}

		if err := DB.Create(&superAdmin).Error; err != nil {
			log.Println("⚠️ Super admin exists or cannot create:", err)
			return nil
		}
		log.Println("✅ Super admin created successfully")
	}

	// Tạo 6 danh mục chính (lĩnh vực pháp luật)
	var categoryCount int64
	DB.Model(&model.Category{}).Count(&categoryCount)

	if categoryCount == 0 {
		log.Println("🔄 Creating default categories...")

		categories := []model.Category{
			{Name: "Xây dựng", Slug: "xay-dung", Description: "Các vấn đề pháp lý về xây dựng", IsActive: true, DisplayOrder: 1},
			{Name: "Doanh nghiệp và đầu tư", Slug: "doanh-nghiep-va-dau-tu", Description: "Luật doanh nghiệp và đầu tư", IsActive: true, DisplayOrder: 2},
			{Name: "Đất quốc phòng kết hợp với hoạt động kinh tế", Slug: "dat-quoc-phong-ket-hop-kinh-te", Description: "Pháp luật về đất quốc phòng", IsActive: true, DisplayOrder: 3},
			{Name: "Lao động", Slug: "lao-dong", Description: "Luật lao động và quan hệ lao động", IsActive: true, DisplayOrder: 4},
			{Name: "Hình sự", Slug: "hinh-su", Description: "Luật hình sự và tố tụng hình sự", IsActive: true, DisplayOrder: 5},
			{Name: "Giải quyết tranh chấp", Slug: "giai-quyet-tranh-chap", Description: "Các vấn đề về giải quyết tranh chấp", IsActive: true, DisplayOrder: 6},
		}

		for _, category := range categories {
			if err := DB.Create(&category).Error; err != nil {
				log.Printf("⚠️  Warning: Failed to create category %s: %v", category.Name, err)
			}
		}
		log.Println("✅ Default categories created successfully")
	}

	// Tạo Homepage Sections mặc định
	if err := createDefaultHomepageSections(); err != nil {
		log.Printf("⚠️  Warning: Failed to create homepage sections: %v", err)
	}

	return nil
}

func GetDB() *gorm.DB {
	return DB
}

// createDefaultHomepageSections tạo dữ liệu mặc định cho homepage sections
func createDefaultHomepageSections() error {
	var sectionCount int64
	DB.Model(&model.HomepageSection{}).Count(&sectionCount)

	if sectionCount > 0 {
		return nil // Đã có dữ liệu, không cần tạo mới
	}

	log.Println("🔄 Creating default homepage sections...")

	// TYPE01: Chúng tôi chuyên
	type01Metadata := `[
		{
			"title": "Chuyên môn",
			"des": "Giao diện dựa trên tailwindcss, tự động thích ứng PC và di động, tăng tính đáp ứng và khả năng sử dụng, cung cấp nhiều giao diện đẹp, giảm chi phí phát triển và bảo trì.",
			"icon": "ph:monitor",
			"color": "#e96656"
		},
		{
			"title": "Tầm nhìn",
			"des": "Cài đặt qua npm hoặc tải mã nguồn để phát triển, tách biệt framework (packages) và ứng dụng (app), giảm sự phụ thuộc giữa các dự án, tăng khả năng mở rộng.",
			"icon": "ph:cd",
			"color": "#34d293"
		},
		{
			"title": "Sứ mệnh",
			"des": "Sử dụng các công nghệ phổ biến như Vue3, Vite5, Nuxt UI, Pinia, Strapi5, MySQL... hoàn toàn miễn phí, không lo giới hạn framework, có thể dùng thương mại.",
			"icon": "ph:planet",
			"color": "#3ab0e2"
		},
		{
			"title": "Giá trị cốt lõi",
			"des": "1. Muốn phát triển website nhanh bằng framework, có kinh nghiệm frontend 1 năm+, 2. Thành thạo Vue.js, từng làm dự án thực tế, 3. Yêu thích công nghệ, ham học hỏi, muốn nâng cao trình độ.",
			"icon": "ph:smiley",
			"color": "#f7d861"
		}
	]`

	section1 := model.HomepageSection{
		Title:       "Chúng tôi chuyên",
		Description: "Chuyên phân tích đủi do và cung cấp các giải pháp về luật đất đai",
		TypeKey:     "TYPE01",
		Metadata:    []byte(type01Metadata),
		Position:    1,
		ShowHome:    true,
	}

	if err := DB.Create(&section1).Error; err != nil {
		log.Printf("⚠️  Warning: Failed to create TYPE01 section: %v", err)
	}

	// TYPE02: Chủ đề - lĩnh vực
	type02Metadata := `[
		{
			"title": "Khéo tay",
			"des": "Tôi tập trung vào lĩnh vực của mình, luôn có quan điểm riêng cho mọi việc, tự tay thực hiện để kiểm chứng có đúng như kỳ vọng không.",
			"icon": "ph:wrench",
			"color": "#3ab0e2"
		},
		{
			"title": "Hiểu kỹ thuật",
			"des": "Thành thạo JavaScript/Node, nắm vững framework Vue/Egg.js, sử dụng thành thạo các plugin hệ sinh thái Vue, hiểu nguyên lý webpack, gulp, nginx, có kiến thức tốt về cơ sở dữ liệu.",
			"icon": "logos:nodejs-icon",
			"color": "#e96656"
		},
		{
			"title": "Linh hoạt",
			"des": "Khi thấy thiết kế đẹp, tôi biết cách tạo ra template tốt, đồng thời phát hiện điểm chưa ổn của chương trình Bag để đề xuất cải tiến.",
			"icon": "ph:smiley-wink",
			"color": "#34d293"
		},
		{
			"title": "Hiếu kỳ",
			"des": "Là lập trình viên hiếu kỳ, tôi luôn hướng tới đổi mới công nghệ, thích suy nghĩ, nghiên cứu và giải quyết vấn đề.",
			"icon": "ph:planet",
			"color": "#409eff"
		},
		{
			"title": "Có thẩm mỹ",
			"des": "Biết phối màu, bố cục website hợp lý, sắp xếp module logic, sử dụng CSS3 và JavaScript để tạo hiệu ứng phản hồi người dùng đơn giản.",
			"icon": "ph:paint-brush-household",
			"color": "#ffca28"
		},
		{
			"title": "Kiên nhẫn",
			"des": "Làm ra một chủ đề tốt như người thợ mộc, cần tỉ mỉ, kiên nhẫn, tận tâm. Có những việc không phải thấy hy vọng mới kiên trì, mà kiên trì rồi mới thấy hy vọng.",
			"icon": "ph:bicycle",
			"color": "#4fc3f7"
		}
	]`

	section2 := model.HomepageSection{
		Title:       "Chủ đề - lĩnh vực",
		Description: "Luật sư hỗ trợ các lĩnh vực ngành nghề sau đây:",
		TypeKey:     "TYPE02",
		Metadata:    []byte(type02Metadata),
		Position:    2,
		ShowHome:    true,
	}

	if err := DB.Create(&section2).Error; err != nil {
		log.Printf("⚠️  Warning: Failed to create TYPE02 section: %v", err)
	}

	log.Println("✅ Default homepage sections created successfully")
	return nil
}
