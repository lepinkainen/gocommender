# 03 - Configuration Management

## Overview

Implement configuration management using github.com/spf13/viper following llm-shared guidelines, with environment variable loading and validation.

## Steps

### 1. Create Configuration Structure

Define `internal/config/config.go` with all required configuration:

```go
package config

import (
    "fmt"
    "time"
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
    APIKey string `mapstructure:"api_key"`
    Model  string `mapstructure:"model"`
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
```

### 2. Implement Configuration Loading

Create configuration loader with validation:

```go
// Load loads configuration from environment variables and files
func Load() (*Config, error) {
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
```

### 3. Environment Variable Mapping

Map environment variables to config structure:

```go
func setDefaults() {
    // Server defaults
    viper.SetDefault("server.host", "localhost")
    viper.SetDefault("server.port", "8080")

    // OpenAI defaults
    viper.SetDefault("openai.model", "gpt-4o")

    // Database defaults
    viper.SetDefault("database.path", "./gocommender.db")

    // Cache defaults
    viper.SetDefault("cache.ttl_success", "720h")  // 30 days
    viper.SetDefault("cache.ttl_failure", "168h")  // 7 days

    // Map environment variables
    viper.BindEnv("plex.url", "PLEX_URL")
    viper.BindEnv("plex.token", "PLEX_TOKEN")
    viper.BindEnv("openai.api_key", "OPENAI_API_KEY")
    viper.BindEnv("external.discogs_token", "DISCOGS_TOKEN")
    viper.BindEnv("external.lastfm_api_key", "LASTFM_API_KEY")
    viper.BindEnv("database.path", "DATABASE_PATH")
    viper.BindEnv("server.port", "PORT")
    viper.BindEnv("server.host", "HOST")
    viper.BindEnv("cache.ttl_success", "CACHE_TTL_SUCCESS")
    viper.BindEnv("cache.ttl_failure", "CACHE_TTL_FAILURE")
}
```

### 4. Configuration Validation

Implement validation for required fields:

```go
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
```

### 5. Database Initialization

Create database setup function in `internal/config/database.go`:

```go
package config

import (
    "database/sql"
    "fmt"
    "os"
    "path/filepath"

    _ "modernc.org/sqlite"
)

// InitDatabase initializes the SQLite database with schema
func InitDatabase(dbPath string) (*sql.DB, error) {
    // Ensure directory exists
    dir := filepath.Dir(dbPath)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create database directory: %w", err)
    }

    // Open database
    db, err := sql.Open("sqlite", dbPath)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }

    // Test connection
    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }

    // Create schema
    if err := createSchema(db); err != nil {
        return nil, fmt.Errorf("failed to create schema: %w", err)
    }

    return db, nil
}

func createSchema(db *sql.DB) error {
    schema := `-- Schema content from internal/models/schema.sql --`
    _, err := db.Exec(schema)
    return err
}
```

## Verification Steps

1. **Configuration Loading**:

   ```bash
   # Create test .env file
   echo "PLEX_URL=http://test.local" > .env.test
   echo "PLEX_TOKEN=test-token" >> .env.test
   echo "OPENAI_API_KEY=test-key" >> .env.test

   # Test configuration loading
   GOCOMMENDER_ENV_FILE=.env.test go run ./cmd/server -config-test
   ```

2. **Environment Variable Mapping**:

   ```bash
   # Test environment variable override
   PLEX_URL=http://override.local go run ./cmd/server -config-test
   ```

3. **Validation Testing**:

   ```bash
   # Test missing required config
   go run ./cmd/server -config-test  # Should fail with validation errors
   ```

4. **Database Initialization**:

   ```bash
   # Test database creation
   DATABASE_PATH=./test.db go run ./cmd/server -init-db-only
   sqlite3 ./test.db ".tables"  # Should show 'artists' table
   ```

## Dependencies

- Previous: `02_data_models.md` (Models for configuration)
- New dependency: `github.com/spf13/viper`
- Imports: `net/url`, `strings`, `path/filepath`, `os`

## Next Steps

Proceed to `04_musicbrainz_integration.md` to implement the MusicBrainz API client using the configuration system.

## Notes

- Configuration supports both environment variables and YAML files
- Validation ensures all required API keys are present before startup
- Database path can be configured for different environments
- TTL values are configurable for cache management
- Optional external APIs gracefully degrade when keys are missing
- Viper automatically handles type conversion for duration strings
