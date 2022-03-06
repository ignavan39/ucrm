package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/ignavan39/ucrm-go/pkg/pg"
	"gopkg.in/yaml.v2"
)

type JWTConfig struct {
	HashSalt       string        `env:"JWT_HASH_SALT"`
	SingingKey     string        `env:"JWT_SINGINGING_KEY"`
	ExpireDuration time.Duration `env:"JWT_EXPIRE_DURATION"`
}

type Config struct {
	Database pg.Config
	JWT      JWTConfig
	Cors     CorsConfig
}

func confFromFile(fileName string) (*CorsConfig, error) {
	log.Println(fmt.Sprintf("reading from %s", fileName))
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	var conf CorsConfig
	defer file.Close()
	if err := yaml.NewDecoder(file).Decode(&conf); err != nil {
		return nil, err
	}
	return &conf, nil
}

func GetConfig() (*Config, error) {

	port, err := strconv.ParseInt(os.Getenv("DATABASE_PORT"), 10, 16)
	if err != nil {
		return nil, err
	}

	expireDuration, err := time.ParseDuration(os.Getenv("JWT_EXPIRE_DURATION"))
	if err != nil {
		return nil, err
	}
	pgCong := pg.Config{
		Password: os.Getenv("DATABASE_PASS"),
		Host:     os.Getenv("DATABASE_HOST"),
		User:     os.Getenv("DATABASE_USER"),
		Port:     uint16(port),
		DB:       os.Getenv("DATABASE_NAME"),
	}
	cors, err := confFromFile("./usr/local/bin/app/develop.yml")
	if err != nil {
		return nil, err
	}

	return &Config{
		Database: pgCong,
		JWT: JWTConfig{
			HashSalt:       os.Getenv("JWT_HASH_SALT"),
			SingingKey:     os.Getenv("JWT_SINGINGING_KEY"),
			ExpireDuration: expireDuration,
		},
		Cors: *cors,
	}, nil
}
