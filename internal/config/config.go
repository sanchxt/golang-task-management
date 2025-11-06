package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	DBPath          string `mapstructure:"db_path"`
	ThemeName       string `mapstructure:"theme_name"`
	DefaultPageSize int    `mapstructure:"default_page_size"`
	MaxPageSize     int    `mapstructure:"max_page_size"`
}

var (
	configDir  string
	configFile string
)

func init() {
	// get home dir
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("failed to get home directory: %v", err))
	}

	configDir = filepath.Join(homeDir, ".taskflow")
	configFile = filepath.Join(configDir, "config.yaml")
}

func GetConfigDir() string {
	return configDir
}

func GetConfigFile() string {
	return configFile
}

func ConfigExists() bool {
	_, err := os.Stat(configFile)
	return err == nil
}

func EnsureConfigDir() error {
	return os.MkdirAll(configDir, 0755)
}

// loads config from file
func LoadConfig() (*Config, error) {
	if err := EnsureConfigDir(); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	if !ConfigExists() {
		return GetDefaultConfig(), nil
	}

	// setup viper
	viper.SetConfigFile(configFile)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// unmarshal into config struct
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// apply defaults if not set
	if cfg.DBPath == "" {
		cfg.DBPath = filepath.Join(configDir, "tasks.db")
	}
	if cfg.DefaultPageSize == 0 {
		cfg.DefaultPageSize = 20
	}
	if cfg.MaxPageSize == 0 {
		cfg.MaxPageSize = 100
	}

	return &cfg, nil
}

// saves config to file
func SaveConfig(cfg *Config) error {
	if err := EnsureConfigDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	viper.Set("db_path", cfg.DBPath)
	viper.Set("theme_name", cfg.ThemeName)
	viper.Set("default_page_size", cfg.DefaultPageSize)
	viper.Set("max_page_size", cfg.MaxPageSize)

	if err := viper.WriteConfigAs(configFile); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// returns default config
func GetDefaultConfig() *Config {
	return &Config{
		DBPath:          filepath.Join(configDir, "tasks.db"),
		ThemeName:       "",
		DefaultPageSize: 20,
		MaxPageSize:     100,
	}
}

// updates theme in config file
func UpdateTheme(themeName string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cfg.ThemeName = themeName
	return SaveConfig(cfg)
}
