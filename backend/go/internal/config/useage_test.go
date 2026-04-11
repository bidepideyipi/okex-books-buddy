package config_test

import (
	"testing"

	"github.com/supermancell/okex-buddy/internal/config"
)

func TestLoadFromEnv(t *testing.T) {
	cfg := config.LoadFromEnv()
	t.Log(cfg)
	// 这里可以添加实际的测试逻辑
}
