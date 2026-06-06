package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Log      LogConfig      `mapstructure:"log"`
	CORS     CORSConfig     `mapstructure:"cors"`
}

type AppConfig struct {
	Name            string        `mapstructure:"name"`
	Env             string        `mapstructure:"env"`
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

type AuthConfig struct {
	AccessSecret          string        `mapstructure:"access_secret"`
	RefreshSecret         string        `mapstructure:"refresh_secret"`
	PasswordSecret        string        `mapstructure:"password_secret"`
	AccessTTL             time.Duration `mapstructure:"access_ttl"`
	RefreshTTL            time.Duration `mapstructure:"refresh_ttl"`
	AccessCookieName      string        `mapstructure:"access_cookie_name"`
	RefreshCookieName     string        `mapstructure:"refresh_cookie_name"`
	SessionCartCookieName string        `mapstructure:"session_cart_cookie_name"`
	CookieDomain          string        `mapstructure:"cookie_domain"`
	CookieSecure          bool          `mapstructure:"cookie_secure"`
	CookieHTTPOnly        bool          `mapstructure:"cookie_http_only"`
	CookieSameSite        string        `mapstructure:"cookie_same_site"`
}

type DatabaseConfig struct {
	DSN             string        `mapstructure:"dsn"`
	AutoMigrate     bool          `mapstructure:"auto_migrate"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

type LogConfig struct {
	Level       string `mapstructure:"level"`
	Encoding    string `mapstructure:"encoding"`
	Development bool   `mapstructure:"development"`
}

type CORSConfig struct {
	AllowedOrigins   []string      `mapstructure:"allowed_origins"`
	AllowedMethods   []string      `mapstructure:"allowed_methods"`
	AllowedHeaders   []string      `mapstructure:"allowed_headers"`
	ExposedHeaders   []string      `mapstructure:"exposed_headers"`
	AllowCredentials bool          `mapstructure:"allow_credentials"`
	MaxAge           time.Duration `mapstructure:"max_age"`
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./configs")
	v.AddConfigPath(".")

	v.SetEnvPrefix("MINI_STORE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		var configNotFound viper.ConfigFileNotFoundError
		if !errorAs(err, &configNotFound) {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	cfg := new(Config)
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.name", "mini-store-go-api")
	v.SetDefault("app.env", "development")
	v.SetDefault("app.host", "0.0.0.0")
	v.SetDefault("app.port", 8080)
	v.SetDefault("app.read_timeout", "10s")
	v.SetDefault("app.write_timeout", "10s")
	v.SetDefault("app.idle_timeout", "60s")
	v.SetDefault("app.shutdown_timeout", "10s")

	v.SetDefault("auth.access_secret", "change-me-access-secret")
	v.SetDefault("auth.refresh_secret", "change-me-refresh-secret")
	v.SetDefault("auth.password_secret", "change-me-password-secret")
	v.SetDefault("auth.access_ttl", "15m")
	v.SetDefault("auth.refresh_ttl", "720h")
	v.SetDefault("auth.access_cookie_name", "mini_store_access_token")
	v.SetDefault("auth.refresh_cookie_name", "mini_store_refresh_token")
	v.SetDefault("auth.session_cart_cookie_name", "session_cart_id")
	v.SetDefault("auth.cookie_domain", "")
	v.SetDefault("auth.cookie_secure", false)
	v.SetDefault("auth.cookie_http_only", true)
	v.SetDefault("auth.cookie_same_site", "lax")

	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.max_open_conns", 50)
	v.SetDefault("database.conn_max_lifetime", "30m")
	v.SetDefault("database.auto_migrate", false)

	v.SetDefault("redis.addr", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 10)

	v.SetDefault("log.level", "debug")
	v.SetDefault("log.encoding", "console")
	v.SetDefault("log.development", true)

	v.SetDefault("cors.allowed_origins", []string{"http://localhost:5173", "http://localhost:3000"})
	v.SetDefault("cors.allowed_methods", []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"})
	v.SetDefault("cors.allowed_headers", []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", "X-Request-Id"})
	v.SetDefault("cors.exposed_headers", []string{"X-Request-Id"})
	v.SetDefault("cors.allow_credentials", true)
	v.SetDefault("cors.max_age", "12h")
}

func errorAs(err error, target *viper.ConfigFileNotFoundError) bool {
	return errors.As(err, target)
}
