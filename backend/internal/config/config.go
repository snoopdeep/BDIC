// Package config reads every runtime setting from the environment exactly once
// at start-up, so nothing deeper in the program reads os.Getenv.
//
// Anything the school has to supply itself — the database password, the signing
// key, the SMS and payment credentials — is listed in .env.example with a blank
// value and a comment saying where to get it. Nothing here has a usable
// production default.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the whole of the application's configuration.
type Config struct {
	Env            string
	Port           string
	DatabaseURL    string
	CORSOrigins    []string
	SigningKey     []byte
	TokenTTL       time.Duration
	UploadDir      string
	MaxUploadBytes int64
	PublicBaseURL  string
	BootstrapAdmin BootstrapAdmin
	Providers      Providers
}

// BootstrapAdmin is the first account created on an empty database so somebody
// can sign in and create the rest. After the real Principal account exists,
// this should be disabled by removing the values from .env.
type BootstrapAdmin struct {
	Email    string
	Password string
	Name     string
}

// Providers holds the external-service credentials. Every one of these is blank
// in the local build. A blank credential does not fail at start-up: the feature
// that needs it records a DRAFT row instead of sending anything, which is what
// the school approved.
type Providers struct {
	SMSProvider       string
	SMSAPIKey         string
	SMSSenderID       string
	SMSDLTEntityID    string
	WhatsAppToken     string
	WhatsAppPhoneID   string
	SMTPHost          string
	SMTPPort          string
	SMTPUser          string
	SMTPPassword      string
	SMTPFrom          string
	PaymentProvider   string
	PaymentKeyID      string
	PaymentKeySecret  string
	PaymentWebhookKey string
	S3Bucket          string
	S3Region          string
}

// ErrMissingDatabaseURL is returned when DATABASE_URL is absent. The program
// cannot do anything useful without a database, so this is fatal rather than
// silently falling back to demo data.
var ErrMissingDatabaseURL = errors.New(
	"DATABASE_URL is not set: copy .env.example to .env and fill it in")

// Load reads the environment and validates it.
func Load() (Config, error) {
	cfg := Config{
		Env:            get("APP_ENV", "development"),
		Port:           get("PORT", "8080"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		CORSOrigins:    splitAndTrim(get("CORS_ORIGINS", "http://localhost:3000")),
		UploadDir:      get("UPLOAD_DIR", "./var/uploads"),
		MaxUploadBytes: getInt64("MAX_UPLOAD_BYTES", 8<<20), // 8 MiB: a photographed document on a 4G phone.
		PublicBaseURL:  get("PUBLIC_BASE_URL", "http://localhost:3000"),
		BootstrapAdmin: BootstrapAdmin{
			Email:    os.Getenv("BOOTSTRAP_ADMIN_EMAIL"),
			Password: os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"),
			Name:     get("BOOTSTRAP_ADMIN_NAME", "BDIC Administrator"),
		},
		Providers: Providers{
			SMSProvider:       os.Getenv("SMS_PROVIDER"),
			SMSAPIKey:         os.Getenv("SMS_API_KEY"),
			SMSSenderID:       os.Getenv("SMS_SENDER_ID"),
			SMSDLTEntityID:    os.Getenv("SMS_DLT_ENTITY_ID"),
			WhatsAppToken:     os.Getenv("WHATSAPP_ACCESS_TOKEN"),
			WhatsAppPhoneID:   os.Getenv("WHATSAPP_PHONE_NUMBER_ID"),
			SMTPHost:          os.Getenv("SMTP_HOST"),
			SMTPPort:          get("SMTP_PORT", "587"),
			SMTPUser:          os.Getenv("SMTP_USER"),
			SMTPPassword:      os.Getenv("SMTP_PASSWORD"),
			SMTPFrom:          os.Getenv("SMTP_FROM"),
			PaymentProvider:   os.Getenv("PAYMENT_PROVIDER"),
			PaymentKeyID:      os.Getenv("PAYMENT_KEY_ID"),
			PaymentKeySecret:  os.Getenv("PAYMENT_KEY_SECRET"),
			PaymentWebhookKey: os.Getenv("PAYMENT_WEBHOOK_SECRET"),
			S3Bucket:          os.Getenv("S3_BUCKET"),
			S3Region:          get("S3_REGION", "ap-south-1"),
		},
	}

	ttlHours := getInt64("TOKEN_TTL_HOURS", 10) // One school day plus the evening.
	cfg.TokenTTL = time.Duration(ttlHours) * time.Hour

	if cfg.DatabaseURL == "" {
		return Config{}, ErrMissingDatabaseURL
	}

	signingKey := os.Getenv("AUTH_SIGNING_KEY")
	if len(signingKey) < 32 {
		return Config{}, fmt.Errorf(
			"AUTH_SIGNING_KEY must be at least 32 characters (got %d): generate one with `openssl rand -base64 48`",
			len(signingKey))
	}
	cfg.SigningKey = []byte(signingKey)

	if cfg.IsProduction() {
		if err := cfg.validateProduction(); err != nil {
			return Config{}, err
		}
	}

	return cfg, nil
}

// IsProduction reports whether this process is serving the real school.
func (c Config) IsProduction() bool {
	return strings.EqualFold(c.Env, "production")
}

// validateProduction refuses to start with the kind of configuration that is
// fine locally and dangerous once parents are using it.
func (c Config) validateProduction() error {
	if c.BootstrapAdmin.Password != "" {
		return errors.New(
			"BOOTSTRAP_ADMIN_PASSWORD must be empty in production: create the Principal account, then remove it")
	}
	for _, origin := range c.CORSOrigins {
		if strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1") {
			return fmt.Errorf("CORS_ORIGINS contains a local address (%q) but APP_ENV is production", origin)
		}
	}
	if !strings.HasPrefix(c.PublicBaseURL, "https://") {
		return errors.New("PUBLIC_BASE_URL must use https:// in production")
	}
	return nil
}

func get(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func getInt64(key string, fallback int64) int64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func splitAndTrim(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
