package handle

import (
	"backend/internal/helpers"
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
)

type S3Handler struct{}

func NewS3Handler() *S3Handler {
	return &S3Handler{}
}

type PutObjectUpload struct {
	Key             string `json:"key" binding:"required"`
	ContentEncoding string `json:"content_encoding" binding:"required"`
	ContentType     string `json:"content_type" binding:"required"`
}

type S3Config struct {
	AccessKey  string
	SecretKey  string
	Endpoint   string
	BucketName string
	IsSSL      bool
	Region     string
}

// getS3Config lấy cấu hình S3 từ environment variables với validation
func getS3Config() S3Config {
	config := S3Config{
		AccessKey:  strings.TrimSpace(os.Getenv("S3_ACCESS_KEY")),
		SecretKey:  strings.TrimSpace(os.Getenv("S3_SECRET_KEY")),
		Endpoint:   strings.TrimSpace(os.Getenv("S3_ENDPOINT")),
		BucketName: strings.TrimSpace(os.Getenv("S3_BUCKET")),
		IsSSL:      os.Getenv("S3_SSL") == "true",
		Region:     strings.TrimSpace(os.Getenv("S3_REGION")),
	}

	// Set default region if not provided
	if config.Region == "" {
		config.Region = "us-east-1"
	}

	return config
}

// createMinioClient tạo kết nối minio client với cấu hình cải thiện
func createMinioClient(config S3Config) (*minio.Client, error) {
	// Validate required config
	if config.AccessKey == "" || config.SecretKey == "" || config.Endpoint == "" {
		return nil, fmt.Errorf("missing required S3 configuration: access_key, secret_key, or endpoint")
	}

	// Create MinIO client with proper options
	options := &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
		Secure: config.IsSSL,
		Region: config.Region,
	}

	client, err := minio.New(config.Endpoint, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return client, nil
}

// GetUploadUrl tạo presigned URL để upload file lên S3
// GetUploadUrl tạo presigned URL để upload file lên S3 + URL xem ảnh an toàn
func (h *S3Handler) GetUploadUrl(c *gin.Context) {
	var data PutObjectUpload
	if err := c.ShouldBindJSON(&data); err != nil {
		helpers.BadRequestResponse(c, "Dữ liệu đầu vào không hợp lệ")
		return
	}

	config := getS3Config()
	minioClient, err := createMinioClient(config)
	if err != nil {
		helpers.InternalErrorResponse(c, "Không thể kết nối tới storage service", err)
		return
	}

	// Kiểm tra bucket
	if err := ensureBucketExists(minioClient, config.BucketName); err != nil {
		helpers.InternalErrorResponse(c, "Lỗi storage service", err)
		return
	}

	// Tạo tên file
	uuidKey := uuid.New()
	folder := getFolder(data.ContentType)
	ext := getFileExtension(data.ContentType)
	objectKey := fmt.Sprintf("%s/%s/%s/%s%s",
		uuid.New().String(),
		folder,
		time.Now().Format("200601"),
		uuidKey.String(),
		ext,
	)

	// Thời gian hết hạn Upload
	expireUploadTime := time.Duration(1800) * time.Second // 30 phút

	// ✅ Tạo URL Upload (Presigned PUT)
	uploadURL, err := minioClient.PresignedPutObject(
		context.Background(),
		config.BucketName,
		objectKey,
		expireUploadTime,
	)
	if err != nil {
		helpers.InternalErrorResponse(c, "Không thể tạo URL upload", err)
		return
	}

	// ✅ Tạo URL xem ảnh an toàn (Presigned GET)
	expireViewTime := time.Duration(24) * time.Hour // Cho xem ảnh 24h
	viewURL, err := minioClient.PresignedGetObject(
		context.Background(),
		config.BucketName,
		objectKey,
		expireViewTime,
		nil,
	)
	if err != nil {
		helpers.InternalErrorResponse(c, "Không thể tạo URL xem file", err)
		return
	}

	// ✅ Direct URL (chỉ dùng nếu bucket đã public)
	directURL := fmt.Sprintf("https://%s/%s/%s", config.Endpoint, config.BucketName, objectKey)

	helpers.SuccessResponse(c, "Tạo URL upload thành công", gin.H{
		"upload_url": uploadURL.String(),
		"view_url":   viewURL.String(),  // <--- AN TOÀN, LUÔN DÙNG ĐƯỢC
		"direct_url": directURL,         // <--- Chỉ dùng nếu làm bucket PUBLIC
		"key":        objectKey,
	})
}


// DeleteS3Object xóa file từ S3
func (h *S3Handler) DeleteS3Object(c *gin.Context) {
	var input struct {
		FilePath string `json:"file_path" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.BadRequestResponse(c, "Đường dẫn file là bắt buộc")
		return
	}

	config := getS3Config()
	minioClient, err := createMinioClient(config)
	if err != nil {
		logrus.Error("Failed to create minio client: ", err)
		helpers.InternalErrorResponse(c, "Không thể kết nối tới storage service", err)
		return
	}

	// Trích xuất bucket và key từ URL hoặc path
	bucketName, objectName, err := parseS3Path(input.FilePath, config.BucketName)
	if err != nil {
		helpers.BadRequestResponse(c, "Đường dẫn file không hợp lệ")
		return
	}

	// Xóa object từ S3
	err = minioClient.RemoveObject(context.Background(), bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		logrus.Error("Failed to delete object from S3: ", err)
		helpers.InternalErrorResponse(c, "Không thể xóa file", err)
		return
	}

	helpers.SuccessResponse(c, "Xóa file thành công", nil)
}

// GetS3BucketMemoryUsage lấy thông tin sử dụng dung lượng
func (h *S3Handler) GetS3BucketMemoryUsage(c *gin.Context) {
	config := getS3Config()
	minioClient, err := createMinioClient(config)
	if err != nil {
		logrus.Error("Failed to create minio client: ", err)
		helpers.InternalErrorResponse(c, "Không thể kết nối tới storage service", err)
		return
	}

	var totalSize int64
	objectCh := minioClient.ListObjects(context.Background(), config.BucketName, minio.ListObjectsOptions{
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			logrus.Error("Error listing objects: ", object.Err)
			helpers.InternalErrorResponse(c, "Lỗi khi lấy thông tin storage", object.Err)
			return
		}
		totalSize += object.Size
	}

	// Chuyển đổi sang MB để dễ đọc
	totalSizeMB := float64(totalSize) / (1024 * 1024)

	helpers.SuccessResponse(c, "Lấy thông tin storage thành công", gin.H{
		"total_size_bytes": totalSize,
		"total_size_mb":    fmt.Sprintf("%.2f", totalSizeMB),
		"bucket_name":      config.BucketName,
	})
}

// getFolder xác định thư mục dựa trên content type
func getFolder(contentType string) string {
	if len(contentType) < 1 {
		return "other"
	}

	switch contentType {
	case "image/png", "image/jpeg", "image/jpg", "image/bmp", "image/gif", "image/webp", "image/svg+xml":
		return "images"
	case "application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-excel", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/pdf", "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"text/plain":
		return "documents"
	case "video/mp4", "video/mpeg", "video/quicktime", "video/x-ms-wmv",
		"audio/mpeg", "audio/wav", "audio/ogg":
		return "media"
	default:
		return "other"
	}
}

// getFileExtension lấy extension từ content type
func getFileExtension(contentType string) string {
	if len(contentType) < 1 {
		return ""
	}

	switch contentType {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/bmp":
		return ".bmp"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	case "application/msword":
		return ".doc"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return ".docx"
	case "application/vnd.ms-excel":
		return ".xls"
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return ".xlsx"
	case "application/pdf":
		return ".pdf"
	case "video/mp4":
		return ".mp4"
	case "video/mpeg":
		return ".mpeg"
	case "video/quicktime":
		return ".mov"
	case "video/x-ms-wmv":
		return ".wmv"
	case "audio/mpeg":
		return ".mp3"
	case "audio/wav":
		return ".wav"
	case "audio/ogg":
		return ".ogg"
	case "text/plain":
		return ".txt"
	case "application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return ".pptx"
	default:
		return ".bin"
	}
}

// ensureBucketExists kiểm tra và tạo bucket nếu chưa tồn tại với cấu hình region
func ensureBucketExists(minioClient *minio.Client, bucketName string) error {
	ctx := context.Background()

	exists, err := minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{
			Region: "us-east-1", // Đảm bảo region consistency
		})
		if err != nil {
			logrus.Error("Failed to create bucket: ", err)
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		logrus.Info("Created bucket: ", bucketName)
	}

	return nil
}

// parseS3Path trích xuất bucket và object name từ S3 path hoặc URL
func parseS3Path(filePath, defaultBucket string) (string, string, error) {
	// Nếu là URL đầy đủ
	if strings.HasPrefix(filePath, "http") {
		u, err := url.Parse(filePath)
		if err != nil {
			return "", "", err
		}

		parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
		if len(parts) < 2 {
			return "", "", fmt.Errorf("invalid S3 URL format")
		}

		bucketName := parts[0]
		objectName := strings.Join(parts[1:], "/")
		return bucketName, objectName, nil
	}

	// Nếu chỉ là object key
	return defaultBucket, strings.TrimPrefix(filePath, "/"), nil
}
