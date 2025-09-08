package repo

import (
	"backend/app"
	"backend/internal/model"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderRepo struct {
	db *gorm.DB
}

func NewOrderRepo() *OrderRepo {
	return &OrderRepo{
		db: app.GetDB(),
	}
}

// Create tạo mới một đơn hàng
func (r *OrderRepo) Create(order *model.Order) error {
	return r.db.Create(order).Error
}

// GetByID lấy đơn hàng theo ID kèm dữ liệu liên quan
func (r *OrderRepo) GetByID(id uuid.UUID) (*model.Order, error) {
	var order model.Order
	err := r.db.Preload("User").
		Preload("OrderItems").
		Preload("OrderItems.Product").
		Where("id = ?", id).First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("order not found")
		}
		return nil, err
	}
	return &order, nil
}

// GetByOrderNumber lấy đơn hàng theo mã đơn
func (r *OrderRepo) GetByOrderNumber(orderNumber string) (*model.Order, error) {
	var order model.Order
	err := r.db.Preload("User").
		Preload("OrderItems").
		Preload("OrderItems.Product").
		Where("order_number = ?", orderNumber).
		First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("order not found")
		}
		return nil, err
	}
	return &order, nil
}

// GetByUserID lấy danh sách đơn hàng theo userID có phân trang
func (r *OrderRepo) GetByUserID(userID uuid.UUID, page, limit int) ([]model.Order, int64, error) {
	var orders []model.Order
	var total int64

	// Đếm tổng số bản ghi
	if err := r.db.Model(&model.Order{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Tính offset
	offset := (page - 1) * limit

	// Lấy đơn hàng kèm dữ liệu liên quan
	err := r.db.Preload("OrderItems").
		Preload("OrderItems.Product").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&orders).Error

	return orders, total, err
}

// GetAll lấy tất cả đơn hàng có phân trang
func (r *OrderRepo) GetAll(page, limit int) ([]model.Order, int64, error) {
	var orders []model.Order
	var total int64

	// Đếm tổng số bản ghi
	if err := r.db.Model(&model.Order{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Tính offset
	offset := (page - 1) * limit

	// Lấy đơn hàng kèm dữ liệu liên quan
	err := r.db.Preload("User").
		Preload("OrderItems").
		Preload("OrderItems.Product").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&orders).Error

	return orders, total, err
}

// Update cập nhật đơn hàng
func (r *OrderRepo) Update(order *model.Order) error {
	return r.db.Save(order).Error
}

// UpdateStatus cập nhật trạng thái đơn hàng
func (r *OrderRepo) UpdateStatus(id uuid.UUID, status string) error {
	updates := map[string]interface{}{
		"status": status,
	}

	// Ghi nhận thời gian cho một số trạng thái cụ thể
	now := time.Now()
	switch status {
	case "shipped":
		updates["shipped_at"] = &now
	case "delivered":
		updates["delivered_at"] = &now
	}

	return r.db.Model(&model.Order{}).Where("id = ?", id).Updates(updates).Error
}

// UpdatePaymentStatus cập nhật trạng thái thanh toán
func (r *OrderRepo) UpdatePaymentStatus(id uuid.UUID, paymentStatus string) error {
	return r.db.Model(&model.Order{}).Where("id = ?", id).Update("payment_status", paymentStatus).Error
}

// GenerateOrderNumber tạo mã đơn hàng duy nhất
func (r *OrderRepo) GenerateOrderNumber() string {
	timestamp := time.Now().Format("20060102150405")
	return fmt.Sprintf("ORD-%s", timestamp)
}

// CheckOrderNumberExists kiểm tra mã đơn đã tồn tại chưa
func (r *OrderRepo) CheckOrderNumberExists(orderNumber string) (bool, error) {
	var count int64
	err := r.db.Model(&model.Order{}).Where("order_number = ?", orderNumber).Count(&count).Error
	return count > 0, err
}

// GetOrderStats lấy thống kê đơn hàng
func (r *OrderRepo) GetOrderStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Tổng số đơn hàng
	var totalOrders int64
	if err := r.db.Model(&model.Order{}).Count(&totalOrders).Error; err != nil {
		return nil, err
	}
	stats["total_orders"] = totalOrders

	// Số lượng đơn theo trạng thái
	var statusCounts []struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	if err := r.db.Model(&model.Order{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Find(&statusCounts).Error; err != nil {
		return nil, err
	}
	stats["orders_by_status"] = statusCounts

	// Tổng doanh thu
	var totalRevenue float64
	if err := r.db.Model(&model.Order{}).
		Where("payment_status = ?", "paid").
		Select("SUM(final_amount)").
		Scan(&totalRevenue).Error; err != nil {
		return nil, err
	}
	stats["total_revenue"] = totalRevenue

	return stats, nil
}

// GetByEmailOrPhone lấy đơn hàng theo email hoặc số điện thoại khách
func (r *OrderRepo) GetByEmailOrPhone(emailOrPhone string, page, limit int) ([]model.Order, int64, error) {
	var orders []model.Order
	var total int64

	// Xây dựng truy vấn tìm theo email hoặc số điện thoại
	query := r.db.Model(&model.Order{}).Where("customer_email = ? OR customer_phone = ?", emailOrPhone, emailOrPhone)

	// Đếm tổng số bản ghi
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Tính offset
	offset := (page - 1) * limit

	// Lấy đơn hàng kèm dữ liệu liên quan
	err := r.db.Preload("OrderItems").
		Preload("OrderItems.Product").
		Where("customer_email = ? OR customer_phone = ?", emailOrPhone, emailOrPhone).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&orders).Error

	return orders, total, err
}

// GetGuestOrderStats lấy thống kê đơn của khách vãng lai
func (r *OrderRepo) GetGuestOrderStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Tổng số đơn của khách vãng lai
	var totalGuestOrders int64
	if err := r.db.Model(&model.Order{}).Where("is_guest_order = ?", true).Count(&totalGuestOrders).Error; err != nil {
		return nil, err
	}
	stats["total_guest_orders"] = totalGuestOrders

	// Tổng số đơn của người dùng đã đăng ký
	var totalUserOrders int64
	if err := r.db.Model(&model.Order{}).Where("is_guest_order = ?", false).Count(&totalUserOrders).Error; err != nil {
		return nil, err
	}
	stats["total_user_orders"] = totalUserOrders

	// Doanh thu từ đơn của khách vãng lai
	var guestRevenue float64
	if err := r.db.Model(&model.Order{}).
		Where("is_guest_order = ? AND payment_status = ?", true, "paid").
		Select("SUM(final_amount)").
		Scan(&guestRevenue).Error; err != nil {
		return nil, err
	}
	stats["guest_revenue"] = guestRevenue

	return stats, nil
}