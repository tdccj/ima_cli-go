package config

import (
	"os"
	"path/filepath"
	"testing"
)

// setupEnv 设置环境变量并返回恢复函数。
func setupEnv(key, value string) func() {
	orig := os.Getenv(key)
	os.Setenv(key, value)
	return func() { os.Setenv(key, orig) }
}

func TestLoadFromEnv(t *testing.T) {
	// 优先使用环境变量
	defer setupEnv("IMA_OPENAPI_CLIENTID", "env-client")()
	defer setupEnv("IMA_OPENAPI_APIKEY", "env-key")()

	// 确保没有配置文件干扰
	defer setupEnv("HOME", "/nonexistent")()

	creds, err := Load()
	if err != nil {
		t.Fatalf("Load 应成功，但返回: %v", err)
	}
	if creds.ClientID != "env-client" {
		t.Errorf("ClientID = %q，期望 env-client", creds.ClientID)
	}
	if creds.APIKey != "env-key" {
		t.Errorf("APIKey = %q，期望 env-key", creds.APIKey)
	}
}

func TestLoadFromFallbackEnvNames(t *testing.T) {
	// 测试兼容旧名称
	defer setupEnv("IMA_CLIENT_ID", "old-client")()
	defer setupEnv("IMA_API_KEY", "old-key")()
	defer setupEnv("HOME", "/nonexistent")()

	creds, err := Load()
	if err != nil {
		t.Fatalf("Load 应成功，但返回: %v", err)
	}
	if creds.ClientID != "old-client" {
		t.Errorf("ClientID = %q，期望 old-client", creds.ClientID)
	}
	if creds.APIKey != "old-key" {
		t.Errorf("APIKey = %q，期望 old-key", creds.APIKey)
	}
}

func TestLoadMissingCredentials(t *testing.T) {
	// 不设置任何凭证
	defer setupEnv("IMA_OPENAPI_CLIENTID", "")()
	defer setupEnv("IMA_OPENAPI_APIKEY", "")()
	defer setupEnv("IMA_CLIENT_ID", "")()
	defer setupEnv("IMA_API_KEY", "")()
	defer setupEnv("HOME", "/nonexistent")()

	_, err := Load()
	if err == nil {
		t.Fatal("无凭证时应返回错误")
	}
}

func TestLoadFromConfigFile(t *testing.T) {
	// 创建临时配置文件
	dir, err := os.MkdirTemp("", "ima-config-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(dir)

	// 模拟 HOME
	defer setupEnv("HOME", dir)()

	// 清除环境变量
	defer setupEnv("IMA_OPENAPI_CLIENTID", "")()
	defer setupEnv("IMA_OPENAPI_APIKEY", "")()

	// 创建配置文件
	configDir := filepath.Join(dir, ".config/ima")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("创建配置目录失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "client_id"), []byte("file-client\n"), 0644); err != nil {
		t.Fatalf("写入 client_id 失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "api_key"), []byte("file-key\n"), 0644); err != nil {
		t.Fatalf("写入 api_key 失败: %v", err)
	}

	creds, err := Load()
	if err != nil {
		t.Fatalf("Load 应成功，但返回: %v", err)
	}
	if creds.ClientID != "file-client" {
		t.Errorf("ClientID = %q，期望 file-client", creds.ClientID)
	}
	if creds.APIKey != "file-key" {
		t.Errorf("APIKey = %q，期望 file-key", creds.APIKey)
	}
}

func TestEnvOverridesConfigFile(t *testing.T) {
	// 环境变量优先级高于配置文件
	dir, err := os.MkdirTemp("", "ima-config-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(dir)

	defer setupEnv("HOME", dir)()
	defer setupEnv("IMA_OPENAPI_CLIENTID", "env-client")()
	defer setupEnv("IMA_OPENAPI_APIKEY", "env-key")()

	// 创建冲突的配置文件
	configDir := filepath.Join(dir, ".config/ima")
	os.MkdirAll(configDir, 0755)
	os.WriteFile(filepath.Join(configDir, "client_id"), []byte("file-client\n"), 0644)
	os.WriteFile(filepath.Join(configDir, "api_key"), []byte("file-key\n"), 0644)

	creds, err := Load()
	if err != nil {
		t.Fatalf("Load 应成功，但返回: %v", err)
	}
	// 环境变量应优先
	if creds.ClientID != "env-client" {
		t.Errorf("环境变量应优先，ClientID = %q", creds.ClientID)
	}
	if creds.APIKey != "env-key" {
		t.Errorf("环境变量应优先，APIKey = %q", creds.APIKey)
	}
}
