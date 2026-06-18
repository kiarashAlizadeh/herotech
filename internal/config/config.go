package config

import (
	"log"
	"sync"

	"github.com/spf13/viper"
)

type Config struct {
	// Server
	ServerPort  string `mapstructure:"SERVER_PORT"`
	Environment string `mapstructure:"ENVIRONMENT"`

	// Database
	DBHost     string `mapstructure:"DB_HOST"`
	DBUser     string `mapstructure:"DB_USER"`
	DBPassword string `mapstructure:"DB_PASSWORD"`
	DBPort     string `mapstructure:"DB_PORT"`
	DBName     string `mapstructure:"DB_NAME"`
}

var (
	once     sync.Once
	instance *Config
)

// Load loads configuration using the Singleton pattern.
func Load() (*Config, error) {
	var err error
	once.Do(func() {
		instance, err = loadConfig()
	})
	if err != nil {
		return nil, err
	}
	return instance, nil
}

// loadConfig performs the actual configuration loading.
func loadConfig() (*Config, error) {
	// Configure for .env file
	viper.SetConfigFile(".env") // Use .env explicitly
	viper.SetConfigType("env")  // Parse as key=value format
	viper.AddConfigPath(".")    // Look in current directory

	// Also read environment variables (overrides .env if set)
	viper.AutomaticEnv()

	_ = viper.BindEnv("SERVER_PORT")
	_ = viper.BindEnv("ENVIRONMENT")
	_ = viper.BindEnv("DB_HOST")
	_ = viper.BindEnv("DB_USER")
	_ = viper.BindEnv("DB_PASSWORD")
	_ = viper.BindEnv("DB_PORT")
	_ = viper.BindEnv("DB_NAME")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Printf("ℹ️ Running without .env file (Using system environment variables)")
		} else {
			log.Printf("⚠️ Error reading config file: %v", err)
		}
	}

	var cfg Config

	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil

}
