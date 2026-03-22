package api

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/wared2003/freekiosk-hub/internal/clients"
	"github.com/wared2003/freekiosk-hub/internal/config"
	"github.com/wared2003/freekiosk-hub/internal/i18n"
	"github.com/wared2003/freekiosk-hub/internal/repositories"
	"github.com/wared2003/freekiosk-hub/internal/services"
	"github.com/wared2003/freekiosk-hub/internal/sse"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// ApiServer 集中路由所需的依赖项
type ApiServer struct {
	Echo         *echo.Echo
	DB           *sql.DB
	TabletRepo   repositories.TabletRepository
	ReportRepo   repositories.ReportRepository
	GroupRepo    repositories.GroupRepository
	FTRepo       *repositories.FieldTripRepository
	MonitorSvc   services.MonitorService
	KioskClient  clients.KioskClient
	Cfg          config.Config
	MediaService services.MediaService
	MQTTService  *services.MQTTService // MQTT 服务，用于实时双向通信
	AuthSvc      services.AuthService  // 企业版: 认证服务
	CmdSvc       services.CommandService // 企业版: 命令服务
	StatusSvc    services.DeviceStatusService // 企业版: 状态服务
	PolicySvc    services.PolicyService // 企业版: 策略服务
	TenantSvc    services.TenantService  // 企业版: 租户服务
	MetricsSvc   *services.MetricsService // 企业版: 指标服务
	AuditSvc     *services.AuditService  // 企业版: 审计日志服务
}

// NewRouter 初始化服务器、处理器和路由
func NewRouter(e *echo.Echo, db *sql.DB,
	tr repositories.TabletRepository,
	rr repositories.ReportRepository,
	gr repositories.GroupRepository,
	ftRepo *repositories.FieldTripRepository,
	ms services.MonitorService,
	ks clients.KioskClient,
	cfg config.Config,
	mes services.MediaService,
	mqttSvc *services.MQTTService, // 新增 MQTT 服务参数
	authSvc services.AuthService, // 企业版: 认证服务
	cmdSvc services.CommandService, // 企业版: 命令服务
	statusSvc services.DeviceStatusService, // 企业版: 状态服务
	policySvc services.PolicyService, // 企业版: 策略服务
	tenantSvc services.TenantService, // 企业版: 租户服务
	metricsSvc *services.MetricsService, // 企业版: 指标服务
	auditSvc *services.AuditService, // 企业版: 审计日志服务

) *ApiServer {
	s := &ApiServer{
		Echo:         e,
		DB:           db,
		TabletRepo:   tr,
		ReportRepo:   rr,
		GroupRepo:    gr,
		FTRepo:       ftRepo,
		MonitorSvc:   ms,
		KioskClient:  ks,
		Cfg:          cfg,
		MediaService: mes,
		MQTTService:  mqttSvc,
		AuthSvc:      authSvc,
		CmdSvc:       cmdSvc,
		StatusSvc:    statusSvc,
		PolicySvc:    policySvc,
		TenantSvc:    tenantSvc,
		MetricsSvc:   metricsSvc,
		AuditSvc:     auditSvc,
	}

	s.setupMiddlewares()
	s.setupRoutes()

	return s
}

func (s *ApiServer) setupMiddlewares() {
	// Language middleware for i18n
	s.Echo.Use(i18n.LanguageMiddleware)

	// Nouveau RequestLogger : Plus propre et structuré
	s.Echo.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogMethod:   true,
		LogLatency:  true,
		LogError:    true,
		LogRemoteIP: true,
		Skipper: func(c echo.Context) bool {
			return strings.Contains(c.Path(), "/sse")
		},
		HandleError: true, // Pour que les erreurs passent aussi par ici
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			if v.Error != nil {
				slog.Error("HTTP Request Error",
					"method", v.Method,
					"uri", v.URI,
					"status", v.Status,
					"latency", v.Latency,
					"remote_ip", v.RemoteIP,
					"error", v.Error,
				)
			} else {
				slog.Info("HTTP Request",
					"method", v.Method,
					"uri", v.URI,
					"status", v.Status,
					"latency", v.Latency,
					"remote_ip", v.RemoteIP,
				)
			}
			return nil
		},
	}))

	s.Echo.Use(middleware.Recover())
	s.Echo.Static("/static", "static")
}

func (s *ApiServer) setupRoutes() {

	kService := services.NewKioskService(s.TabletRepo, s.GroupRepo, s.KioskClient, s.Cfg.KioskPort)

	homeH := NewHtmlHomeHandler(s.TabletRepo, s.ReportRepo, s.GroupRepo)
	tabletH := NewHtmlTabletHandler(s.TabletRepo, s.ReportRepo, s.GroupRepo, kService, s.MediaService)
	groupH := NewGroupHandler(s.GroupRepo)
	bcastSvc := services.NewBroadcastService(s.FTRepo, s.MQTTService)
	fieldtripH := NewFieldTripHandler(s.FTRepo, "" /* signing pubkey — empty for MVP */, bcastSvc)
	fieldtripUIH := NewFieldTripUIHandler(s.FTRepo, bcastSvc)
	exportH := NewExportHandler(s.FTRepo)
	downloadH := NewDownloadHandler("apk", s.Cfg.ServerPort)

	systemJsonH := NewSystemJSONHandler(s.DB)

	// --- 2. ROUTES PUBLIQUES / SYSTÈME ---
	s.Echo.GET("/health", systemJsonH.HandleHealthCheck)

	// MQTT 状态检查端点
	s.Echo.GET("/health/mqtt", func(c echo.Context) error {
		if s.MQTTService == nil {
			return c.JSON(503, map[string]interface{}{
				"status":  "unavailable",
				"message": "MQTT 服务未配置",
			})
		}
		if !s.MQTTService.IsConnected() {
			return c.JSON(503, map[string]interface{}{
				"status":  "disconnected",
				"message": "MQTT 未连接到 Broker",
			})
		}
		return c.JSON(200, map[string]interface{}{
			"status":  "connected",
			"message": "MQTT 服务正常运行",
		})
	})

	s.Echo.GET("/", homeH.HandleIndex)
	tablets := s.Echo.Group("/tablets")
	{
		tablets.GET("/:id", tabletH.HandleDetails)
		tablets.GET("/:id/groups-selection", groupH.HandleTabletGroupsSelection)
		tablets.POST("/:tabletID/groups/:groupID/toggle", groupH.HandleToggleGroup)

		//commands
		tablets.POST("/:id/command/beep", tabletH.HandleBeep)
		tablets.POST("/:id/command/reload", tabletH.HandleReload)
		tablets.POST("/:id/command/reboot", tabletH.HandleReboot)
		tablets.POST("/:id/command/wake", tabletH.HandleWakeUp)
		tablets.POST("/:id/command/screen-status", tabletH.HandleScreenStatus)
		tablets.POST("/:id/command/screensaver-status", tabletH.HandleScreenSaver)
		tablets.POST("/:id/command/navigate", tabletH.HandleNavigate)
		tablets.GET("/:id/navigate-modal", tabletH.HandleNavigateModal)

		tablets.GET("/:id/sound-modal", tabletH.HandleSoundModal)
		tablets.POST("/:id/sound/upload", tabletH.HandleUploadSound)
		//e.DELETE("/sound/:filename", tabletH.HandleDeleteSound)

		tablets.POST("/:id/command/play-sound", tabletH.HandlePlaySound)
		tablets.POST("/:id/command/gtsl-tts", tabletH.HandleGtslTTSSound)
		tablets.POST("/:id/command/stop-sound", tabletH.HandleStopSound)

	}

	groupRoutes := s.Echo.Group("/groups")
	{
		groupRoutes.GET("", groupH.HandleGroups)
		groupRoutes.GET("/new", groupH.HandleNewGroup)
		groupRoutes.GET("/edit/:id", groupH.HandleEditGroup)
		groupRoutes.POST("/save", groupH.HandleSaveGroup)
		groupRoutes.DELETE("/:id", groupH.HandleDeleteGroup)
	}

	// Field Trip UI routes
	fieldtripRoutes := s.Echo.Group("/fieldtrip")
	{
		fieldtripRoutes.GET("", fieldtripUIH.HandleFieldTripPage)
		fieldtripRoutes.GET("/groups/new", fieldtripUIH.HandleNewGroup)
		fieldtripRoutes.GET("/groups/:id/edit", fieldtripUIH.HandleEditGroup)
		fieldtripRoutes.POST("/groups/save", fieldtripUIH.HandleSaveGroup)
		fieldtripRoutes.DELETE("/groups/:id", fieldtripUIH.HandleDeleteGroup)
		fieldtripRoutes.DELETE("/devices/:id", fieldtripUIH.HandleDeleteDevice)
		fieldtripRoutes.POST("/broadcast", fieldtripUIH.HandleBroadcast)
		fieldtripRoutes.POST("/devices/:id/whitelist", fieldtripUIH.HandleSetWhitelist)
		fieldtripRoutes.POST("/ota/upload", fieldtripUIH.HandleOTAUpload)
	}

	// Download page for APK
	s.Echo.GET("/download", downloadH.HandleDownloadPage)

	// --- 5. 企业版认证 API ---
	if s.AuthSvc != nil {
		authH := NewAuthHandler(s.AuthSvc)
		authRoutes := s.Echo.Group("/api/v2/auth")
		{
			authRoutes.POST("/register", authH.HandleRegister)
			authRoutes.POST("/refresh", authH.HandleRefreshToken)
			authRoutes.GET("/validate/:deviceId", authH.HandleValidateDevice)
			authRoutes.DELETE("/device/:deviceId", authH.HandleRevokeDevice)
			authRoutes.POST("/token", authH.HandleGetToken)
		}
	}

	// --- 6. 企业版命令 API ---
	if s.CmdSvc != nil {
		cmdH := NewCommandHandler(s.CmdSvc, s.StatusSvc)
		tenantRoutes := s.Echo.Group("/api/v2/tenants/:tenantId")
		{
			// 设备命令
			tenantRoutes.POST("/devices/:deviceId/commands", cmdH.HandleSendCommand)
			tenantRoutes.GET("/devices/:deviceId/commands/history", cmdH.HandleGetCommandHistory)
			tenantRoutes.GET("/commands/:commandId", cmdH.HandleGetCommandByID)
			tenantRoutes.DELETE("/commands/:commandId", cmdH.HandleCancelCommand)

			// 批量命令
			tenantRoutes.POST("/commands/batch", cmdH.HandleSendBatchCommand)
		}
	}

	// --- 7. 企业版状态 API ---
	if s.StatusSvc != nil {
		statusH := NewStatusHandler(s.StatusSvc)
		statusRoutes := s.Echo.Group("/api/v2/tenants/:tenantId")
		{
			statusRoutes.GET("/devices/:deviceId/status", statusH.HandleGetDeviceStatus)
			statusRoutes.GET("/devices/status", statusH.HandleGetAllDeviceStatuses)
		}
	}

	// --- 8. 企业版策略 API ---
	if s.PolicySvc != nil {
		policyH := NewPolicyHandler(s.PolicySvc)
		policyRoutes := s.Echo.Group("/api/v2/tenants/:tenantId")
		{
			// 策略管理
			policyRoutes.POST("/policies", policyH.CreatePolicy)
			policyRoutes.GET("/policies", policyH.ListPolicies)
			policyRoutes.GET("/policies/:policyId", policyH.GetPolicy)
			policyRoutes.PUT("/policies/:policyId", policyH.UpdatePolicy)
			policyRoutes.DELETE("/policies/:policyId", policyH.DeletePolicy)

			// 策略分配
			policyRoutes.POST("/policies/:policyId/assign", policyH.AssignPolicy)

			// 白名单管理
			policyRoutes.POST("/policies/:policyId/whitelist", policyH.AddAppToWhitelist)
			policyRoutes.DELETE("/policies/:policyId/whitelist/:packageName", policyH.RemoveAppFromWhitelist)

			// 设备策略
			policyRoutes.GET("/devices/:deviceId/policy", policyH.GetDevicePolicy)
			policyRoutes.GET("/devices/:deviceId/whitelist", policyH.GetDeviceWhitelist)
		}
	}

	// --- 9. 企业版租户 API ---
	if s.TenantSvc != nil {
		tenantH := NewTenantHandler(s.TenantSvc)
		tenantRoutes := s.Echo.Group("/api/v2/tenants")
		{
			tenantRoutes.POST("", tenantH.HandleCreateTenant)
			tenantRoutes.GET("", tenantH.HandleListTenants)
			tenantRoutes.GET("/:tenantId", tenantH.HandleGetTenant)
			tenantRoutes.PUT("/:tenantId", tenantH.HandleUpdateTenant)
			tenantRoutes.DELETE("/:tenantId", tenantH.HandleDeleteTenant)
			tenantRoutes.GET("/:tenantId/quota", tenantH.HandleGetQuota)
			tenantRoutes.PUT("/:tenantId/quota", tenantH.HandleUpdateQuota)
		}
	}

	// --- 10. 企业版 Prometheus 指标 ---
	if s.MetricsSvc != nil {
		s.Echo.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
	}

	// --- 11. 企业版审计日志 API ---
	if s.AuditSvc != nil {
		auditH := NewAuditHandler(s.AuditSvc)
		// 审计日志查询 (挂载在租户路径下)
		auditRoutes := s.Echo.Group("/api/v2/tenants/:tenantId")
		{
			auditRoutes.GET("/audit-logs", auditH.HandleQueryAuditLogs)
		}
	}

	// --- Field Trip API v2 ---
	otaHandler := NewOTAHandler("apk")
	fieldtrip := s.Echo.Group("/api/v2/fieldtrip")
	{
		fieldtrip.POST("/groups", fieldtripH.CreateGroup)
		fieldtrip.GET("/groups", fieldtripH.ListGroups)
		fieldtrip.DELETE("/groups/:id", fieldtripH.DeleteGroup)
		fieldtrip.POST("/devices", fieldtripH.CreateDevice)
		fieldtrip.GET("/devices", fieldtripH.ListDevices)
		fieldtrip.DELETE("/devices/:id", fieldtripH.DeleteDevice)
		fieldtrip.PATCH("/devices/:id", fieldtripH.UpdateDevice)
		fieldtrip.POST("/devices/bind", fieldtripH.BindDevice)
		fieldtrip.POST("/devices/:id/location", fieldtripH.ReportLocation)
		fieldtrip.GET("/devices/:id/location/history", fieldtripH.GetLocationHistory)
		fieldtrip.GET("/commands", fieldtripH.PollCommands)
		fieldtrip.POST("/devices/:id/whitelist", fieldtripH.SetWhitelist)
		fieldtrip.POST("/broadcast", fieldtripH.SendBroadcast)
		fieldtrip.POST("/ota/upload", otaHandler.UploadOTA)
		fieldtrip.GET("/ota/list", otaHandler.ListOTA)
		fieldtrip.GET("/groups/:id/export", exportH.HandleExportPDF)
	}
	s.Echo.Static("/apk", "apk")

	// --- 8. WebSocket 实时通信 ---
	wsHandler := NewWebSocketHandler(sse.WsHub)
	s.Echo.GET("/api/v2/ws", wsHandler.HandleWebSocket)
	s.Echo.GET("/api/v2/ws/connections", wsHandler.HandleGetConnectionCount)

	// s.Echo.GET("/admin/import", adminPageH.HandleImportPage)

	// // --- 4. ROUTES API (JSON) ---
	// // On groupe les routes API sous /api/v1
	// apiV1 := s.Echo.Group("/api/v1")

	// // Si une clé API est configurée, on pourrait ajouter un middleware ici
	// // apiV1.Use(CustomApiKeyMiddleware(s.ApiKey))

	// apiV1.GET("/tablets", tabletJsonH.HandleListTablets)
	// apiV1.POST("/tablets/import", tabletJsonH.HandleBulkImport) // Pour tes 500 tablettes
	// apiV1.POST("/tablets/:ip/scan", tabletJsonH.HandleManualScan)

	//sse
	s.Echo.GET("/sse/global", func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "text/event-stream")
		c.Response().Header().Set("Cache-Control", "no-cache")
		c.Response().Header().Set("Connection", "keep-alive")
		c.Response().Header().Set("X-Accel-Buffering", "no")
		fmt.Fprintf(c.Response().Writer, "data: connected\n\n")
		c.Response().Flush()

		ch := sse.Instance.SubscribeGlobal()
		defer sse.Instance.Unsubscribe(ch, 0)
		for {
			select {
			case <-ch:
				// On envoie l'event "refresh"
				fmt.Fprintf(c.Response().Writer, "event: update\ndata: \n\n")
				c.Response().Flush()
			case <-c.Request().Context().Done():
				return nil
			}
		}
	})

	s.Echo.GET("/sse/tablet/:id", func(c echo.Context) error {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

		c.Response().Header().Set("Content-Type", "text/event-stream")
		c.Response().Header().Set("Cache-Control", "no-cache")
		c.Response().Header().Set("Connection", "keep-alive")
		c.Response().Header().Set("X-Accel-Buffering", "no")
		fmt.Fprintf(c.Response().Writer, "data: connected\n\n")
		c.Response().Flush()

		ch := sse.Instance.SubscribeTablet(id)
		defer sse.Instance.Unsubscribe(ch, id)

		for {
			select {
			case <-ch:
				fmt.Fprintf(c.Response().Writer, "event: update\ndata: \n\n")
				c.Response().Flush()
			case <-c.Request().Context().Done():
				return nil
			}
		}
	})
}
