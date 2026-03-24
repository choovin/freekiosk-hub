package models

// AppPackage 应用包模型
type AppPackage struct {
	ID            string   `json:"id" db:"id"`
	Name         string   `json:"name" db:"name"`
	PackageName  string   `json:"package_name" db:"package_name"` // Android包名
	Version      string   `json:"version" db:"version"`
	VersionCode  int      `json:"version_code" db:"version_code"`
	FileSize     int64    `json:"file_size" db:"file_size"`
	FilePath     string   `json:"file_path" db:"file_path"`
	SHA256       string   `json:"sha256" db:"sha256"`
	Description  string   `json:"description" db:"description"`
	TenantID     string   `json:"tenant_id" db:"tenant_id"`
	UploadedBy   string   `json:"uploaded_by" db:"uploaded_by"`
	UploadURL    string   `json:"upload_url" db:"upload_url"` // 供设备下载的URL
	InstallCount int      `json:"install_count" db:"install_count"` // 安装次数
	CreatedAt    int64    `json:"created_at" db:"created_at"`
	UpdatedAt    int64    `json:"updated_at" db:"updated_at"`
}

// AppInstall 应用的安装记录
type AppInstall struct {
	ID          string `json:"id" db:"id"`
	PackageID   string `json:"package_id" db:"package_id"`
	DeviceID    string `json:"device_id" db:"device_id"`
	TenantID    string `json:"tenant_id" db:"tenant_id"`
	Version     string `json:"version" db:"version"`
	Status      string `json:"status" db:"status"` // pending/installing/installed/failed/uninstalled
	ErrorMsg    string `json:"error_msg" db:"error_msg"`
	InstalledAt *int64 `json:"installed_at" db:"installed_at"`
	CreatedAt   int64  `json:"created_at" db:"created_at"`
	UpdatedAt   int64  `json:"updated_at" db:"updated_at"`
}

// InstallStatus 安装状态枚举
type InstallStatus string

const (
	InstallStatusPending    InstallStatus = "pending"     // 等待安装
	InstallStatusInstalling InstallStatus = "installing" // 安装中
	InstallStatusInstalled InstallStatus = "installed"  // 已安装
	InstallStatusFailed    InstallStatus = "failed"     // 安装失败
	InstallStatusUninstalled InstallStatus = "uninstalled" // 已卸载
)
