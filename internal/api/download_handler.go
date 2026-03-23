package api

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/skip2/go-qrcode"
)

// DownloadHandler handles the APK download page
type DownloadHandler struct {
	APKDir     string
	ServerPort string
}

// NewDownloadHandler creates a new DownloadHandler
func NewDownloadHandler(apkDir string, serverPort string) *DownloadHandler {
	return &DownloadHandler{
		APKDir:     apkDir,
		ServerPort: serverPort,
	}
}

// APKInfo holds information about the latest APK
type APKInfo struct {
	Filename    string
	Version     string
	URL        string
	Size       int64
	DownloadURL string // Full URL with host:port
}

// getLatestAPK finds the most recently modified APK file
func (h *DownloadHandler) getLatestAPK() (*APKInfo, error) {
	// Ensure APK directory exists
	if _, err := os.Stat(h.APKDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("APK directory does not exist: %s", h.APKDir)
	}

	entries, err := os.ReadDir(h.APKDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read APK directory: %w", err)
	}

	var latestFile os.FileInfo

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".apk") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if latestFile == nil || info.ModTime().After(latestFile.ModTime()) {
			latestFile = info
		}
	}

	if latestFile == nil {
		return nil, fmt.Errorf("no APK files found in %s", h.APKDir)
	}

	filename := latestFile.Name()
	version := extractVersion(filename)
	if version == "" {
		version = "Unknown"
	}

	downloadURL := fmt.Sprintf("http://localhost:%s/apk/%s", h.ServerPort, filename)

	return &APKInfo{
		Filename:    filename,
		Version:    version,
		URL:        fmt.Sprintf("/apk/%s", filename),
		Size:       latestFile.Size(),
		DownloadURL: downloadURL,
	}, nil
}

// generateQRCode generates a QR code as base64-encoded PNG
func generateQRCode(content string) (string, error) {
	png, err := qrcode.Encode(content, qrcode.Medium, 256)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(png), nil
}

// HandleDownloadPage renders the APK download page
func (h *DownloadHandler) HandleDownloadPage(c echo.Context) error {
	apkInfo, err := h.getLatestAPK()
	if err != nil {
		// Return a page indicating no APK is available
		noAPKHTML := `<!DOCTYPE html>
<html lang="zh">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>下载 FreeKiosk 研学版</title>
	<style>
		* { margin: 0; padding: 0; box-sizing: border-box; }
		body {
			min-height: 100vh;
			background: linear-gradient(135deg, #0f0f23 0%%, #1a1a2e 100%%);
			display: flex;
			align-items: center;
			justify-content: center;
			font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
		}
		.card {
			background: rgba(255, 255, 255, 0.1);
			backdrop-filter: blur(20px);
			border: 1px solid rgba(255, 255, 255, 0.2);
			border-radius: 24px;
			padding: 48px;
			text-align: center;
			color: white;
			max-width: 400px;
			width: 90%%;
		}
		h1 { font-size: 28px; margin-bottom: 16px; font-weight: 700; }
		p { color: rgba(255, 255, 255, 0.7); font-size: 16px; }
		.error { color: #ff6b6b; }
	</style>
</head>
<body>
	<div class="card">
		<h1>下载 FreeKiosk 研学版</h1>
		<p class="error">暂无 APK 文件</p>
		<p>请先上传 APK 文件后再访问此页面</p>
	</div>
</body>
</html>`
		return c.String(http.StatusOK, noAPKHTML)
	}

	// Generate QR code
	qrCodeB64, err := generateQRCode(apkInfo.DownloadURL)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to generate QR code")
	}

	qrCodeDataURL := fmt.Sprintf("data:image/png;base64,%s", qrCodeB64)

	// Format file size
	sizeStr := formatFileSize(apkInfo.Size)

	// Build download page with QR code and APK info
	downloadPageHTML := fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>下载 FreeKiosk 研学版</title>
	<style>
		* { margin: 0; padding: 0; box-sizing: border-box; }
		body {
			min-height: 100vh;
			background: linear-gradient(135deg, #0f0f23 0%%, #1a1a2e 100%%);
			display: flex;
			align-items: center;
			justify-content: center;
			font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
		}
		.card {
			background: rgba(255, 255, 255, 0.1);
			backdrop-filter: blur(20px);
			border: 1px solid rgba(255, 255, 255, 0.2);
			border-radius: 24px;
			padding: 48px;
			text-align: center;
			color: white;
			max-width: 420px;
			width: 90%%;
		}
		h1 { font-size: 28px; margin-bottom: 8px; font-weight: 700; }
		.subtitle { color: rgba(255, 255, 255, 0.6); font-size: 14px; margin-bottom: 32px; }
		.app-icon { font-size: 64px; margin-bottom: 16px; line-height: 1; }
		.app-name { font-size: 13px; color: rgba(255,255,255,0.5); letter-spacing: 2px; text-transform: uppercase; margin-bottom: 24px; }
		.apk-btn {
			display: inline-flex;
			align-items: center;
			gap: 8px;
			background: #6366f1;
			color: white;
			font-weight: 600;
			font-size: 15px;
			padding: 12px 24px;
			border-radius: 12px;
			text-decoration: none;
			margin-bottom: 20px;
			transition: background 0.2s, transform 0.2s;
		}
		.apk-btn:hover { background: #4f46e5; transform: scale(1.02); }
		.divider {
			color: rgba(255,255,255,0.3);
			font-size: 12px;
			margin-bottom: 20px;
			display: flex;
			align-items: center;
			gap: 12px;
		}
		.divider::before, .divider::after {
			content: '';
			flex: 1;
			height: 1px;
			background: rgba(255,255,255,0.2);
		}
		.qr-container {
			background: white;
			border-radius: 16px;
			padding: 16px;
			display: inline-block;
			margin-bottom: 24px;
		}
		.qr-container img {
			display: block;
			width: 200px;
			height: 200px;
		}
		.version {
			display: inline-block;
			background: rgba(255, 255, 255, 0.15);
			border-radius: 8px;
			padding: 8px 16px;
			font-size: 14px;
			margin-bottom: 24px;
		}
		.version span { color: rgba(255, 255, 255, 0.5); margin-right: 8px; }
		.version strong { color: #4ade80; }
		.url-container {
			background: rgba(0, 0, 0, 0.3);
			border-radius: 12px;
			padding: 12px 16px;
			margin-bottom: 16px;
			display: flex;
			align-items: center;
			gap: 12px;
		}
		.url-text {
			flex: 1;
			font-size: 12px;
			color: rgba(255, 255, 255, 0.8);
			word-break: break-all;
			text-align: left;
		}
		.copy-btn {
			background: #4ade80;
			color: #0f0f23;
			border: none;
			border-radius: 8px;
			padding: 8px 16px;
			font-size: 13px;
			font-weight: 600;
			cursor: pointer;
			transition: all 0.2s;
			white-space: nowrap;
		}
		.copy-btn:hover { background: #22c55e; transform: scale(1.05); }
		.copy-btn:active { transform: scale(0.98); }
		.size { color: rgba(255, 255, 255, 0.5); font-size: 12px; margin-top: 16px; }
		.success-msg {
			color: #4ade80;
			font-size: 13px;
			margin-top: 8px;
			opacity: 0;
			transition: opacity 0.3s;
		}
		.success-msg.show { opacity: 1; }
		.instructions {
			margin-top: 24px;
			background: rgba(255,255,255,0.05);
			border: 1px solid rgba(255,255,255,0.1);
			border-radius: 12px;
			padding: 16px 20px;
			text-align: left;
		}
		.instructions-title {
			font-size: 12px;
			color: rgba(255,255,255,0.4);
			margin-bottom: 12px;
			text-transform: uppercase;
			letter-spacing: 1px;
		}
		.step {
			display: flex;
			align-items: center;
			gap: 12px;
			font-size: 13px;
			color: rgba(255,255,255,0.8);
			margin-bottom: 8px;
		}
		.step:last-child { margin-bottom: 0; }
		.step-num {
			width: 22px;
			height: 22px;
			background: rgba(99,102,241,0.3);
			border: 1px solid rgba(99,102,241,0.5);
			border-radius: 50%%;
			display: flex;
			align-items: center;
			justify-content: center;
			font-size: 11px;
			font-weight: 700;
			color: #a5b4fc;
			flex-shrink: 0;
		}
		.step-text strong { color: white; }
</head>
<body>
	<div class="card">
		<div class="app-icon">📱</div>
		<p class="app-name">FreeKiosk / 研学版</p>
		<h1>下载 FreeKiosk 研学版</h1>
		<p class="subtitle">扫描二维码下载安装包</p>

		<a href="%s" class="apk-btn">
			<span>⬇️</span>
			<span>下载安装 APK</span>
		</a>

		<div class="divider">或扫码下载</div>

		<div class="qr-container">
			<img src="%s" alt="QR Code" />
		</div>

		<div class="version">
			<span>版本</span>
			<strong>%s</strong>
		</div>

		<div class="url-container">
			<div class="url-text" id="download-url">%s</div>
			<button class="copy-btn" onclick="copyURL()">复制链接</button>
		</div>
		<div class="success-msg" id="copy-msg">链接已复制到剪贴板</div>

		<p class="size">%s</p>

		<div class="instructions">
			<div class="instructions-title">使用步骤</div>
			<div class="step">
				<div class="step-num">1</div>
				<div class="step-text">下载安装 <strong>APK</strong></div>
			</div>
			<div class="step">
				<div class="step-num">2</div>
				<div class="step-text">首次打开请<strong>扫码绑定</strong>设备</div>
			</div>
			<div class="step">
				<div class="step-num">3</div>
				<div class="step-text">开始<strong>研学之旅</strong></div>
			</div>
		</div>
	</div>

	<script>
		function copyURL() {
			const url = document.getElementById('download-url').textContent;
			navigator.clipboard.writeText(url).then(() => {
				const msg = document.getElementById('copy-msg');
				msg.classList.add('show');
				setTimeout(() => msg.classList.remove('show'), 2000);
			});
		}
	</script>
</body>
</html>`, apkInfo.URL, qrCodeDataURL, apkInfo.Version, apkInfo.DownloadURL, sizeStr)

	return c.String(http.StatusOK, downloadPageHTML)
}

// formatFileSize converts bytes to human-readable string
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
