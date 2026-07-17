// Package config 负责加载 IMA API 的认证凭证。
//
// 凭证优先级（高 → 低）：
//  1. 环境变量 IMA_OPENAPI_CLIENTID / IMA_OPENAPI_APIKEY
//  2. 环境变量 IMA_CLIENT_ID / IMA_API_KEY（兼容旧名称）
//  3. 配置文件 ~/.config/ima/client_id 和 ~/.config/ima/api_key
//
// 客户端 ID 和 API Key 均需要同时存在，缺少任一字段都会返回错误。
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Credentials 保存 IMA API 的认证信息。
type Credentials struct {
	ClientID string
	APIKey   string
}

// Load 按优先级尝试加载凭证，返回第一个成功获取的凭证对。
// 如果所有来源均无法获取完整凭证，返回错误。
func Load() (*Credentials, error) {
	// 第一优先级：环境变量
	clientID := os.Getenv("IMA_OPENAPI_CLIENTID")
	apiKey := os.Getenv("IMA_OPENAPI_APIKEY")

	// 第二优先级：兼容旧名称
	if clientID == "" {
		clientID = os.Getenv("IMA_CLIENT_ID")
	}
	if apiKey == "" {
		apiKey = os.Getenv("IMA_API_KEY")
	}

	// 第三优先级：配置文件
	if clientID == "" || apiKey == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			cid, _ := os.ReadFile(filepath.Join(home, ".config/ima/client_id"))
			ak, _ := os.ReadFile(filepath.Join(home, ".config/ima/api_key"))
			if len(cid) > 0 && len(ak) > 0 {
				clientID = strings.TrimSpace(string(cid))
				apiKey = strings.TrimSpace(string(ak))
			}
		}
	}

	if clientID == "" || apiKey == "" {
		return nil, fmt.Errorf("未找到 IMA 凭证。请设置环境变量 IMA_OPENAPI_CLIENTID/IMA_OPENAPI_APIKEY，或将凭证放在 ~/.config/ima/ 下")
	}

	return &Credentials{ClientID: clientID, APIKey: apiKey}, nil
}
