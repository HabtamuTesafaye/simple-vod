package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	ListenAddr string
	DataDir    string

	// S3 Config
	S3Endpoint       string
	S3PublicEndpoint string
	S3AccessKey      string
	S3SecretKey      string
	S3Region         string
	S3Bucket         string
	S3UsePathStyle   bool

	// Video Settings
	MaxUploadSize int64
	AllowedTypes  []string
	PresignExpiry time.Duration
}

func LoadConfig() *Config {
	_ = godotenv.Load() // Ignore error if .env is missing

	cfg := &Config{
		ListenAddr:       getEnv("VOD_LISTEN_ADDR", ":8080"),
		DataDir:          getEnv("VOD_DATA_DIR", "./data"),
		S3Endpoint:       getEnv("VOD_S3_ENDPOINT", "http://localhost:3901"),
		S3PublicEndpoint: getEnv("VOD_S3_PUBLIC_ENDPOINT", "http://localhost:3901"),
		S3AccessKey:      getEnv("VOD_S3_ACCESS_KEY", ""),
		S3SecretKey:      getEnv("VOD_S3_SECRET_KEY", ""),
		S3Region:         getEnv("VOD_S3_REGION", "garage"),
		S3Bucket:         getEnv("VOD_S3_BUCKET", "vod-private"),
		S3UsePathStyle:   getEnvBool("VOD_S3_USE_PATH_STYLE", true),
		MaxUploadSize:    getEnvInt64("VOD_MAX_UPLOAD_SIZE", 524288000), // 500MB
		AllowedTypes:     strings.Split(getEnv("VOD_ALLOWED_TYPES", "mp4,mov,mkv,webm"), ","),
		PresignExpiry:    time.Duration(getEnvInt64("VOD_PRESIGN_EXPIRY", 14400)) * time.Second,
	}

	if cfg.S3AccessKey == "" || cfg.S3SecretKey == "" {
		log.Println("WARNING: S3 credentials are not set")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		b, err := strconv.ParseBool(value)
		if err == nil {
			return b
		}
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	if value, exists := os.LookupEnv(key); exists {
		i, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			return i
		}
	}
	return fallback
}
