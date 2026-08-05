package ima

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"ima_cli-go/internal/config"
)

// DefaultBaseURL IMA OpenAPI 的基础地址。
const DefaultBaseURL = "https://ima.qq.com"

// Client 封装 IMA OpenAPI 的 HTTP 调用。
// 内部维护 http.Client，所有请求自动携带认证头。
type Client struct {
	BaseURL    string
	Creds      *config.Credentials
	httpClient *http.Client
}

// NewClient 创建一个新的 IMA API 客户端。
func NewClient(creds *config.Credentials) *Client {
	return &Client{
		BaseURL: DefaultBaseURL,
		Creds:   creds,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// SetBaseURL 设置自定义的基础 URL，用于测试或代理场景。
func (c *Client) SetBaseURL(url string) {
	c.BaseURL = url
}

// Post 发送一个 POST 请求到指定的 API 路径。
//   - apiPath：API 路径，如 "openapi/note/v1/list_notebook"
//   - body：请求体，会被序列化为 JSON
//
// 请求会自动添加认证头（ClientID/APIKey）和 skill 版本信息。
// 返回的 APIResponse 保证 code=0，否则返回错误。
func (c *Client) Post(apiPath string, body RequestBody) (*APIResponse, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/"+apiPath, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置认证头和内容类型
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ima-openapi-clientid", c.Creds.ClientID)
	req.Header.Set("ima-openapi-apikey", c.Creds.APIKey)

	// 附加 skill 版本信息，用于服务端兼容性判断
	home, _ := os.UserHomeDir()
	metaPath := filepath.Join(home, ".config/opencode/skills/ima-skill/meta.json")
	if metaBytes, err := os.ReadFile(metaPath); err == nil {
		var meta struct {
			Version string `json:"version"`
		}
		if json.Unmarshal(metaBytes, &meta) == nil && meta.Version != "" {
			req.Header.Set("ima-openapi-ctx", "skill_version="+meta.Version)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析统一响应格式
	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %s", string(respBody))
	}

	// 业务错误检查
	if apiResp.Code != 0 {
		return nil, fmt.Errorf("API 错误 (%d): %s", apiResp.Code, apiResp.Msg)
	}

	return &apiResp, nil
}

// GetInt64 从 map 中安全提取 int64 类型的值。
func GetInt64(m map[string]any, key string) int64 {
	v, _ := m[key].(float64)
	return int64(v)
}

// GetString 从 map 中安全提取 string 类型的值。
func GetString(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

// GetBool 从 map 中安全提取 bool 类型的值。
func GetBool(m map[string]any, key string) bool {
	v, _ := m[key].(bool)
	return v
}

// GetSlice 从 map 中安全提取 []any 类型的值。
func GetSlice(m map[string]any, key string) []any {
	v, _ := m[key].([]any)
	return v
}
