package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Success(t *testing.T) {
	os.Setenv("YANDEX_API_KEY", "test-key")
	os.Setenv("YANDEX_FOLDER_ID", "test-folder")
	defer os.Unsetenv("YANDEX_API_KEY")
	defer os.Unsetenv("YANDEX_FOLDER_ID")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "8080", cfg.Port)
	assert.True(t, cfg.PrometheusEnabled)
}

func TestLoad_MissingKeys(t *testing.T) {
	// очищаем переменные, если были
	os.Unsetenv("YANDEX_API_KEY")
	os.Unsetenv("YANDEX_FOLDER_ID")

	_, err := Load()
	assert.Error(t, err)
}
