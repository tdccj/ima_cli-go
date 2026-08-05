package ima

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ima_cli-go/internal/config"
)

// testClient 创建一个测试用的 IMA 客户端，指向一个可控制的测试服务器。
func testClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	creds := &config.Credentials{ClientID: "test-client", APIKey: "test-key"}
	client := NewClient(creds)
	client.SetBaseURL(server.URL)
	// 移除 BaseURL 后的路径拼接问题：Post() 中拼接为 baseURL + "/" + apiPath
	return client, server
}

func TestPostSuccess(t *testing.T) {
	// 模拟 API 成功响应
	client, server := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		// 验证请求头
		if r.Header.Get("ima-openapi-clientid") != "test-client" {
			t.Error("缺少 clientid 头")
		}
		if r.Header.Get("ima-openapi-apikey") != "test-key" {
			t.Error("缺少 apikey 头")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Content-Type 不是 application/json")
		}
		if r.Method != "POST" {
			t.Errorf("请求方法应为 POST，得到 %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"msg":"成功","data":{"id":"abc123","name":"test"}}`))
	})
	defer server.Close()

	resp, err := client.Post("test/path", RequestBody{"limit": 10})
	if err != nil {
		t.Fatalf("Post 失败: %v", err)
	}
	if resp.Code != 0 {
		t.Errorf("Code = %d，期望 0", resp.Code)
	}
	if GetString(resp.Data, "id") != "abc123" {
		t.Errorf("data.id = %q，期望 abc123", GetString(resp.Data, "id"))
	}
	if GetString(resp.Data, "name") != "test" {
		t.Errorf("data.name = %q，期望 test", GetString(resp.Data, "name"))
	}
}

func TestPostBusinessError(t *testing.T) {
	// 模拟 API 业务错误
	client, server := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":110001,"msg":"参数非法","data":{}}`))
	})
	defer server.Close()

	_, err := client.Post("test/path", nil)
	if err == nil {
		t.Fatal("应返回业务错误，但未返回")
	}
}

func TestPostInvalidJSON(t *testing.T) {
	// 模拟响应不是合法 JSON
	client, server := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not-json`))
	})
	defer server.Close()

	_, err := client.Post("test/path", nil)
	if err == nil {
		t.Fatal("无效 JSON 应返回错误")
	}
}

func TestPostServerError(t *testing.T) {
	// 模拟服务端 500
	client, server := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`internal error`))
	})
	defer server.Close()

	_, err := client.Post("test/path", nil)
	if err == nil {
		t.Fatal("500 应返回错误")
	}
}

func TestPostRequestPath(t *testing.T) {
	// 验证请求路径拼接
	var capturedPath string
	client, server := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"msg":"ok","data":{}}`))
	})
	defer server.Close()

	client.Post("openapi/note/v1/list_notebook", nil)
	if capturedPath != "/openapi/note/v1/list_notebook" {
		t.Errorf("请求路径 = %q，期望 /openapi/note/v1/list_notebook", capturedPath)
	}
}

func TestHelperFunctions(t *testing.T) {
	// 测试 map 提取辅助函数
	data := map[string]any{
		"id":     "abc",
		"count":  42.0,
		"active": true,
		"items":  []any{"a", "b"},
	}

	if got := GetString(data, "id"); got != "abc" {
		t.Errorf("GetString = %q", got)
	}
	if got := GetInt64(data, "count"); got != 42 {
		t.Errorf("GetInt64 = %d", got)
	}
	if got := GetBool(data, "active"); got != true {
		t.Errorf("GetBool = %v", got)
	}
	if got := GetSlice(data, "items"); len(got) != 2 {
		t.Errorf("GetSlice 长度 = %d", len(got))
	}

	// 缺失 key 应返回零值
	if got := GetString(data, "missing"); got != "" {
		t.Errorf("缺失 key 应返回空字符串，得到 %q", got)
	}
}

func TestRequestBodyJSON(t *testing.T) {
	// 验证 RequestBody 的序列化
	body := RequestBody{
		"name":   "test",
		"limit":  10,
		"active": true,
	}
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	var parsed map[string]any
	json.Unmarshal(b, &parsed)
	if parsed["name"] != "test" {
		t.Errorf("name = %v", parsed["name"])
	}
	if parsed["limit"].(float64) != 10 {
		t.Errorf("limit = %v", parsed["limit"])
	}
}
