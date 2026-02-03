package consts

const (
	// JWT
	JWT_SECRET_KEY   = "your-secret-key-here"
	JWT_EXPIRE_HOURS = 24

	// Vai trò người dùng
	ROLE_ADMIN = "admin"
	ROLE_USER  = "user"

	// Thông báo phản hồi
	MSG_SUCCESS             = "Success"
	MSG_INVALID_CREDENTIALS = "Invalid credentials"
	MSG_USER_NOT_FOUND      = "User not found"
	MSG_EMAIL_EXISTS        = "Email already exists"
	MSG_USERNAME_EXISTS     = "Username already exists"
	MSG_UNAUTHORIZED        = "Unauthorized"
	MSG_FORBIDDEN           = "Forbidden"
	MSG_INTERNAL_ERROR      = "Internal server error"
	MSG_VALIDATION_ERROR    = "Validation error"
)

// Vai trò người dùng
const (
	RoleOwner  = "owner"  // Cấp độ cao nhất - chỉ một tài khoản
	RoleAdmin  = "admin"  // Có thể quản lý người dùng và phân quyền
	RoleMember = "member" // Cấp độ người dùng cơ bản
	RoleUser   = "user"   // Khách hàng/người dùng khách (để tương thích)
)

// Cấp độ quyền
const (
	PermissionFull  = "full"  // Tất cả quyền
	PermissionWrite = "write" // Tạo, cập nhật, đọc
	PermissionRead  = "read"  // Chỉ đọc
	PermissionNone  = "none"  // Không có quyền truy cập
)

// Ràng buộc hệ thống
const (
	MaxOwnerAccounts = 1       // Chỉ cho phép một tài khoản owner
	OwnerUsername    = "owner" // Tên người dùng owner cố định
)

// Phân cấp vai trò để kiểm tra quyền
var RoleHierarchy = map[string]int{
	RoleOwner:  4,
	RoleAdmin:  3,
	RoleMember: 2,
	RoleUser:   1,
}

// Kiểm tra xem role1 có quyền cao hơn hoặc bằng role2 không
func HasPermission(role1, role2 string) bool {
	level1, exists1 := RoleHierarchy[role1]
	level2, exists2 := RoleHierarchy[role2]

	if !exists1 || !exists2 {
		return false
	}

	return level1 >= level2
}

// Kiểm tra xem người dùng có thể quản lý vai trò đích không
func CanManageRole(userRole, targetRole string) bool {
	// Owner có thể quản lý tất cả trừ owner khác
	if userRole == RoleOwner && targetRole != RoleOwner {
		return true
	}

	// Admin có thể quản lý member và user, nhưng không phải owner hoặc admin khác
	if userRole == RoleAdmin && (targetRole == RoleMember || targetRole == RoleUser) {
		return true
	}

	return false
}
