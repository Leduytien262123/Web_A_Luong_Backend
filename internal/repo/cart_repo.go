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

// GetCartByID lấy giỏ hàng theo ID
func (r *CartRepo) GetCartByID(cartID uuid.UUID) (*model.Cart, error) {
	var cart model.Cart
	err := r.db.Preload("CartItems").Preload("CartItems.Product").Where("id = ?", cartID).First(&cart).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("cart not found")
		}
		return nil, err
	}
	return &cart, nil
}

// GetCartTotal tính tổng giá trị giỏ hàng
func (r *CartRepo) GetCartTotal(userID uuid.UUID) (float64, error) {
	cart, err := r.GetOrCreateCart(userID)
	if err != nil {
		return 0, err
	}

	var total float64
	for _, item := range cart.CartItems {
		if item.Product != nil {
			total += item.Product.Price * float64(item.Quantity)
		}
	}

	return total, nil
}