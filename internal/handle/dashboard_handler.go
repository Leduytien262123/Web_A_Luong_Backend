//go:build ignore
// +build ignore

// Dashboard handler disabled; remove build tag to re-enable when dashboard domain models exist.
package handle

import (
	"backend/internal/helpers"
	"backend/internal/repo"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	repo *repo.DashboardRepo
}

func NewDashboardHandler() *DashboardHandler {
	return &DashboardHandler{
		repo: repo.NewDashboardRepo(),
	}
}

// GetFullOverview - API 1: Tổng quan dashboard (gộp 6 API)
// Trả về: Summary stats + Revenue chart + Order status + Top products + Top categories + Recent activities
func (h *DashboardHandler) GetFullOverview(c *gin.Context) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30) // Default 30 days

	if startStr := c.Query("start_date"); startStr != "" {
		if parsed, err := time.Parse("2006-01-02", startStr); err == nil {
			startDate = parsed
		}
	}

	if endStr := c.Query("end_date"); endStr != "" {
		if parsed, err := time.Parse("2006-01-02", endStr); err == nil {
			endDate = parsed
		}
	}

	period := c.DefaultQuery("period", "day")

	result, err := h.repo.GetFullOverview(startDate, endDate, period)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy dữ liệu dashboard", err)
		return
	}

	helpers.SuccessResponse(c, "Lấy dữ liệu dashboard thành công", result)
}

// GetAnalytics - API 2: Phân tích chi tiết (gộp 6 API)
// Trả về: Category stats + Product stats + Payment stats + Order type stats + Top customers + Revenue chart
func (h *DashboardHandler) GetAnalytics(c *gin.Context) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)

	if startStr := c.Query("start_date"); startStr != "" {
		if parsed, err := time.Parse("2006-01-02", startStr); err == nil {
			startDate = parsed
		}
	}

	if endStr := c.Query("end_date"); endStr != "" {
		if parsed, err := time.Parse("2006-01-02", endStr); err == nil {
			endDate = parsed
		}
	}

	period := c.DefaultQuery("period", "day")

	result, err := h.repo.GetAnalytics(startDate, endDate, period)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy dữ liệu phân tích", err)
		return
	}

	helpers.SuccessResponse(c, "Lấy dữ liệu phân tích thành công", result)
}

// GetAlerts - API 3: Cảnh báo & hoạt động (gộp 3 API)
// Trả về: Low stock products + Pending orders + New customers + Recent activities + Alert counts
func (h *DashboardHandler) GetAlerts(c *gin.Context) {
	result, err := h.repo.GetAlerts()
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy cảnh báo", err)
		return
	}

	helpers.SuccessResponse(c, "Lấy cảnh báo thành công", result)
}
