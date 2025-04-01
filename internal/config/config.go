package config

import (
	"bufio"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env        string `env:"ENV" envconfig:"ENV"`
	ServerPort int    `env:"SERVER_PORT" envconfig:"SERVER_PORT"`
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

var ErrInvalidString = errors.New("invalid string")
var ErrFileFormat = errors.New("incorrect file format")

func LoadEnv() error {
	filePath := fetchConfigPath()

	if filepath.Ext(filePath) != ".env" {
		return ErrFileFormat
	}
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return ErrInvalidString
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		os.Setenv(key, value)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
