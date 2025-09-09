# Reset Database Script - CHỈ SỬ DỤNG CHO DEVELOPMENT!
# Cách sử dụng: .\reset-database.ps1

param(
    [switch]$Confirm = $false
)

Write-Host "🚨 CẢNH BÁO: Script này sẽ XÓA TOÀN BỘ DATABASE!" -ForegroundColor Red
Write-Host "📁 Thư mục hiện tại: $(Get-Location)" -ForegroundColor Yellow

if (-not $Confirm) {
    $response = Read-Host "Bạn có chắc chắn muốn reset database? (yes/no)"
    if ($response -ne "yes") {
        Write-Host "❌ Hủy bỏ reset database" -ForegroundColor Green
        exit 0
    }
}

Write-Host "🔄 Bắt đầu reset database..." -ForegroundColor Yellow

# 1. Stop các process Go đang chạy
Write-Host "1️⃣ Dừng server đang chạy..." -ForegroundColor Cyan
Get-Process -Name "go" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 2

# 2. Backup file gốc
Write-Host "2️⃣ Backup file database.go..." -ForegroundColor Cyan
Copy-Item "app\database.go" "app\database.go.backup" -Force

# 3. Uncomment function ResetDatabase
Write-Host "3️⃣ Uncomment function ResetDatabase..." -ForegroundColor Cyan
$content = Get-Content "app\database.go" -Raw
$content = $content -replace '/\*\s*\n(func ResetDatabase.*?\n}\s*)\*/', '$1'
Set-Content "app\database.go" $content

# 4. Thêm reset call vào main.go
Write-Host "4️⃣ Thêm reset call vào main.go..." -ForegroundColor Cyan
Copy-Item "cmd\main.go" "cmd\main.go.backup" -Force
$mainContent = Get-Content "cmd\main.go" -Raw
$resetCall = @"
	// 🚨 RESET DATABASE - TEMPORARY
	if err := app.ResetDatabase(); err != nil {
		log.Fatal("Failed to reset database:", err)
	}

"@
$mainContent = $mainContent -replace '(// Connect to database and initialize)', "$resetCall`$1"
Set-Content "cmd\main.go" $mainContent

# 5. Chạy reset
Write-Host "5️⃣ Chạy database reset..." -ForegroundColor Cyan
go run .\cmd\main.go
if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Reset database thành công!" -ForegroundColor Green
} else {
    Write-Host "❌ Reset database thất bại!" -ForegroundColor Red
}

# 6. Restore files
Write-Host "6️⃣ Khôi phục files gốc..." -ForegroundColor Cyan
Move-Item "app\database.go.backup" "app\database.go" -Force
Move-Item "cmd\main.go.backup" "cmd\main.go" -Force

Write-Host "🎉 Hoàn thành! Database đã được reset và files đã được khôi phục." -ForegroundColor Green
Write-Host "💡 Bây giờ bạn có thể chạy server bình thường: go run .\cmd\main.go" -ForegroundColor Yellow
