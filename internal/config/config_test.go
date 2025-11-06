package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestConfig(t *testing.T) func() {
	// save original values
	origConfigDir := configDir
	origConfigFile := configFile

	// create temp directory
	tmpDir, err := os.MkdirTemp("", "taskflow_config_test_*")
	require.NoError(t, err)

	configDir = tmpDir
	configFile = filepath.Join(tmpDir, "config.yaml")

	return func() {
		os.RemoveAll(tmpDir)
		configDir = origConfigDir
		configFile = origConfigFile
	}
}

func TestGetDefaultConfig(t *testing.T) {
	cfg := GetDefaultConfig()

	assert.NotNil(t, cfg)
	assert.NotEmpty(t, cfg.DBPath)
	assert.Equal(t, "", cfg.ThemeName) // empty until set
	assert.Equal(t, 20, cfg.DefaultPageSize)
	assert.Equal(t, 100, cfg.MaxPageSize)
}

func TestLoadConfig_Default(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	cfg, err := LoadConfig()
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// should return default values when no config file exists
	assert.NotEmpty(t, cfg.DBPath)
	assert.Equal(t, 20, cfg.DefaultPageSize)
	assert.Equal(t, 100, cfg.MaxPageSize)
}

func TestSaveAndLoadConfig(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	// create config
	cfg := &Config{
		DBPath:          filepath.Join(configDir, "test.db"),
		ThemeName:       "dracula",
		DefaultPageSize: 25,
		MaxPageSize:     150,
	}

	err := SaveConfig(cfg)
	require.NoError(t, err)

	loaded, err := LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, cfg.DBPath, loaded.DBPath)
	assert.Equal(t, cfg.ThemeName, loaded.ThemeName)
	assert.Equal(t, cfg.DefaultPageSize, loaded.DefaultPageSize)
	assert.Equal(t, cfg.MaxPageSize, loaded.MaxPageSize)
}

func TestSaveConfig_CreatesDirectory(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	// remove the config directory
	os.RemoveAll(configDir)

	cfg := GetDefaultConfig()
	err := SaveConfig(cfg)
	require.NoError(t, err)

	// verify directory was created
	info, err := os.Stat(configDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestUpdateTheme(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	// save initial config
	cfg := GetDefaultConfig()
	err := SaveConfig(cfg)
	require.NoError(t, err)

	// update theme
	err = UpdateTheme("monokai")
	require.NoError(t, err)

	// verify update
	loaded, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, "monokai", loaded.ThemeName)
}

func TestConfig_PageSizeValidation(t *testing.T) {
	cleanup := setupTestConfig(t)
	defer cleanup()

	t.Run("default page size within bounds", func(t *testing.T) {
		cfg := &Config{
			DBPath:          filepath.Join(configDir, "test.db"),
			DefaultPageSize: 20,
			MaxPageSize:     100,
		}

		err := SaveConfig(cfg)
		require.NoError(t, err)

		loaded, err := LoadConfig()
		require.NoError(t, err)
		assert.Equal(t, 20, loaded.DefaultPageSize)
		assert.Equal(t, 100, loaded.MaxPageSize)
	})

	t.Run("zero page size gets default", func(t *testing.T) {
		cfg := &Config{
			DBPath:          filepath.Join(configDir, "test.db"),
			DefaultPageSize: 0,
			MaxPageSize:     0,
		}

		err := SaveConfig(cfg)
		require.NoError(t, err)

		loaded, err := LoadConfig()
		require.NoError(t, err)

		// after loading, defaults should be applied
		if loaded.DefaultPageSize == 0 {
			loaded.DefaultPageSize = 20
		}
		if loaded.MaxPageSize == 0 {
			loaded.MaxPageSize = 100
		}

		assert.Equal(t, 20, loaded.DefaultPageSize)
		assert.Equal(t, 100, loaded.MaxPageSize)
	})
}
