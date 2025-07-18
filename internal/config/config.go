package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Plex     PlexConfig     `mapstructure:"plex"`
	OpenAI   OpenAIConfig   `mapstructure:"openai"`
	External ExternalConfig `mapstructure:"external"`
	Database DatabaseConfig `mapstructure:"database"`
	Cache    CacheConfig    `mapstructure:"cache"`
}

// ServerConfig contains HTTP server settings
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
}

// PlexConfig contains Plex server settings
type PlexConfig struct {
	URL   string `mapstructure:"url"`
	Token string `mapstructure:"token"`
}

// OpenAIConfig contains OpenAI API settings
type OpenAIConfig struct {
	APIKey             string `mapstructure:"api_key"`
	Model              string `mapstructure:"model"`
	PromptTemplatePath string `mapstructure:"prompt_template_path"`
}

// ExternalConfig contains optional external API configurations
type ExternalConfig struct {
	DiscogsToken string `mapstructure:"discogs_token"`
	LastFMAPIKey string `mapstructure:"lastfm_api_key"`
}

// DatabaseConfig contains database settings
type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

// CacheConfig contains caching behavior settings
type CacheConfig struct {
	TTLSuccess time.Duration `mapstructure:"ttl_success"`
	TTLFailure time.Duration `mapstructure:"ttl_failure"`
}

// Load loads configuration from environment variables and files
func Load() (*Config, error) {
	// Load .env file if it exists (optional)
	if err := godotenv.Load(); err != nil {
		// .env file not found or unreadable - this is ok, continue with env vars
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Set environment variable prefix
	viper.SetEnvPrefix("GOCOMMENDER")
	viper.AutomaticEnv()

	// Set defaults
	setDefaults()

	// Read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate required fields
	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

func setDefaults() {
	// Server defaults
	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("server.port", "8080")

	// OpenAI defaults
	viper.SetDefault("openai.model", "gpt-4o")
	viper.SetDefault("openai.prompt_template_path", "./prompts/openai_recommendation.tmpl")

	// Database defaults
	viper.SetDefault("database.path", "./gocommender.db")

	// Cache defaults
	viper.SetDefault("cache.ttl_success", "720h") // 30 days
	viper.SetDefault("cache.ttl_failure", "168h") // 7 days

	// Map environment variables
	viper.BindEnv("plex.url", "PLEX_URL")
	viper.BindEnv("plex.token", "PLEX_TOKEN")
	viper.BindEnv("openai.api_key", "OPENAI_API_KEY")
	viper.BindEnv("openai.prompt_template_path", "OPENAI_PROMPT_TEMPLATE_PATH")
	viper.BindEnv("external.discogs_token", "DISCOGS_TOKEN")
	viper.BindEnv("external.lastfm_api_key", "LASTFM_API_KEY")
	viper.BindEnv("database.path", "DATABASE_PATH")
	viper.BindEnv("server.port", "PORT")
	viper.BindEnv("server.host", "HOST")
	viper.BindEnv("cache.ttl_success", "CACHE_TTL_SUCCESS")
	viper.BindEnv("cache.ttl_failure", "CACHE_TTL_FAILURE")
}

func validate(config *Config) error {
	var errors []string

	// Required fields
	if config.Plex.URL == "" {
		errors = append(errors, "PLEX_URL is required")
	}
	if config.Plex.Token == "" {
		errors = append(errors, "PLEX_TOKEN is required")
	}
	if config.OpenAI.APIKey == "" {
		errors = append(errors, "OPENAI_API_KEY is required")
	}

	// Validate URLs
	if config.Plex.URL != "" && !isValidURL(config.Plex.URL) {
		errors = append(errors, "PLEX_URL must be a valid URL")
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation errors: %s", strings.Join(errors, ", "))
	}

	return nil
}

func isValidURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}
