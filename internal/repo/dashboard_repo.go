//go:build ignore
// +build ignore

// Dashboard repo disabled for article-first deployment; enable by removing build tag.
package repo

import (
	"backend/app"
	"backend/internal/model"
	"time"

	"gorm.io/gorm"
)

type DashboardRepo struct {
	db *gorm.DB
}

func NewDashboardRepo() *DashboardRepo {
	return &DashboardRepo{
		db: app.DB,
	}
}

// GetOverviewStats - Lấy thống kê tổng quan dashboard
func (r *DashboardRepo) GetOverviewStats(startDate, endDate time.Time) (*model.DashboardOverviewResponse, error) {
	var result model.DashboardOverviewResponse

	// Tính toán khoảng thời gian trước đó để so sánh
	duration := endDate.Sub(startDate)
	prevStartDate := startDate.Add(-duration)
	prevEndDate := startDate

	// Tổng doanh thu hiện tại
	var currentRevenue float64
	err := r.db.Model(&model.Order{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("status NOT IN (?)", []string{"cancelled"}).
		Select("COALESCE(SUM(final_amount), 0)").
		Scan(&currentRevenue).Error
	if err != nil {
		return nil, err
	}
	result.TotalRevenue = currentRevenue

	// Tổng doanh thu kỳ trước
	var prevRevenue float64
	r.db.Model(&model.Order{}).
		Where("created_at BETWEEN ? AND ?", prevStartDate, prevEndDate).
		Where("status NOT IN (?)", []string{"cancelled"}).
		Select("COALESCE(SUM(final_amount), 0)").
		Scan(&prevRevenue)

	// Tính % tăng trưởng doanh thu
	if prevRevenue > 0 {
		result.RevenueGrowth = ((currentRevenue - prevRevenue) / prevRevenue) * 100
	}

	// Tổng số đơn hàng hiện tại
	var currentOrders int64
	r.db.Model(&model.Order{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("status NOT IN (?)", []string{"cancelled"}).
		Count(&currentOrders)
	result.TotalOrders = int(currentOrders)

	// Tổng số đơn hàng kỳ trước
	var prevOrders int64
	r.db.Model(&model.Order{}).
		Where("created_at BETWEEN ? AND ?", prevStartDate, prevEndDate).
		Where("status NOT IN (?)", []string{"cancelled"}).
		Count(&prevOrders)

	// Tính % tăng trưởng đơn hàng
	if prevOrders > 0 {
		result.OrdersGrowth = ((float64(currentOrders) - float64(prevOrders)) / float64(prevOrders)) * 100
	}

	// Tổng số sản phẩm đã bán hiện tại
	var currentProductsSold int64
	r.db.Model(&model.OrderItem{}).
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Where("orders.created_at BETWEEN ? AND ?", startDate, endDate).
		Where("orders.status NOT IN (?)", []string{"cancelled"}).
		Select("COALESCE(SUM(order_items.quantity), 0)").
		Scan(&currentProductsSold)
	result.TotalProducts = int(currentProductsSold)

	// Tổng số sản phẩm đã bán kỳ trước
	var prevProductsSold int64
	r.db.Model(&model.OrderItem{}).
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Where("orders.created_at BETWEEN ? AND ?", prevStartDate, prevEndDate).
		Where("orders.status NOT IN (?)", []string{"cancelled"}).
		Select("COALESCE(SUM(order_items.quantity), 0)").
		Scan(&prevProductsSold)

	// Tính % tăng trưởng sản phẩm
	if prevProductsSold > 0 {
		result.ProductsGrowth = ((float64(currentProductsSold) - float64(prevProductsSold)) / float64(prevProductsSold)) * 100
	}

	// Tổng số khách hàng hiện tại
	var currentCustomers int64
	r.db.Model(&model.User{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("role IN (?)", []string{"user", "member"}).
		Count(&currentCustomers)
	result.TotalCustomers = int(currentCustomers)

	// Tổng số khách hàng kỳ trước
	var prevCustomers int64
	r.db.Model(&model.User{}).
		Where("created_at BETWEEN ? AND ?", prevStartDate, prevEndDate).
		Where("role IN (?)", []string{"user", "member"}).
		Count(&prevCustomers)

	// Tính % tăng trưởng khách hàng
	if prevCustomers > 0 {
		result.CustomersGrowth = ((float64(currentCustomers) - float64(prevCustomers)) / float64(prevCustomers)) * 100
	}

	// Đơn hàng đang chờ xử lý
	var pendingOrders int64
	r.db.Model(&model.Order{}).
		Where("status IN (?)", []string{"pending", "new", "confirmed"}).
		Count(&pendingOrders)
	result.PendingOrders = int(pendingOrders)

	// Sản phẩm sắp hết hàng
	var lowStockCount int64
	r.db.Model(&model.Product{}).
		Where("stock <= min_stock").
		Where("stock > 0").
		Count(&lowStockCount)
	result.LowStockProducts = int(lowStockCount)

	// Giá trị đơn hàng trung bình
	if result.TotalOrders > 0 {
		result.AverageOrderValue = result.TotalRevenue / float64(result.TotalOrders)
	}

	return &result, nil
}

// GetRevenueByTime - Lấy doanh thu theo thời gian (ngày/tuần/tháng/năm)
func (r *DashboardRepo) GetRevenueByTime(startDate, endDate time.Time, period string) ([]model.RevenueByTimeResponse, error) {
	var results []model.RevenueByTimeResponse

	var dateFormat string
	switch period {
	case "day":
		dateFormat = "%Y-%m-%d"
	case "week":
		dateFormat = "%Y-%u" // Year-Week
	case "month":
		dateFormat = "%Y-%m"
	case "year":
		dateFormat = "%Y"
	default:
		dateFormat = "%Y-%m-%d"
	}

	query := `
		SELECT 
			DATE_FORMAT(created_at, ?) as period,
			COALESCE(SUM(final_amount), 0) as revenue,
			COUNT(*) as orders,
			COALESCE(SUM((SELECT SUM(quantity) FROM order_items WHERE order_items.order_id = orders.id)), 0) as products_sold
		FROM orders
		WHERE created_at BETWEEN ? AND ?
			AND status NOT IN ('cancelled')
		GROUP BY period
		ORDER BY period ASC
	`

	err := r.db.Raw(query, dateFormat, startDate, endDate).Scan(&results).Error
	return results, err
}

// GetCategoryStatistics - Thống kê theo danh mục
func (r *DashboardRepo) GetCategoryStatistics(startDate, endDate time.Time, limit int) ([]model.CategoryStatistics, error) {
	var results []model.CategoryStatistics

	query := `
		SELECT 
			c.id as category_id,
			c.name as category_name,
			COALESCE(SUM(oi.quantity), 0) as products_sold,
			COALESCE(SUM(oi.total), 0) as revenue,
			COUNT(DISTINCT o.id) as orders
		FROM categories c
		LEFT JOIN products p ON p.category_id = c.id
		LEFT JOIN order_items oi ON oi.product_id = p.id
		LEFT JOIN orders o ON o.id = oi.order_id
		WHERE o.created_at BETWEEN ? AND ?
			AND o.status NOT IN ('cancelled')
		GROUP BY c.id, c.name
		ORDER BY revenue DESC
	`

	if limit > 0 {
		query += " LIMIT ?"
		err := r.db.Raw(query, startDate, endDate, limit).Scan(&results).Error
		if err != nil {
			return nil, err
		}
	} else {
		err := r.db.Raw(query, startDate, endDate).Scan(&results).Error
		if err != nil {
			return nil, err
		}
	}

	// Tính tổng doanh thu để tính phần trăm
	var totalRevenue float64
	for _, result := range results {
		totalRevenue += result.Revenue
	}

	// Tính phần trăm cho mỗi danh mục
	for i := range results {
		if totalRevenue > 0 {
			results[i].Percentage = (results[i].Revenue / totalRevenue) * 100
		}
	}

	return results, nil
}

// GetTopProducts - Lấy top sản phẩm bán chạy
func (r *DashboardRepo) GetTopProducts(startDate, endDate time.Time, limit int) ([]model.ProductStatistics, error) {
	var results []model.ProductStatistics

	query := `
		SELECT 
			p.id as product_id,
			p.name as product_name,
			p.sku,
			COALESCE(SUM(oi.quantity), 0) as quantity_sold,
			COALESCE(SUM(oi.total), 0) as revenue,
			p.stock,
			p.rating,
			p.review_count
		FROM products p
		LEFT JOIN order_items oi ON oi.product_id = p.id
		LEFT JOIN orders o ON o.id = oi.order_id
		WHERE o.created_at BETWEEN ? AND ?
			AND o.status NOT IN ('cancelled')
		GROUP BY p.id, p.name, p.sku, p.stock, p.rating, p.review_count
		ORDER BY quantity_sold DESC
		LIMIT ?
	`

	err := r.db.Raw(query, startDate, endDate, limit).Scan(&results).Error
	return results, err
}

// GetLowStockProducts - Lấy sản phẩm sắp hết hàng
func (r *DashboardRepo) GetLowStockProducts(limit int) ([]model.LowStockProduct, error) {
	var results []model.LowStockProduct

	err := r.db.Model(&model.Product{}).
		Select("id as product_id, name as product_name, sku, stock as current_stock, min_stock").
		Where("stock <= min_stock").
		Where("stock > 0").
		Order("stock ASC").
		Limit(limit).
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// Xác định status dựa trên mức độ thiếu hàng
	for i := range results {
		if results[i].CurrentStock == 0 {
			results[i].Status = "out_of_stock"
		} else if results[i].CurrentStock <= results[i].MinStock/2 {
			results[i].Status = "critical"
		} else {
			results[i].Status = "low"
		}
	}

	return results, nil
}

// GetOrderStatusStatistics - Thống kê trạng thái đơn hàng
func (r *DashboardRepo) GetOrderStatusStatistics(startDate, endDate time.Time) ([]model.OrderStatusStatistics, error) {
	var results []model.OrderStatusStatistics

	query := `
		SELECT 
			status,
			COUNT(*) as count
		FROM orders
		WHERE created_at BETWEEN ? AND ?
		GROUP BY status
		ORDER BY count DESC
	`

	err := r.db.Raw(query, startDate, endDate).Scan(&results).Error
	if err != nil {
		return nil, err
	}

	// Tính tổng để tính phần trăm
	var total int
	for _, result := range results {
		total += result.Count
	}

	// Tính phần trăm cho mỗi status
	for i := range results {
		if total > 0 {
			results[i].Percentage = (float64(results[i].Count) / float64(total)) * 100
		}
	}

	return results, nil
}

// GetPaymentMethodStatistics - Thống kê phương thức thanh toán
func (r *DashboardRepo) GetPaymentMethodStatistics(startDate, endDate time.Time) ([]model.PaymentMethodStatistics, error) {
	var results []model.PaymentMethodStatistics

	query := `
		SELECT 
			payment_method as method,
			COUNT(*) as count,
			COALESCE(SUM(final_amount), 0) as revenue
		FROM orders
		WHERE created_at BETWEEN ? AND ?
			AND status NOT IN ('cancelled')
		GROUP BY payment_method
		ORDER BY revenue DESC
	`

	err := r.db.Raw(query, startDate, endDate).Scan(&results).Error
	if err != nil {
		return nil, err
	}

	// Tính tổng để tính phần trăm
	var totalRevenue float64
	for _, result := range results {
		totalRevenue += result.Revenue
	}

	// Tính phần trăm cho mỗi phương thức
	for i := range results {
		if totalRevenue > 0 {
			results[i].Percentage = (results[i].Revenue / totalRevenue) * 100
		}
	}

	return results, nil
}

// GetOrderTypeStatistics - Thống kê loại đơn hàng
func (r *DashboardRepo) GetOrderTypeStatistics(startDate, endDate time.Time) ([]model.OrderTypeStatistics, error) {
	var results []model.OrderTypeStatistics

	query := `
		SELECT 
			COALESCE(order_type, 'online') as type,
			COUNT(*) as count,
			COALESCE(SUM(final_amount), 0) as revenue
		FROM orders
		WHERE created_at BETWEEN ? AND ?
			AND status NOT IN ('cancelled')
		GROUP BY order_type
		ORDER BY revenue DESC
	`

	err := r.db.Raw(query, startDate, endDate).Scan(&results).Error
	if err != nil {
		return nil, err
	}

	// Tính tổng để tính phần trăm
	var totalRevenue float64
	for _, result := range results {
		totalRevenue += result.Revenue
	}

	// Tính phần trăm cho mỗi loại
	for i := range results {
		if totalRevenue > 0 {
			results[i].Percentage = (results[i].Revenue / totalRevenue) * 100
		}
	}

	return results, nil
}

// GetTopCustomers - Lấy top khách hàng chi tiêu nhiều
func (r *DashboardRepo) GetTopCustomers(startDate, endDate time.Time, limit int) ([]model.CustomerStatistics, error) {
	var results []model.CustomerStatistics

	query := `
		SELECT 
			u.id as user_id,
			u.full_name,
			u.email,
			u.phone,
			COUNT(o.id) as total_orders,
			COALESCE(SUM(o.final_amount), 0) as total_spent,
			MAX(o.created_at) as last_order_date,
			COALESCE(AVG(o.final_amount), 0) as average_order_value
		FROM users u
		LEFT JOIN orders o ON o.user_id = u.id
		WHERE o.created_at BETWEEN ? AND ?
			AND o.status NOT IN ('cancelled')
		GROUP BY u.id, u.full_name, u.email, u.phone
		ORDER BY total_spent DESC
		LIMIT ?
	`

	err := r.db.Raw(query, startDate, endDate, limit).Scan(&results).Error
	return results, err
}

// GetNewCustomers - Lấy khách hàng mới
func (r *DashboardRepo) GetNewCustomers(startDate, endDate time.Time, limit int) ([]model.CustomerStatistics, error) {
	var results []model.CustomerStatistics

	query := `
		SELECT 
			u.id as user_id,
			u.full_name,
			u.email,
			u.phone,
			COUNT(o.id) as total_orders,
			COALESCE(SUM(o.final_amount), 0) as total_spent,
			MAX(o.created_at) as last_order_date,
			COALESCE(AVG(o.final_amount), 0) as average_order_value
		FROM users u
		LEFT JOIN orders o ON o.user_id = u.id
		WHERE u.created_at BETWEEN ? AND ?
			AND u.role IN ('user', 'member')
		GROUP BY u.id, u.full_name, u.email, u.phone
		ORDER BY u.created_at DESC
		LIMIT ?
	`

	err := r.db.Raw(query, startDate, endDate, limit).Scan(&results).Error
	return results, err
}

// GetRecentActivities - Lấy hoạt động gần đây
func (r *DashboardRepo) GetRecentActivities(limit int) ([]model.RecentActivity, error) {
	var results []model.RecentActivity

	// Lấy đơn hàng mới nhất
	var orders []struct {
		Type      string
		Desc      string
		Timestamp time.Time
		UserName  string
		Amount    float64
	}

	r.db.Raw(`
		SELECT 
			'order' as type,
			CONCAT('Đơn hàng #', order_code, ' - ', status) as desc,
			created_at as timestamp,
			name as user_name,
			final_amount as amount
		FROM orders
		ORDER BY created_at DESC
		LIMIT ?
	`, limit/2).Scan(&orders)

	for _, order := range orders {
		results = append(results, model.RecentActivity{
			Type:        order.Type,
			Description: order.Desc,
			Timestamp:   order.Timestamp.Format("2006-01-02 15:04:05"),
			UserName:    order.UserName,
			Amount:      order.Amount,
		})
	}

	// Lấy đánh giá mới nhất
	var reviews []struct {
		Type      string
		Desc      string
		Timestamp time.Time
		UserName  string
	}

	r.db.Raw(`
		SELECT 
			'review' as type,
			CONCAT('Đánh giá ', r.rating, ' sao cho sản phẩm: ', p.name) as desc,
			r.created_at as timestamp,
			u.full_name as user_name
		FROM reviews r
		JOIN products p ON p.id = r.product_id
		JOIN users u ON u.id = r.user_id
		ORDER BY r.created_at DESC
		LIMIT ?
	`, limit/2).Scan(&reviews)

	for _, review := range reviews {
		results = append(results, model.RecentActivity{
			Type:        review.Type,
			Description: review.Desc,
			Timestamp:   review.Timestamp.Format("2006-01-02 15:04:05"),
			UserName:    review.UserName,
		})
	}

	return results, nil
}

// GetFullOverview - API 1: Lấy tất cả dữ liệu overview (gộp 5 API)
func (r *DashboardRepo) GetFullOverview(startDate, endDate time.Time, period string) (*model.DashboardFullOverview, error) {
	result := &model.DashboardFullOverview{}

	// 1. Summary statistics
	summary, err := r.GetOverviewStats(startDate, endDate)
	if err != nil {
		return nil, err
	}
	result.Summary = *summary

	// 2. Revenue chart (7 days for quick load)
	revenueChart, err := r.GetRevenueByTime(startDate, endDate, period)
	if err != nil {
		return nil, err
	}
	result.RevenueChart = revenueChart

	// 3. Order status chart
	orderStatusChart, err := r.GetOrderStatusStatistics(startDate, endDate)
	if err != nil {
		return nil, err
	}
	result.OrderStatusChart = orderStatusChart

	// 4. Top 5 products
	topProducts, err := r.GetTopProducts(startDate, endDate, 5)
	if err != nil {
		return nil, err
	}
	result.TopProducts = topProducts

	// 5. Top 5 categories
	topCategories, err := r.GetCategoryStatistics(startDate, endDate, 5)
	if err != nil {
		return nil, err
	}
	result.TopCategories = topCategories

	// 6. Recent 10 activities
	recentActivities, err := r.GetRecentActivities(10)
	if err != nil {
		return nil, err
	}
	result.RecentActivities = recentActivities

	return result, nil
}

// GetAnalytics - API 2: Lấy dữ liệu phân tích chi tiết (gộp 4 API)
func (r *DashboardRepo) GetAnalytics(startDate, endDate time.Time, period string) (*model.DashboardAnalytics, error) {
	result := &model.DashboardAnalytics{}

	// 1. All category stats
	categoryStats, err := r.GetCategoryStatistics(startDate, endDate, 0)
	if err != nil {
		return nil, err
	}
	result.CategoryStats = categoryStats

	// 2. Top 20 products
	productStats, err := r.GetTopProducts(startDate, endDate, 20)
	if err != nil {
		return nil, err
	}
	result.ProductStats = productStats

	// 3. Payment method statistics
	paymentStats, err := r.GetPaymentMethodStatistics(startDate, endDate)
	if err != nil {
		return nil, err
	}
	result.PaymentMethodStats = paymentStats

	// 4. Order type statistics
	orderTypeStats, err := r.GetOrderTypeStatistics(startDate, endDate)
	if err != nil {
		return nil, err
	}
	result.OrderTypeStats = orderTypeStats

	// 5. Top 10 customers
	topCustomers, err := r.GetTopCustomers(startDate, endDate, 10)
	if err != nil {
		return nil, err
	}
	result.TopCustomers = topCustomers

	// 6. Revenue by time (for detailed chart)
	revenueByTime, err := r.GetRevenueByTime(startDate, endDate, period)
	if err != nil {
		return nil, err
	}
	result.RevenueByTime = revenueByTime

	return result, nil
}

// GetAlerts - API 3: Lấy cảnh báo và hoạt động (gộp 2 API)
func (r *DashboardRepo) GetAlerts() (*model.DashboardAlerts, error) {
	result := &model.DashboardAlerts{}

	// 1. Low stock products (top 20)
	lowStockProducts, err := r.GetLowStockProducts(20)
	if err != nil {
		return nil, err
	}
	result.LowStockProducts = lowStockProducts

	// 2. Count critical and warning alerts
	criticalCount := 0
	warningCount := 0
	for _, product := range lowStockProducts {
		if product.Status == "critical" || product.Status == "out_of_stock" {
			criticalCount++
		} else if product.Status == "low" {
			warningCount++
		}
	}
	result.CriticalAlerts = criticalCount
	result.WarningAlerts = warningCount

	// 3. Pending orders count
	var pendingOrders int64
	r.db.Model(&model.Order{}).
		Where("status IN (?)", []string{"pending", "new", "confirmed"}).
		Count(&pendingOrders)
	result.PendingOrders = int(pendingOrders)

	// 4. New customers (last 7 days)
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -7)
	newCustomers, err := r.GetNewCustomers(startDate, endDate, 10)
	if err != nil {
		return nil, err
	}
	result.NewCustomers = newCustomers

	// 5. Recent activities (20 items)
	recentActivities, err := r.GetRecentActivities(20)
	if err != nil {
		return nil, err
	}
	result.RecentActivities = recentActivities

	return result, nil
}
