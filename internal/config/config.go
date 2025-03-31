package config

import (
	"flag"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	ServerPort int `env:"SERVER_PORT" env-default:"8080"`
	DataBase   DatabaseConfig

	ConnectionPool ConnectionPoolConfig
}

type DatabaseConfig struct {
	URL string `env:"DATABASE_URL" env-required:"true"`
}

type ConnectionPoolConfig struct {
	MaxOpenConns int           `env:"MAX_OPEN_CONNS" env-default:"25"`
	MaxIdleConns int           `env:"MAX_IDLE_CONNS" env-default:"25"`
	MaxLifetime  time.Duration `env:"MAX_LIFETIME" env-default:"300s"`
}

func MustLoad() *Config {
	configPath := fetchConfigPath()
	if configPath == "" {
		panic("config path is empty")
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("config file does not exist: " + configPath)
	}
	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic("cannot read config: " + err.Error())
	}
	return &cfg
}

func fetchConfigPath() string {
	var res string
	flag.StringVar(&res, "config", "", "path to config file")
	flag.Parse()
	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}
	return res

}
