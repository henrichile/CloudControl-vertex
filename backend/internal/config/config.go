package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Port         string
	DatabasePath string
	DockerHost   string
	OllamaHost   string
	OllamaModel  string
	JWTSecret    string
}

func Load() *Config {
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("DATABASE_PATH", "./cloudcontrol.db")
	viper.SetDefault("DOCKER_HOST", "unix:///var/run/docker.sock")
	viper.SetDefault("OLLAMA_HOST", "http://localhost:11434")
	viper.SetDefault("OLLAMA_MODEL", "llama3")
	viper.SetDefault("JWT_SECRET", "change-me-in-production")

	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	_ = viper.ReadInConfig()

	return &Config{
		Port:         viper.GetString("PORT"),
		DatabasePath: viper.GetString("DATABASE_PATH"),
		DockerHost:   viper.GetString("DOCKER_HOST"),
		OllamaHost:   viper.GetString("OLLAMA_HOST"),
		OllamaModel:  viper.GetString("OLLAMA_MODEL"),
		JWTSecret:    viper.GetString("JWT_SECRET"),
	}
}
