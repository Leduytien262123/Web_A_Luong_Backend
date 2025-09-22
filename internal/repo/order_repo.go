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

// Create tạo mới một đơn hàng với logic tự động tạo user
func (r *OrderRepo) Create(order *model.Order) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Nếu không có UserID nhưng có thông tin customer, tự động tạo user mới
		if order.UserID == nil && order.CustomerEmail != "" && order.CustomerPhone != "" {
			// Kiểm tra xem đã có user với email hoặc phone này chưa
			var existingUser model.User
			err := tx.Where("email = ? OR phone = ?", order.CustomerEmail, order.CustomerPhone).First(&existingUser).Error
			
			if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
				// Tạo user mới
				newUser := model.User{
					ID:       uuid.New(),
					Username: order.CustomerEmail, // Sử dụng email làm username
					FullName: order.CustomerName,
					Email:    order.CustomerEmail,
					Phone:    order.CustomerPhone,
					Password: "auto_generated", // Có thể hash password mặc định hoặc để trống
					Role:     "user",
					IsActive: true,
				}
				
				if err := tx.Create(&newUser).Error; err != nil {
					return err
				}
				
				// Gán UserID cho order
				order.UserID = &newUser.ID
				order.IsGuestOrder = false
			} else if err == nil {
				// User đã tồn tại, gán UserID
				order.UserID = &existingUser.ID
				order.IsGuestOrder = false
			} else {
				return err
			}
		}
		
		// Tạo order
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		
		// Cập nhật thống kê user nếu có UserID
		if order.UserID != nil {
			if err := r.updateUserOrderStats(tx, *order.UserID, order.FinalAmount, 1); err != nil {
				return err
			}
		}
		
		return nil
	})
}

// updateUserOrderStats cập nhật thống kê đơn hàng của user
func (r *OrderRepo) updateUserOrderStats(tx *gorm.DB, userID uuid.UUID, orderAmount float64, orderCount int) error {
	updates := map[string]interface{}{
		"total_orders": gorm.Expr("total_orders + ?", orderCount),
		"total_spent":  gorm.Expr("total_spent + ?", orderAmount),
		"last_order_at": time.Now(),
	}
	
	return tx.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error
}

// GetByID lấy đơn hàng theo ID kèm dữ liệu liên quan
func (r *OrderRepo) GetByID(id uuid.UUID) (*model.Order, error) {
	var order model.Order
	err := r.db.Preload("User").
		Preload("Creator").
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
		Preload("Creator").
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
	err := r.db.Preload("User").
		Preload("Creator").
		Preload("OrderItems").
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
		Preload("Creator").
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

// UpdateStatus cập nhật trạng thái đơn hàng và thống kê user
func (r *OrderRepo) UpdateStatus(id uuid.UUID, status string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Lấy thông tin order hiện tại
		var order model.Order
		if err := tx.Where("id = ?", id).First(&order).Error; err != nil {
			return err
		}
		
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
			// Cập nhật completed_orders khi đơn hàng được giao thành công
			if order.UserID != nil {
				if err := tx.Model(&model.User{}).Where("id = ?", order.UserID).
					Update("completed_orders", gorm.Expr("completed_orders + 1")).Error; err != nil {
					return err
				}
			}
		case "cancelled":
			updates["cancelled_at"] = &now
			// Trừ lại thống kê nếu đơn hàng bị hủy
			if order.UserID != nil {
				if err := r.updateUserOrderStats(tx, *order.UserID, -order.FinalAmount, -1); err != nil {
					return err
				}
			}
		}

		return tx.Model(&model.Order{}).Where("id = ?", id).Updates(updates).Error
	})
}

// UpdatePaymentStatus cập nhật trạng thái thanh toán
func (r *OrderRepo) UpdatePaymentStatus(id uuid.UUID, paymentStatus string) error {
	return r.db.Model(&model.Order{}).Where("id = ?", id).Update("payment_status", paymentStatus).Error
}

// GenerateOrderNumberForDate tạo mã đơn hàng theo định dạng WebShop-DDMMYYNNN
// Ví dụ: WebShop-180925001 (18/09/25 + sequence 001)
func (r *OrderRepo) GenerateOrderNumberForDate(t time.Time) (string, error) {
	datePart := t.Format("020106") // ddmmyy
	prefix := fmt.Sprintf("WebShop-%s", datePart)

	var lastOrder model.Order
	err := r.db.Where("order_number LIKE ?", prefix+"%").Order("order_number DESC").Limit(1).First(&lastOrder).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Không có đơn nào cho ngày hôm nay, bắt đầu từ 1
			return prefix + "001", nil
		}
		return "", err
	}

	// Lấy phần số cuối của mã đơn
	suffix := lastOrder.OrderNumber[len(prefix):]
	var lastSeq int
	if suffix != "" {
		_, err := fmt.Sscanf(suffix, "%d", &lastSeq)
		if err != nil {
			// Nếu parse lỗi, fallback start 1
			lastSeq = 0
		}
	}

	nextSeq := lastSeq + 1
	var seqStr string
	if nextSeq <= 999 {
		seqStr = fmt.Sprintf("%03d", nextSeq)
	} else {
		seqStr = fmt.Sprintf("%d", nextSeq) // cho phép >999
	}

	return prefix + seqStr, nil
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
	err := r.db.Preload("User").
		Preload("Creator").
		Preload("OrderItems").
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

// AdminUpdate cập nhật đơn hàng bởi admin
func (r *OrderRepo) AdminUpdate(id uuid.UUID, updates map[string]interface{}) error {
	return r.db.Model(&model.Order{}).Where("id = ?", id).Updates(updates).Error
}

// Delete xóa đơn hàng (soft delete)
func (r *OrderRepo) Delete(id uuid.UUID) error {
	return r.db.Delete(&model.Order{}, id).Error
}

// GetAllWithFilters lấy đơn hàng với bộ lọc cho admin
func (r *OrderRepo) GetAllWithFilters(page, limit int, status, paymentStatus, orderType string) ([]model.Order, int64, error) {
	var orders []model.Order
	var total int64

	query := r.db.Model(&model.Order{})

	// Áp dụng bộ lọc
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if paymentStatus != "" {
		query = query.Where("payment_status = ?", paymentStatus)
	}
	if orderType != "" {
		query = query.Where("order_type = ?", orderType)
	}

	// Đếm tổng số bản ghi
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Tính offset
	offset := (page - 1) * limit

	// Sử dụng lại query và thêm preload + phân trang
	q := query
	err := q.Preload("User").
		Preload("Creator").
		Preload("OrderItems").
		Preload("OrderItems.Product").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&orders).Error

	return orders, total, err
}