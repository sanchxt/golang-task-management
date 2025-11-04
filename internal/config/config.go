package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	DBPath string
}

func GetDefaultConfig() (*Config, error) {
	// get home dir
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// create taskflow directory in home
	taskflowDir := filepath.Join(homeDir, ".taskflow")
	dbPath := filepath.Join(taskflowDir, "tasks.db")

	return &Config{
		DBPath: dbPath,
	}, nil
}
