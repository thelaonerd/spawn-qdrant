package config

import (
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	RestPort int
	GrpcPort int
}

func LoadConfig() *Config {
	_ = godotenv.Load() // Ignore error if .env doesn't exist

	return &Config{
		RestPort: viper.GetInt("rest-port"),
		GrpcPort: viper.GetInt("grpc-port"),
	}
}
