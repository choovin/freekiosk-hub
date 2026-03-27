// Copyright (C) 2026 wared2003
// SPDX-License-Identifier: AGPL-3.0-or-later
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/wared2003/freekiosk-hub/internal/services"

	"github.com/wared2003/freekiosk-hub/internal/repositories"

	"github.com/wared2003/freekiosk-hub/internal/network"

	"github.com/wared2003/freekiosk-hub/internal/databases"

	"github.com/wared2003/freekiosk-hub/internal/config"

	"github.com/wared2003/freekiosk-hub/internal/sse"

	"github.com/wared2003/freekiosk-hub/internal/clients"

	"github.com/wared2003/freekiosk-hub/internal/api"
	"github.com/wared2003/freekiosk-hub/internal/i18n"

	_ "github.com/wared2003/freekiosk-hub/docs" // swagger docs
)

type ApiKeyTransport struct {
	Transport http.RoundTripper
	ApiKey    string
}

func (t *ApiKeyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// On ajoute le header à chaque requête sortante
	req.Header.Add("X-Api-Key", t.ApiKey)
	return t.Transport.RoundTrip(req)
}

func main() {
	// 1. Configuration & Logger initialization
	cfg := config.Load()

	slog.Info("🚀 Starting FreeKiosk Hub",
		"port", cfg.ServerPort,
		"db_path", cfg.DBPath,
	)

	// Initialize i18n translations
	i18nStore := i18n.GetStore()
	// Use /app/locales for Docker, internal/i18n/locales for local development
	localePath := "internal/i18n/locales"
	if _, err := os.Stat("/app/locales"); err == nil {
		localePath = "/app/locales"
	}
	if err := i18nStore.LoadTranslations(localePath); err != nil {
		slog.Warn("Could not load all translations", "error", err)
	}
	// Set default language (can be overridden per-request via middleware)
	i18n.SetLang("en")

	// Global context for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var httpClient *http.Client

	// 2. Network Management (Tailscale vs Standard)
	if cfg.TSAuthKey != "" {
		slog.Info("🔐 Tailscale auth key detected, connecting to tailnet...")

		tsNode, err := network.InitTailscale(cfg.TSAuthKey, "freekiosk-hub-server")
		if err != nil {
			slog.Error("❌ Failed to initialize Tailscale", "error", err)
			os.Exit(1)
		}
		defer tsNode.Close()

		slog.Info("⏳ Waiting for Tailscale network to be up...")
		if _, err := tsNode.Server.Up(ctx); err != nil {
			slog.Error("❌ Could not bring Tailscale network up", "error", err)
			os.Exit(1)
		}
		slog.Info("✅ Tailscale network is operational")
		httpClient = tsNode.Client
	} else {
		slog.Warn("⚠️ No Tailscale key found. Using standard network stack.")
		httpClient = &http.Client{
			Timeout: 15 * time.Second,
		}
	}

	httpClient.Transport = &ApiKeyTransport{
		Transport: httpClient.Transport,
		ApiKey:    cfg.KioskApiKey,
	}

	// 3. Database connection
	db, err := databases.Open(cfg.DBPath)
	if err != nil {
		slog.Error("❌ Failed to open database", "path", cfg.DBPath, "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// 4. Repositories & Clients initialization
	tabletRepo := repositories.NewTabletRepository(db)
	reportRepo := repositories.NewReportRepository(db)
	groupRepo := repositories.NewGroupRepository(db)
	kioskClient := clients.NewKioskClient(httpClient)
	ftRepo := repositories.NewFieldTripRepository(db, cfg.BaseURL)

	// Ensure tables exist
	if err := tabletRepo.InitTable(); err != nil {
		slog.Error("❌ Failed to initialize tablets table", "error", err)
		os.Exit(1)
	}
	if err := reportRepo.InitTable(); err != nil {
		slog.Error("❌ Failed to initialize reports table", "error", err)
		os.Exit(1)
	}
	if err := groupRepo.InitTable(); err != nil {
		slog.Error("Échec initialisation table groups", "err", err)
		os.Exit(1)
	}
	if err := ftRepo.InitSchema(); err != nil {
		slog.Error("Échec initialisation table fieldtrip", "err", err)
		os.Exit(1)
	}
	slog.Info("✅ Database schema is ready")

	mediaService := services.NewMediaService(cfg.MediaDir, cfg.BaseURL)

	// 5. 监控服务初始化
	monitorSvc := services.NewMonitorService(
		tabletRepo,
		reportRepo,
		kioskClient,
		cfg.MaxWorkers,
		cfg.KioskPort,
		cfg.PollInterval,
		cfg.RetentionDays,
	)

	// 6. MQTT 服务初始化
	// MQTT 提供实时双向通信能力，用于设备状态同步和命令下发
	var mqttService *services.MQTTService
	mqttCfg := &services.MQTTServiceConfig{
		BrokerURL:  cfg.MQTTBrokerURL,
		Port:       cfg.MQTTPort,
		ClientID:   cfg.MQTTClientID,
		Username:   cfg.MQTTUsername,
		Password:   cfg.MQTTPassword,
		UseTLS:     cfg.MQTTUseTLS,
		KeepAlive:  cfg.MQTTKeepAlive,
		CleanStart: cfg.MQTTCleanStart,
		TenantID:   "default", // 默认租户 ID，后续可从配置读取
	}
	mqttService = services.NewMQTTService(tabletRepo, reportRepo, mqttCfg)

	// 连接 MQTT Broker
	if err := mqttService.Connect(ctx); err != nil {
		slog.Warn("⚠️ MQTT 连接失败，将使用 HTTP 轮询模式", "error", err)
	} else {
		slog.Info("✅ MQTT 服务已连接", "broker", cfg.MQTTBrokerURL, "port", cfg.MQTTPort)
	}
	defer func() {
		if mqttService.IsConnected() {
			if err := mqttService.Disconnect(context.Background()); err != nil {
				slog.Error("❌ MQTT 断开连接失败", "error", err)
			}
		}
	}()

	// 7. Policy 服务初始化
	policyRepo := repositories.NewSecurityPolicyRepository(db)
	appWhitelistRepo := repositories.NewAppWhitelistRepository(db)
	policySvc := services.NewPolicyService(policyRepo, appWhitelistRepo, services.PolicyServiceConfig{
		DefaultPolicyID: "00000000-0000-0000-0000-000000000001",
	})
	slog.Info("✅ Policy 服务已初始化")

	// 8. 租户服务初始化
	tenantRepo := repositories.NewTenantRepository(db)
	if err := tenantRepo.InitTable(ctx); err != nil {
		slog.Error("❌ Failed to initialize tenants table", "error", err)
		os.Exit(1)
	}
	deviceRepo := repositories.NewDeviceRepository(db)
	tenantSvc := services.NewTenantService(tenantRepo, deviceRepo)
	slog.Info("✅ 租户服务已初始化")

	// 9. 指标服务初始化
	metricsSvc := services.NewMetricsService()
	slog.Info("✅ 指标服务已初始化")

	// 10. 审计日志服务初始化
	auditSvc := services.NewAuditService(db)
	if err := auditSvc.InitTable(ctx); err != nil {
		slog.Error("❌ Failed to initialize audit_logs table", "error", err)
		os.Exit(1)
	}
	slog.Info("✅ 审计日志服务已初始化")

	// 11. MDM平板设备服务初始化
	mdmTabletRepo := repositories.NewSQLiteMDMTabletRepository(db)
	mdmTabletSvc := services.NewMDMTabletService(mdmTabletRepo)
	if err := mdmTabletRepo.InitSchema(ctx); err != nil {
		slog.Error("❌ Failed to initialize MDM tablets schema", "error", err)
		os.Exit(1)
	}
	slog.Info("✅ MDM平板设备服务已初始化")

	// 12. 配置档案服务初始化
	configRepo := repositories.NewSQLiteConfigurationRepository(db)
	configSvc := services.NewConfigurationService(configRepo)
	if err := configRepo.InitSchema(ctx); err != nil {
		slog.Error("❌ Failed to initialize configuration schema", "error", err)
		os.Exit(1)
	}
	slog.Info("✅ 配置档案服务已初始化")

	// 13. 应用包服务初始化
	appPkgRepo := repositories.NewSQLiteAppPackageRepository(db)
	appPkgSvc := services.NewAppPackageService(appPkgRepo, cfg.MediaDir, cfg.BaseURL)
	if err := appPkgRepo.InitSchema(ctx); err != nil {
		slog.Error("❌ Failed to initialize app package schema", "error", err)
		os.Exit(1)
	}
	slog.Info("✅ 应用包服务已初始化")

	// 14. 地理围栏服务初始化
	geofenceRepo := repositories.NewSQLiteGeofenceRepository(db)
	geofenceSvc := services.NewGeofenceService(geofenceRepo)
	if err := geofenceRepo.InitSchema(ctx); err != nil {
		slog.Error("❌ Failed to initialize geofence schema", "error", err)
		os.Exit(1)
	}
	slog.Info("✅ 地理围栏服务已初始化")

	// 15. 远程控制服务初始化
	remoteControlRepo := repositories.NewSQLiteRemoteControlRepository(db)
	remoteControlSvc := services.NewRemoteControlService(remoteControlRepo)
	if err := remoteControlRepo.InitSchema(ctx); err != nil {
		slog.Error("❌ Failed to initialize remote control schema", "error", err)
		os.Exit(1)
	}
	slog.Info("✅ 远程控制服务已初始化")

	// 16. 推送通知服务初始化
	pushRepo := repositories.NewSQLitePushNotificationRepository(db)
	pushSvc := services.NewPushNotificationService(pushRepo, mqttService)
	if err := pushRepo.InitSchema(ctx); err != nil {
		slog.Error("❌ Failed to initialize push notification schema", "error", err)
		os.Exit(1)
	}
	slog.Info("✅ 推送通知服务已初始化")

	// 17. 用户管理服务初始化
	userRepo := repositories.NewSQLiteUserRepository(db)
	userSvc := services.NewUserService(userRepo)
	if err := userRepo.InitSchema(ctx); err != nil {
		slog.Error("❌ Failed to initialize user schema", "error", err)
		os.Exit(1)
	}
	slog.Info("✅ 用户管理服务已初始化")

	// 7. 初始化 WebSocket Hub (用于实时通知)
	sse.InitWsHub()
	slog.Info("✅ WebSocket Hub 已初始化")

	// 6. Launch Background Monitor Service
	if cfg.PollInterval > 0 {
		go func() {
			slog.Info("📡 Starting background monitoring service", "interval", cfg.PollInterval)
			if err := monitorSvc.Start(ctx); err != nil && err != context.Canceled {
				slog.Error("❌ Monitor service exited with error", "error", err)
			}
		}()
	} else {
		slog.Warn("ℹ️ Automatic monitoring is disabled (POLL_INTERVAL <= 0)")
	}

	e := echo.New()
	e.Renderer = &api.TemplRenderer{}
	api.NewRouter(e, db.DB, tabletRepo, reportRepo, groupRepo, ftRepo, monitorSvc, kioskClient, *cfg, mediaService, mqttService, nil, nil, nil, policySvc, tenantSvc, metricsSvc, auditSvc, mdmTabletRepo, mdmTabletSvc, configSvc, appPkgSvc, geofenceSvc, remoteControlSvc, pushSvc, userSvc)
	e.Static("/media", cfg.MediaDir)
	go func() {
		slog.Info("🌐 Web Server starting", "port", cfg.ServerPort)
		if err := e.Start(":" + cfg.ServerPort); err != nil && err != http.ErrServerClosed {
			slog.Error("❌ Server failed", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("🌐 Hub is fully operational. Waiting for interrupt signals...")
	<-ctx.Done()

	slog.Warn("⚠️ Shutdown signal received, stopping server...")

	time.Sleep(1 * time.Second)
	slog.Info("👋 Shutdown complete. Bye!")
}
