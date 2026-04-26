// Package config provides configuration management using Viper.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Server    ServerConfig
	TTS       TTSConfig
	Queue     QueueConfig
	Storage   StorageConfig
	Logging   LoggingConfig
	Providers ProvidersConfig
}

// ProvidersConfig holds configuration for all TTS providers.
type ProvidersConfig struct {
	Default string           `mapstructure:"default"`
	List    []ProviderConfig `mapstructure:"list"`
}

// ProviderConfig holds configuration for a single TTS provider.
type ProviderConfig struct {
	Name           string        `mapstructure:"name"`
	Type           string        `mapstructure:"type"`
	MaxConcurrent  int           `mapstructure:"max_concurrent"`
	Timeout        time.Duration `mapstructure:"timeout"`
	APIKey         string        `mapstructure:"api_key"`          // For elevenlabs
	ModelID        string        `mapstructure:"model_id"`         // For elevenlabs (default model)
	BaseURL        string        `mapstructure:"base_url"`         // For selfhosted
	TTSEndpoint    string        `mapstructure:"tts_endpoint"`     // For selfhosted
	VoicesEndpoint string        `mapstructure:"voices_endpoint"`  // For selfhosted
	HealthEndpoint string        `mapstructure:"health_endpoint"`  // For selfhosted
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

// Load loads configuration from config file and environment variables.
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "60s")
	v.SetDefault("server.write_timeout", "60s")
	v.SetDefault("tts.default_voice_id", "pNInz6obpgDQGcFmaJgB")
	v.SetDefault("tts.max_sync_text_length", 5000)
	v.SetDefault("tts.sync_timeout", "30s")
	v.SetDefault("queue.worker_count", 4)
	v.SetDefault("queue.max_concurrent_jobs", 100)
	v.SetDefault("storage.audio_storage_path", "./audio_cache")
	v.SetDefault("storage.job_retention_hours", 24)
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")

	// Try to read config file
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

	if err := v.ReadInConfig(); err != nil {
		// Config file is optional - fall back to env vars and defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Read from environment (overrides config file)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Also support legacy flat env vars for backwards compatibility
	legacyEnvMappings := map[string]string{
		"HTTP_PORT":           "server.port",
		"HTTP_READ_TIMEOUT":   "server.read_timeout",
		"HTTP_WRITE_TIMEOUT":  "server.write_timeout",
		"ELEVENLABS_API_KEY":  "tts.elevenlabs_api_key",
		"DEFAULT_VOICE_ID":    "tts.default_voice_id",
		"MAX_SYNC_TEXT_LENGTH": "tts.max_sync_text_length",
		"SYNC_TIMEOUT":        "tts.sync_timeout",
		"WORKER_COUNT":        "queue.worker_count",
		"MAX_CONCURRENT_JOBS": "queue.max_concurrent_jobs",
		"AUDIO_STORAGE_PATH":  "storage.audio_storage_path",
		"JOB_RETENTION_HOURS": "storage.job_retention_hours",
		"LOG_LEVEL":           "logging.level",
		"LOG_FORMAT":          "logging.format",
	}
	for envKey, configKey := range legacyEnvMappings {
		if val := os.Getenv(envKey); val != "" {
			v.Set(configKey, val)
		}
	}

	// Parse durations from strings
	readTimeout, err := time.ParseDuration(v.GetString("server.read_timeout"))
	if err != nil {
		readTimeout = 60 * time.Second
	}

	writeTimeout, err := time.ParseDuration(v.GetString("server.write_timeout"))
	if err != nil {
		writeTimeout = 60 * time.Second
	}

	syncTimeout, err := time.ParseDuration(v.GetString("tts.sync_timeout"))
	if err != nil {
		syncTimeout = 30 * time.Second
	}

	cfg := &Config{
		Server: ServerConfig{
			Port:         v.GetInt("server.port"),
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
		},
		TTS: TTSConfig{
			ElevenLabsAPIKey:  expandEnvVars(v.GetString("tts.elevenlabs_api_key")),
			DefaultVoiceID:    v.GetString("tts.default_voice_id"),
			MaxSyncTextLength: v.GetInt("tts.max_sync_text_length"),
			SyncTimeout:       syncTimeout,
		},
		Queue: QueueConfig{
			WorkerCount:       v.GetInt("queue.worker_count"),
			MaxConcurrentJobs: v.GetInt("queue.max_concurrent_jobs"),
		},
		Storage: StorageConfig{
			AudioStoragePath:  v.GetString("storage.audio_storage_path"),
			JobRetentionHours: v.GetInt("storage.job_retention_hours"),
		},
		Logging: LoggingConfig{
			Level:  v.GetString("logging.level"),
			Format: v.GetString("logging.format"),
		},
	}

	// Load providers configuration
	if err := loadProvidersConfig(v, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// loadProvidersConfig loads the providers section from viper.
func loadProvidersConfig(v *viper.Viper, cfg *Config) error {
	cfg.Providers.Default = v.GetString("providers.default")

	// Get the providers list
	providersRaw := v.Get("providers.list")
	if providersRaw == nil {
		// No providers configured - create default from legacy config
		if cfg.TTS.ElevenLabsAPIKey != "" {
			cfg.Providers.Default = "elevenlabs"
			cfg.Providers.List = []ProviderConfig{
				{
					Name:          "elevenlabs",
					Type:          "elevenlabs",
					APIKey:        cfg.TTS.ElevenLabsAPIKey,
					MaxConcurrent: 4,
					Timeout:       30 * time.Second,
				},
			}
		}
		return nil
	}

	// Parse providers list
	providersList, ok := providersRaw.([]interface{})
	if !ok {
		return fmt.Errorf("providers.list must be an array")
	}

	for _, p := range providersList {
		providerMap, ok := p.(map[string]interface{})
		if !ok {
			return fmt.Errorf("each provider must be an object")
		}

		pc := ProviderConfig{
			Name:           getString(providerMap, "name"),
			Type:           getString(providerMap, "type"),
			MaxConcurrent:  getInt(providerMap, "max_concurrent", 4),
			Timeout:        getDuration(providerMap, "timeout", 30*time.Second),
			APIKey:         expandEnvVars(getString(providerMap, "api_key")),
			ModelID:        getString(providerMap, "model_id"),
			BaseURL:        getString(providerMap, "base_url"),
			TTSEndpoint:    getString(providerMap, "tts_endpoint"),
			VoicesEndpoint: getString(providerMap, "voices_endpoint"),
			HealthEndpoint: getString(providerMap, "health_endpoint"),
		}

		// Set defaults for selfhosted endpoints
		if pc.Type == "selfhosted" {
			if pc.TTSEndpoint == "" {
				pc.TTSEndpoint = "/api/v1/tts"
			}
			if pc.VoicesEndpoint == "" {
				pc.VoicesEndpoint = "/api/v1/models"
			}
			if pc.HealthEndpoint == "" {
				pc.HealthEndpoint = "/api/v1/health"
			}
		}

		cfg.Providers.List = append(cfg.Providers.List, pc)
	}

	return nil
}

// expandEnvVars expands ${VAR} syntax in strings.
func expandEnvVars(s string) string {
	return os.Expand(s, os.Getenv)
}

// getString safely gets a string from a map.
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// getInt safely gets an int from a map with a default.
func getInt(m map[string]interface{}, key string, defaultVal int) int {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case int64:
			return int(val)
		case float64:
			return int(val)
		}
	}
	return defaultVal
}

// getDuration safely gets a duration from a map with a default.
func getDuration(m map[string]interface{}, key string, defaultVal time.Duration) time.Duration {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			if d, err := time.ParseDuration(s); err == nil {
				return d
			}
		}
	}
	return defaultVal
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	// Validate providers configuration
	if err := c.Providers.Validate(); err != nil {
		return err
	}
	return nil
}

// Validate validates the providers configuration.
func (p *ProvidersConfig) Validate() error {
	// Must have at least one provider configured
	if len(p.List) == 0 {
		return fmt.Errorf("at least one provider must be configured")
	}

	// Check for duplicate provider names
	names := make(map[string]bool)
	for _, provider := range p.List {
		if provider.Name == "" {
			return fmt.Errorf("provider name cannot be empty")
		}
		if provider.Type == "" {
			return fmt.Errorf("provider %q must have a type", provider.Name)
		}
		if names[provider.Name] {
			return fmt.Errorf("duplicate provider name: %q", provider.Name)
		}
		names[provider.Name] = true
	}

	// Default provider must exist in the list
	if p.Default == "" {
		return fmt.Errorf("default provider must be specified")
	}
	if !names[p.Default] {
		return fmt.Errorf("default provider %q not found in providers list", p.Default)
	}

	return nil
}
