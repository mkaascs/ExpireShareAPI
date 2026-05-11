package config

import (
	"expire-share/internal/lib/sizes"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

const (
	EnvLocal = "local"
	EnvDev   = "dev"
	EnvProd  = "prod"
)

type Config struct {
	Env                string `yaml:"env" env-default:"local"`
	DbHost             string `yaml:"db_host" env-required:"true"`
	DbPassword         string `yaml:"-" env:"MYSQL_ROOT_PASSWORD" env-required:"true"`
	DbConnectionString string `yaml:"-"`
	Storage            `yaml:"storage"`
	HttpServer         `yaml:"http_server"`
	Service            `yaml:"service"`
	RateLimiter        `yaml:"rate_limiter"`
	AuthService        `yaml:"auth_service"`
	Redis              `yaml:"redis"`
}

type Storage struct {
	Type string `yaml:"type" env-default:"local"`
	Path string `yaml:"path" env-required:"true"`
}

type HttpServer struct {
	Port         int           `yaml:"port" env-required:"true"`
	WriteTimeout time.Duration `yaml:"write_timeout" env-default:"5s"`
	ReadTimeout  time.Duration `yaml:"read_timeout" env-default:"10s"`
	IdleTimeout  time.Duration `yaml:"idle_timeout" env-default:"60s"`
	CORS         `yaml:"cors"`
}

type Redis struct {
	Addr        string        `yaml:"addr" env-required:"true"`
	Password    string        `yaml:"-" env:"REDIS_PASSWORD" env-required:"true"`
	DB          int           `yaml:"db" env-default:"0"`
	DialTimeout time.Duration `yaml:"dial_timeout" env-default:"10s"`
	Timeout     time.Duration `yaml:"read_timeout" env-default:"5s"`
	MaxRetries  int           `yaml:"max_retries" env-default:"1"`
}

type RateLimiterParams struct {
	Field         string        `yaml:"field" env-default:""`
	MaxAttempts   int           `yaml:"max_attempts" env-default:"5"`
	Window        time.Duration `yaml:"window" env-default:"20m"`
	BlockDuration time.Duration `yaml:"block_duration" env-default:"15m"`
}

type RateLimiter struct {
	Admin    RateLimiterParams `yaml:"admin"`
	Files    RateLimiterParams `yaml:"files"`
	Register RateLimiterParams `yaml:"register"`
}

type CORS struct {
	AllowedOrigins     []string `yaml:"-"`
	AllowedOriginsEnv  string   `yaml:"-" env:"CORS_ALLOWED_ORIGINS" env-required:"true"`
	AllowedCredentials bool     `yaml:"allow_credentials" env-default:"true"`
	MaxAge             int      `yaml:"max_age" env-default:"86400"`
}

type AuthService struct {
	Addr string `yaml:"addr" env-required:"true"`
}

type Service struct {
	DefaultTtl      time.Duration `yaml:"default_ttl" env-default:"1h"`
	MaxDownloads    int16         `yaml:"default_max_downloads" env-default:"1"`
	AliasLength     int16         `yaml:"alias_length" env-default:"6"`
	FileWorkerDelay time.Duration `yaml:"file_worker_delay" env-default:"5m"`
	Permissions     `yaml:"permissions"`
}

type Permissions struct {
	MaxFilesSizeForUser        string `yaml:"max_files_size_for_user" env-default:"500mb"`
	MaxFilesSizeForVip         string `yaml:"max_files_size_for_vip" env-default:"2gb"`
	MaxUploadedFiles           int    `yaml:"max_uploaded_files" env-default:"50"`
	MaxFilesSizeForUserInBytes int64
	MaxFilesSizeForVipInBytes  int64
}

func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		log.Fatal(err)
	}

	return cfg
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		return nil, fmt.Errorf("env variable CONFIG_PATH not found")
	}

	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file %s does not exist", cfgPath)
	}

	var cfg Config
	if err := cleanenv.ReadConfig(cfgPath, &cfg); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	origins := strings.Split(cfg.AllowedOriginsEnv, ",")
	for index := range origins {
		origins[index] = strings.TrimSpace(origins[index])
	}

	cfg.AllowedOrigins = origins

	if err := setPermissionsConfig(&cfg); err != nil {
		return nil, err
	}

	cfg.DbConnectionString = fmt.Sprintf(
		"root:%s@tcp(%s)/ExpireShare?charset=utf8&parseTime=True",
		cfg.DbPassword,
		cfg.DbHost)

	return &cfg, nil
}

func setPermissionsConfig(cfg *Config) error {
	const errMsg = "failed to parse max file size in config"

	bytes, err := sizes.ToBytes(cfg.MaxFilesSizeForUser)
	if err != nil {
		return fmt.Errorf("%s: %w", errMsg, err)
	}

	cfg.MaxFilesSizeForUserInBytes = bytes

	bytes, err = sizes.ToBytes(cfg.MaxFilesSizeForVip)
	if err != nil {
		return fmt.Errorf("%s: %w", errMsg, err)
	}

	cfg.MaxFilesSizeForVipInBytes = bytes
	return nil
}
