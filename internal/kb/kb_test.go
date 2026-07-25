package kb

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ima_cli-go/internal/config"
	"ima_cli-go/internal/ima"
)

// testClient 创建一个指向测试服务器的 IMA 客户端。
func testClient(t *testing.T, handler http.HandlerFunc) *ima.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	creds := &config.Credentials{ClientID: "t", APIKey: "t"}
	client := ima.NewClient(creds)
	client.SetBaseURL(server.URL)
	t.Cleanup(server.Close)
	return client
}

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		file string
		want string
	}{
		{"report.pdf", "application/pdf"},
		{"doc.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"slide.pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation"},
		{"data.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{"notes.md", "text/markdown"},
		{"image.png", "image/png"},
		{"photo.jpg", "image/jpeg"},
		{"photo.jpeg", "image/jpeg"},
		{"icon.webp", "image/webp"},
		{"readme.txt", "text/plain"},
		{"mindmap.xmind", "application/x-xmind"},
		{"audio.mp3", "audio/mpeg"},
		{"unknown.xyz", "application/octet-stream"},
		{"noext", "application/octet-stream"},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			got := detectContentType(tt.file)
			if got != tt.want {
				t.Errorf("detectContentType(%q) = %q，期望 %q", tt.file, got, tt.want)
			}
		})
	}
}

func TestDetectMediaType(t *testing.T) {
	tests := []struct {
		file string
		want int32
	}{
		{"f.pdf", 1},
		{"f.doc", 3},
		{"f.docx", 3},
		{"f.ppt", 4},
		{"f.pptx", 4},
		{"f.xls", 5},
		{"f.xlsx", 5},
		{"f.csv", 5},
		{"f.md", 7},
		{"f.png", 9},
		{"f.jpg", 9},
		{"f.jpeg", 9},
		{"f.webp", 9},
		{"f.txt", 13},
		{"f.xmind", 14},
		{"f.mp3", 15},
		{"f.m4a", 15},
		{"f.wav", 15},
		{"f.aac", 15},
		{"f.unknown", 1}, // 默认 PDF
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			got := detectMediaType(tt.file)
			if got != tt.want {
				t.Errorf("detectMediaType(%q) = %d，期望 %d", tt.file, got, tt.want)
			}
		})
	}
}

func TestUrlEncode(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"has space", "has%20space"},
		{"a&b=c", "a%26b%3Dc"},
		{"100%", "100%25"},
		{"normal_chars_123", "normal_chars_123"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := urlEncode(tt.input)
			if got != tt.want {
				t.Errorf("urlEncode(%q) = %q，期望 %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildCOSAuth(t *testing.T) {
	// 使用固定参数验证签名格式
	auth := buildCOSAuth(
		"AKIDtest",
		"secretKey123",
		"PUT",
		"/test-key",
		map[string]string{
			"host":           "bucket.cos.region.myqcloud.com",
			"content-length": "100",
		},
		1700000000,
		1700003600,
	)
	// 验证签名包含必要字段
	checks := []string{
		"q-sign-algorithm=sha1",
		"q-ak=AKIDtest",
		"q-sign-time=",
		"q-key-time=",
		"q-header-list=",
		"q-url-param-list=",
		"q-signature=",
	}
	for _, c := range checks {
		if !strings.Contains(auth, c) {
			t.Errorf("签名缺少字段: %s", c)
		}
	}
}

func TestGetKBList(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"code": 0,
			"msg": "ok",
			"data": {
				"addable_knowledge_base_list": [
					{"id": "kb1", "name": "开发"},
					{"id": "kb2", "name": "收藏"}
				]
			}
		}`))
	})

	list, err := GetKBList(client)
	if err != nil {
		t.Fatalf("GetKBList 失败: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("期望 2 个知识库，得到 %d", len(list))
	}
	if list[0].ID != "kb1" || list[0].Name != "开发" {
		t.Errorf("第一个知识库信息不正确: %+v", list[0])
	}
	if list[1].ID != "kb2" || list[1].Name != "收藏" {
		t.Errorf("第二个知识库信息不正确: %+v", list[1])
	}
}

func TestGetKBInfo(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		ids := body["ids"].([]any)
		if len(ids) != 1 || ids[0] != "kb1" {
			t.Errorf("ids = %v", ids)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"msg":"ok","data":{"infos":{"kb1":{"id":"kb1","name":"开发"}}}}`))
	})

	if err := GetKBInfo(client, "kb1"); err != nil {
		t.Fatalf("GetKBInfo 失败: %v", err)
	}
}

func TestBrowseKB(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["knowledge_base_id"] != "kb1" {
			t.Errorf("knowledge_base_id = %v", body["knowledge_base_id"])
		}
		// 不传 folder_id
		if _, ok := body["folder_id"]; ok {
			t.Error("空 folder_id 时不应发送该字段")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"msg":"ok","data":{"knowledge_list":[]}}`))
	})

	if err := BrowseKB(client, "kb1", ""); err != nil {
		t.Fatalf("BrowseKB 失败: %v", err)
	}
}

func TestBrowseKBWithFolder(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["folder_id"] != "folder1" {
			t.Errorf("folder_id = %v，期望 folder1", body["folder_id"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"msg":"ok","data":{"knowledge_list":[]}}`))
	})

	if err := BrowseKB(client, "kb1", "folder1"); err != nil {
		t.Fatalf("BrowseKB 失败: %v", err)
	}
}

func TestSearchKB(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["query"] != "golang" {
			t.Errorf("query = %v", body["query"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"msg":"ok","data":{"info_list":[]}}`))
	})

	if err := SearchKB(client, "kb1", "golang"); err != nil {
		t.Fatalf("SearchKB 失败: %v", err)
	}
}

func TestAddURL(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		urls := body["urls"].([]any)
		if len(urls) != 1 || urls[0] != "https://example.com" {
			t.Error("url 参数不正确")
		}
		if _, ok := body["folder_id"]; ok {
			t.Error("空 folder_id 时不应发送该字段")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"msg":"ok","data":{"results":{"https://example.com":{"url":"https://example.com","ret_code":0}}}}`))
	})

	if err := AddURL(client, "kb1", "", "https://example.com"); err != nil {
		t.Fatalf("AddURL 失败: %v", err)
	}
}

func TestAddURLWithFolder(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["folder_id"] != "folder1" {
			t.Errorf("folder_id = %v", body["folder_id"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"msg":"ok","data":{"results":{}}}`))
	})

	if err := AddURL(client, "kb1", "folder1", "https://example.com"); err != nil {
		t.Fatalf("AddURL 失败: %v", err)
	}
}

func TestGetMediaInfo(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"msg":"ok","data":{"media_type":1}}`))
	})

	if err := GetMediaInfo(client, "media123"); err != nil {
		t.Fatalf("GetMediaInfo 失败: %v", err)
	}
}

func TestUploadFileApiError(t *testing.T) {
	// 测试 create_media 阶段 API 错误
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":110001,"msg":"参数非法","data":{}}`))
	})

	err := UploadFile(client, "kb1", "/nonexistent/file.pdf", "")
	if err == nil {
		t.Fatal("不存在的文件应返回错误")
	}
}
