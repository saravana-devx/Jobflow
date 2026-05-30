package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds every environment variable the app reads. It is the single
// source of truth — no other package should call viper directly.
type Config struct {
	// Postgres
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string

	// JWT
	AccessSecret string
	JWTIssuer    string
	JWTAudience  string

	// Redis
	RedisAddr     string
	RedisPassword string

	// RabbitMQ
	RabbitMQHost  string
	RabbitMQUser  string
	RabbitMQPass  string
	RabbitMQVHost string
}

var cfg *Config

// Load reads .env via viper, populates the singleton Config, validates that
// required values are present, and applies defaults. Call it once from main()
// before anything touches Get().
func Load() error {
	viper.AddConfigPath(".")
	viper.SetConfigType("env")
	viper.SetConfigName(".env")
	viper.AutomaticEnv() // real env vars (e.g. in containers) override .env

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	c := &Config{
		PostgresHost:     viper.GetString("POSTGRES_HOST"),
		PostgresPort:     viper.GetString("POSTGRES_PORT"),
		PostgresUser:     viper.GetString("POSTGRES_USER"),
		PostgresPassword: viper.GetString("POSTGRES_PASSWORD"),
		PostgresDB:       viper.GetString("POSTGRES_DB"),
		AccessSecret:     viper.GetString("ACCESS_SECRET"),
		JWTIssuer:        viper.GetString("JWT_ISSUER"),
		JWTAudience:      viper.GetString("JWT_AUDIENCE"),
		RedisAddr:        viper.GetString("REDIS_ADDR"),
		RedisPassword:    viper.GetString("REDIS_PASSWORD"),
		RabbitMQHost:     viper.GetString("RABBITMQ_HOST"),
		RabbitMQUser:     viper.GetString("RABBITMQ_DEFAULT_USER"),
		RabbitMQPass:     viper.GetString("RABBITMQ_DEFAULT_PASS"),
		RabbitMQVHost:    viper.GetString("RABBITMQ_DEFAULT_VHOST"),
	}

	if c.RedisAddr == "" {
		c.RedisAddr = "localhost:6379"
	}

	if err := c.validate(); err != nil {
		return err
	}

	cfg = c
	return nil
}

// validate fails fast if any required key is missing, listing all of them at
// once instead of panicking on the first. REDIS_PASSWORD is intentionally
// optional (local dev may run Redis without auth).
func (c *Config) validate() error {
	required := map[string]string{
		"POSTGRES_HOST":          c.PostgresHost,
		"POSTGRES_PORT":          c.PostgresPort,
		"POSTGRES_USER":          c.PostgresUser,
		"POSTGRES_PASSWORD":      c.PostgresPassword,
		"POSTGRES_DB":            c.PostgresDB,
		"ACCESS_SECRET":          c.AccessSecret,
		"JWT_ISSUER":             c.JWTIssuer,
		"JWT_AUDIENCE":           c.JWTAudience,
		"RABBITMQ_HOST":          c.RabbitMQHost,
		"RABBITMQ_DEFAULT_USER":  c.RabbitMQUser,
		"RABBITMQ_DEFAULT_PASS":  c.RabbitMQPass,
		"RABBITMQ_DEFAULT_VHOST": c.RabbitMQVHost,
	}

	var missing []string
	for key, val := range required {
		if val == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required config: %s", strings.Join(missing, ", "))
	}
	return nil
}

func (c *Config) RabbitMQURL() string {
	return fmt.Sprintf("amqp://%s:%s@%s:5672/%s",
		c.RabbitMQUser,
		c.RabbitMQPass,
		c.RabbitMQHost,
		c.RabbitMQVHost,
	)
}

// Get returns the loaded config. It panics if Load has not run yet — that's a
// wiring bug (programming error), not a runtime condition to handle.
func Get() *Config {
	if cfg == nil {
		panic("config.Get called before config.Load")
	}
	return cfg
}
