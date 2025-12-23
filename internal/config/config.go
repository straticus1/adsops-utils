package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	// Server
	Environment string `mapstructure:"environment"`
	Port        string `mapstructure:"port"`
	LogLevel    string `mapstructure:"log_level"`

	// Database
	Database DatabaseConfig `mapstructure:"database"`

	// Redis
	Redis RedisConfig `mapstructure:"redis"`

	// JWT
	JWT JWTConfig `mapstructure:"jwt"`

	// OAuth2 Providers
	OAuth2 OAuth2Config `mapstructure:"oauth2"`

	// AWS
	AWS AWSConfig `mapstructure:"aws"`

	// Email
	Email EmailConfig `mapstructure:"email"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DBName       string `mapstructure:"dbname"`
	SSLMode      string `mapstructure:"sslmode"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

// DSN returns the database connection string
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// Addr returns the Redis address
func (r *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	SecretKey            string `mapstructure:"secret_key"`
	AccessTokenDuration  int    `mapstructure:"access_token_duration"`  // minutes
	RefreshTokenDuration int    `mapstructure:"refresh_token_duration"` // days
	Issuer               string `mapstructure:"issuer"`
}

// OAuth2Config holds OAuth2 provider configurations
type OAuth2Config struct {
	AfterDark OAuth2Provider `mapstructure:"afterdark"`
	Google    OAuth2Provider `mapstructure:"google"`
}

// OAuth2Provider holds configuration for a single OAuth2 provider
type OAuth2Provider struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url"`
	AuthURL      string `mapstructure:"auth_url"`
	TokenURL     string `mapstructure:"token_url"`
	UserInfoURL  string `mapstructure:"userinfo_url"`
	Scopes       string `mapstructure:"scopes"`
}

// GetScopes returns scopes as a slice
func (o *OAuth2Provider) GetScopes() []string {
	if o.Scopes == "" {
		return []string{}
	}
	return strings.Split(o.Scopes, ",")
}

// AWSConfig holds AWS configuration
type AWSConfig struct {
	Region          string `mapstructure:"region"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	S3Bucket        string `mapstructure:"s3_bucket"`
	SQSQueueURL     string `mapstructure:"sqs_queue_url"`
}

// EmailConfig holds email configuration
type EmailConfig struct {
	From        string `mapstructure:"from"`
	ReplyTo     string `mapstructure:"reply_to"`
	BaseURL     string `mapstructure:"base_url"` // For approval links
	CompanyName string `mapstructure:"company_name"`
}

// Load loads configuration from environment and config files
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/adsops-utils")

	// Set defaults
	viper.SetDefault("environment", "development")
	viper.SetDefault("port", "8080")
	viper.SetDefault("log_level", "info")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("jwt.access_token_duration", 15)
	viper.SetDefault("jwt.refresh_token_duration", 7)
	viper.SetDefault("jwt.issuer", "changes.afterdarksys.com")
	viper.SetDefault("aws.region", "us-east-1")

	// Environment variable bindings
	viper.SetEnvPrefix("ADSOPS")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found is OK, we use env vars
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &cfg, nil
}
