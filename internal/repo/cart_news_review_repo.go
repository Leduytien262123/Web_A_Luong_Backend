package repo

import (
	"backend/app"
	"backend/internal/model"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CartRepo struct {
	db *gorm.DB
}

func NewCartRepo() *CartRepo {
	return &CartRepo{
		db: app.GetDB(),
	}
}

// GetOrCreateCart lấy hoặc tạo giỏ hàng cho người dùng
func (r *CartRepo) GetOrCreateCart(userID uuid.UUID) (*model.Cart, error) {
	var cart model.Cart
	err := r.db.Where("user_id = ?", userID).First(&cart).Error
	
	if errors.Is(err, gorm.ErrRecordNotFound) {
		 // Tạo giỏ hàng mới nếu chưa tồn tại
		cart = model.Cart{UserID: userID}
		if err := r.db.Create(&cart).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	// Nạp các mục giỏ hàng kèm thông tin sản phẩm
	if err := r.db.Preload("CartItems").Preload("CartItems.Product").Where("id = ?", cart.ID).First(&cart).Error; err != nil {
		return nil, err
	}

	return &cart, nil
}

// AddItem thêm mới hoặc cập nhật một mục trong giỏ hàng
func (r *CartRepo) AddItem(userID, productID uuid.UUID, quantity int) error {
	cart, err := r.GetOrCreateCart(userID)
	if err != nil {
		return err
	}

	// Kiểm tra xem mục đã tồn tại chưa
	var cartItem model.CartItem
	err = r.db.Where("cart_id = ? AND product_id = ?", cart.ID, productID).First(&cartItem).Error
	
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Tạo mục mới
		cartItem = model.CartItem{
			CartID:    cart.ID,
			ProductID: productID,
			Quantity:  quantity,
		}
		return r.db.Create(&cartItem).Error
	} else if err != nil {
		return err
	} else {
		// Cập nhật mục đã tồn tại
		cartItem.Quantity += quantity
		return r.db.Save(&cartItem).Error
	}
}

// UpdateItemQuantity cập nhật số lượng của mục trong giỏ
func (r *CartRepo) UpdateItemQuantity(userID, productID uuid.UUID, quantity int) error {
	cart, err := r.GetOrCreateCart(userID)
	if err != nil {
		return err
	}

	if quantity <= 0 {
		return r.RemoveItem(userID, productID)
	}

	return r.db.Model(&model.CartItem{}).
		Where("cart_id = ? AND product_id = ?", cart.ID, productID).
		Update("quantity", quantity).Error
}

// RemoveItem xóa một mục khỏi giỏ hàng
func (r *CartRepo) RemoveItem(userID, productID uuid.UUID) error {
	cart, err := r.GetOrCreateCart(userID)
	if err != nil {
		return err
	}

	return r.db.Where("cart_id = ? AND product_id = ?", cart.ID, productID).Delete(&model.CartItem{}).Error
}

// ClearCart xóa toàn bộ mục trong giỏ hàng
func (r *CartRepo) ClearCart(userID uuid.UUID) error {
	cart, err := r.GetOrCreateCart(userID)
	if err != nil {
		return err
	}

	return r.db.Where("cart_id = ?", cart.ID).Delete(&model.CartItem{}).Error
}

type NewsRepo struct {
	db *gorm.DB
}

func NewNewsRepo() *NewsRepo {
	return &NewsRepo{
		db: app.GetDB(),
	}
}

// Create tạo mới một bài viết tin tức
func (r *NewsRepo) Create(news *model.News) error {
	return r.db.Create(news).Error
}

// GetByID lấy bài viết tin tức theo ID
func (r *NewsRepo) GetByID(id uuid.UUID) (*model.News, error) {
	var news model.News
	err := r.db.Preload("Author").Where("id = ?", id).First(&news).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("news not found")
		}
		return nil, err
	}
	return &news, nil
}

// GetBySlug lấy bài viết tin tức theo slug
func (r *NewsRepo) GetBySlug(slug string) (*model.News, error) {
	var news model.News
	err := r.db.Preload("Author").Where("slug = ?", slug).First(&news).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("news not found")
		}
		return nil, err
	}
	return &news, nil
}

// GetAll lấy danh sách tin tức có phân trang
func (r *NewsRepo) GetAll(page, limit int, publishedOnly bool) ([]model.News, int64, error) {
	var news []model.News
	var total int64

	query := r.db.Model(&model.News{})
	if publishedOnly {
		query = query.Where("is_published = ?", true)
	}

	// Đếm tổng số bản ghi
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Tính toán offset
	offset := (page - 1) * limit

	// Lấy tin tức kèm thông tin tác giả
	err := query.Preload("Author").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&news).Error

	return news, total, err
}

// Update cập nhật một bài viết tin tức
func (r *NewsRepo) Update(news *model.News) error {
	return r.db.Save(news).Error
}

// Delete xóa mềm một bài viết tin tức
func (r *NewsRepo) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&model.News{}).Error
}

// IncrementViewCount tăng số lượt xem
func (r *NewsRepo) IncrementViewCount(id uuid.UUID) error {
	return r.db.Model(&model.News{}).Where("id = ?", id).UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).Error
}

// CheckSlugExists kiểm tra slug đã tồn tại hay chưa
func (r *NewsRepo) CheckSlugExists(slug string, excludeID uuid.UUID) (bool, error) {
	var count int64
	query := r.db.Model(&model.News{}).Where("slug = ?", slug)
	if excludeID != uuid.Nil {
		query = query.Where("id != ?", excludeID)
	}
	err := query.Count(&count).Error
	return count > 0, err
}

type ReviewRepo struct {
	db *gorm.DB
}

func NewReviewRepo() *ReviewRepo {
	return &ReviewRepo{
		db: app.GetDB(),
	}
}

// Create tạo mới một đánh giá
func (r *ReviewRepo) Create(review *model.Review) error {
	return r.db.Create(review).Error
}

// GetByProductID lấy danh sách đánh giá theo ID sản phẩm
func (r *ReviewRepo) GetByProductID(productID uuid.UUID, page, limit int) ([]model.Review, int64, error) {
	var reviews []model.Review
	var total int64

	// Đếm tổng số bản ghi
	if err := r.db.Model(&model.Review{}).Where("product_id = ? AND is_active = ?", productID, true).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Tính toán offset
	offset := (page - 1) * limit

	// Lấy đánh giá kèm thông tin người dùng
	err := r.db.Preload("User").
		Where("product_id = ? AND is_active = ?", productID, true).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&reviews).Error

	return reviews, total, err
}

// GetByUserID lấy danh sách đánh giá theo ID người dùng
func (r *ReviewRepo) GetByUserID(userID uuid.UUID, page, limit int) ([]model.Review, int64, error) {
	var reviews []model.Review
	var total int64

	// Đếm tổng số bản ghi
	if err := r.db.Model(&model.Review{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Tính toán offset
	offset := (page - 1) * limit

	// Lấy đánh giá kèm thông tin sản phẩm
	err := r.db.Preload("Product").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&reviews).Error

	return reviews, total, err
}

// CheckUserReviewExists kiểm tra người dùng đã đánh giá sản phẩm này chưa
func (r *ReviewRepo) CheckUserReviewExists(userID, productID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&model.Review{}).Where("user_id = ? AND product_id = ?", userID, productID).Count(&count).Error
	return count > 0, err
}

// Update cập nhật một đánh giá
func (r *ReviewRepo) Update(review *model.Review) error {
	return r.db.Save(review).Error
}

// Delete xóa mềm một đánh giá
func (r *ReviewRepo) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&model.Review{}).Error
}

// ToggleStatus đảo trạng thái kích hoạt của đánh giá
func (r *ReviewRepo) ToggleStatus(id uuid.UUID) error {
	return r.db.Model(&model.Review{}).Where("id = ?", id).Update("is_active", gorm.Expr("NOT is_active")).Error
}