package config

import (
	"log"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort    string
	DBPath        string
	PollInterval  time.Duration
	MaxWorkers    int
	TSAuthKey     string
	LogLevel      string
	KioskPort     string
	RetentionDays int
	KioskApiKey   string
	MediaDir      string
	BaseURL       string

	// MQTT 配置
	MQTTBrokerURL  string
	MQTTPort       int
	MQTTClientID   string
	MQTTUsername   string
	MQTTPassword   string
	MQTTUseTLS     bool
	MQTTKeepAlive  time.Duration
	MQTTCleanStart bool

	// PostgreSQL 配置 (企业版)
	PostgresHost     string
	PostgresPort     int
	PostgresUser     string
	PostgresPassword string
	PostgresDatabase string
	PostgresSSLMode  string
	UsePostgres      bool

	// JWT 认证配置
	JWTSigningKey      string
	JWTAccessTokenTTL  time.Duration
	JWTRefreshTokenTTL time.Duration
	JWTIssuer          string

	// 证书签发配置
	CACertificatePath string
	CAKeyPath         string
	CertValidityDays  int

	// Web 认证配置
	WebUsername string
	WebPassword string
}

func Load() *Config {

	if err := godotenv.Load(); err != nil {
		log.Println("ℹ️ Aucun fichier .env trouvé, utilisation des variables système ou par défaut")
	}

	cfg := &Config{
		ServerPort:    getEnv("SERVER_PORT", "8081"),
		DBPath:        getEnv("DB_PATH", "freekiosk.db"),
		PollInterval:  parseDuration(getEnv("POLL_INTERVAL", "30s")),
		MaxWorkers:    parseInt(getEnv("MAX_WORKERS", "5")),
		TSAuthKey:     os.Getenv("TS_AUTHKEY"),
		LogLevel:      getEnv("LOG_LEVEL", "INFO"),
		KioskPort:     getEnv("KIOSK_PORT", "8080"),
		RetentionDays: parseInt(getEnv("RETENTION_DAYS", "31")),
		KioskApiKey:   getEnv("KIOSK_API_KEY", ""),
		MediaDir:      getEnv("MEDIA_DIR", "media"),
		BaseURL:       getEnv("BASE_URL", "localhost:8081"),

		// MQTT 配置
		MQTTBrokerURL:  getEnv("MQTT_BROKER_URL", "localhost"),
		MQTTPort:       parseInt(getEnv("MQTT_PORT", "1883")),
		MQTTClientID:   getEnv("MQTT_CLIENT_ID", "freekiosk-hub"),
		MQTTUsername:   os.Getenv("MQTT_USERNAME"),
		MQTTPassword:   os.Getenv("MQTT_PASSWORD"),
		MQTTUseTLS:     parseBool(getEnv("MQTT_USE_TLS", "false")),
		MQTTKeepAlive:  parseDuration(getEnv("MQTT_KEEPALIVE", "60s")),
		MQTTCleanStart: parseBool(getEnv("MQTT_CLEAN_START", "false")),

		// PostgreSQL 配置 (企业版)
		PostgresHost:     getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:     parseInt(getEnv("POSTGRES_PORT", "5432")),
		PostgresUser:     getEnv("POSTGRES_USER", "freekiosk"),
		PostgresPassword: os.Getenv("POSTGRES_PASSWORD"),
		PostgresDatabase: getEnv("POSTGRES_DATABASE", "freekiosk"),
		PostgresSSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
		UsePostgres:      parseBool(getEnv("USE_POSTGRES", "false")),

		// JWT 认证配置
		JWTSigningKey:      getEnv("JWT_SIGNING_KEY", ""),
		JWTAccessTokenTTL:  parseDuration(getEnv("JWT_ACCESS_TOKEN_TTL", "1h")),
		JWTRefreshTokenTTL: parseDuration(getEnv("JWT_REFRESH_TOKEN_TTL", "720h")), // 30 days
		JWTIssuer:          getEnv("JWT_ISSUER", "freekiosk-hub"),

		// 证书签发配置
		CACertificatePath: getEnv("CA_CERTIFICATE_PATH", "certs/ca.crt"),
		CAKeyPath:         getEnv("CA_KEY_PATH", "certs/ca.key"),
		CertValidityDays:  parseInt(getEnv("CERT_VALIDITY_DAYS", "365")),

		// Web 认证配置
		WebUsername: getEnv("WEB_USERNAME", "admin"),
		WebPassword: getEnv("WEB_PASSWORD", "admin123"),
	}

	initLogger(cfg.LogLevel)

	return cfg
}

func initLogger(level string) {
	var slogLevel slog.Level

	switch level {
	case "DEBUG":
		slogLevel = slog.LevelDebug
	case "WARN":
		slogLevel = slog.LevelWarn
	case "ERROR":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slogLevel,
	})

	logger := slog.New(handler)

	slog.SetDefault(logger)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		slog.Warn("Intervalle de temps invalide, retour à 30s", "valeur", s)
		return 30 * time.Second
	}
	return d
}

func parseInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		slog.Warn("Nombre entier invalide, retour à 5", "valeur", s)
		return 5
	}
	return i
}

func parseBool(s string) bool {
	return s == "true" || s == "1" || s == "yes"
}
