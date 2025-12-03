// Package config provides configuration management using Viper.
package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Server  ServerConfig
	TTS     TTSConfig
	Queue   QueueConfig
	Storage StorageConfig
	Logging LoggingConfig
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// TTSConfig holds TTS-related configuration.
type TTSConfig struct {
	ElevenLabsAPIKey  string        `mapstructure:"elevenlabs_api_key"`
	DefaultVoiceID    string        `mapstructure:"default_voice_id"`
	MaxSyncTextLength int           `mapstructure:"max_sync_text_length"`
	SyncTimeout       time.Duration `mapstructure:"sync_timeout"`
}

// QueueConfig holds job queue configuration.
type QueueConfig struct {
	WorkerCount       int `mapstructure:"worker_count"`
	MaxConcurrentJobs int `mapstructure:"max_concurrent_jobs"`
}

// StorageConfig holds storage configuration.
type StorageConfig struct {
	AudioStoragePath  string `mapstructure:"audio_storage_path"`
	JobRetentionHours int    `mapstructure:"job_retention_hours"`
}

// LoggingConfig holds logging configuration.
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("http_port", 8080)
	v.SetDefault("http_read_timeout", "60s")
	v.SetDefault("http_write_timeout", "60s")
	v.SetDefault("default_voice_id", "pNInz6obpgDQGcFmaJgB")
	v.SetDefault("max_sync_text_length", 5000)
	v.SetDefault("sync_timeout", "30s")
	v.SetDefault("worker_count", 4)
	v.SetDefault("max_concurrent_jobs", 100)
	v.SetDefault("audio_storage_path", "./audio_cache")
	v.SetDefault("job_retention_hours", 24)
	v.SetDefault("log_level", "info")
	v.SetDefault("log_format", "json")

	// Read from environment
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Parse durations from strings
	readTimeout, err := time.ParseDuration(v.GetString("http_read_timeout"))
	if err != nil {
		readTimeout = 60 * time.Second
	}

	writeTimeout, err := time.ParseDuration(v.GetString("http_write_timeout"))
	if err != nil {
		writeTimeout = 60 * time.Second
	}

	syncTimeout, err := time.ParseDuration(v.GetString("sync_timeout"))
	if err != nil {
		syncTimeout = 30 * time.Second
	}

	cfg := &Config{
		Server: ServerConfig{
			Port:         v.GetInt("http_port"),
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
		},
		TTS: TTSConfig{
			ElevenLabsAPIKey:  v.GetString("elevenlabs_api_key"),
			DefaultVoiceID:    v.GetString("default_voice_id"),
			MaxSyncTextLength: v.GetInt("max_sync_text_length"),
			SyncTimeout:       syncTimeout,
		},
		Queue: QueueConfig{
			WorkerCount:       v.GetInt("worker_count"),
			MaxConcurrentJobs: v.GetInt("max_concurrent_jobs"),
		},
		Storage: StorageConfig{
			AudioStoragePath:  v.GetString("audio_storage_path"),
			JobRetentionHours: v.GetInt("job_retention_hours"),
		},
		Logging: LoggingConfig{
			Level:  v.GetString("log_level"),
			Format: v.GetString("log_format"),
		},
	}

	return cfg, nil
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	// ElevenLabs API key is required for production use
	// but we allow empty for testing/development
	return nil
}
