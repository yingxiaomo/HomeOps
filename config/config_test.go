package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	os.Setenv("TG_BOT_TOKEN", "test_token")
	os.Setenv("ADMIN_ID", "123456")
	
	LoadConfig()
	
	assert.Equal(t, "test_token", AppConfig.BotToken)
	assert.Equal(t, int64(123456), AppConfig.AdminID)
}

func TestEnvHelpers(t *testing.T) {
	os.Setenv("TEST_INT", "123")
	os.Setenv("TEST_SLICE", "a,b, c ")

	assert.Equal(t, int64(123), getEnvAsInt("TEST_INT", 0))
	assert.Equal(t, int64(0), getEnvAsInt("NON_EXISTENT", 0))

	slice := getEnvAsSlice("TEST_SLICE")
	assert.Len(t, slice, 3)
	assert.Equal(t, "a", slice[0])
	assert.Equal(t, "b", slice[1])
	assert.Equal(t, "c", slice[2])
}
