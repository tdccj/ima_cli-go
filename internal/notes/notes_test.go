package notes

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

func TestListNotebooks(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "list_notebook") {
			t.Errorf("路径应为 list_notebook，得到 %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"code": 0,
			"msg": "ok",
			"data": {
				"note_folder_infos": [
					{"folder_id": "f1", "name": "工作"},
					{"folder_id": "f2", "name": "学习"}
				]
			}
		}`))
	})

	// 该函数直接打印到 stdout，我们只验证不报错
	err := ListNotebooks(client)
	if err != nil {
		t.Fatalf("ListNotebooks 失败: %v", err)
	}
}

func TestListNotes(t *testing.T) {
	var capturedBody map[string]any
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"msg":"ok","data":{"note_book_list":[]}}`))
	})

	// 不传 folder_id
	if err := ListNotes(client, ""); err != nil {
		t.Fatalf("ListNotes 失败: %v", err)
	}
	// 应不包含 folder_id 字段
	if _, exists := capturedBody["folder_id"]; exists {
		t.Error("空 folder_id 时不应发送该字段")
	}

	// 传 folder_id
	if err := ListNotes(client, "f1"); err != nil {
		t.Fatalf("ListNotes 失败: %v", err)
	}
	if capturedBody["folder_id"] != "f1" {
		t.Errorf("folder_id = %v，期望 f1", capturedBody["folder_id"])
	}
}

func TestSearchNote(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["search_type"].(float64) != 1 {
			t.Error("search_type 应为 1")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"msg":"ok","data":{"search_note_infos":[]}}`))
	})

	if err := SearchNote(client, "golang"); err != nil {
		t.Fatalf("SearchNote 失败: %v", err)
	}
}

func TestGetNoteContent(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"msg":"ok","data":{"content":"笔记正文内容"}}`))
	})

	if err := GetNoteContent(client, "note123"); err != nil {
		t.Fatalf("GetNoteContent 失败: %v", err)
	}
}

func TestCreateNote(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		// 验证 content_format 固定为 1
		if body["content_format"].(float64) != 1 {
			t.Error("content_format 应为 1")
		}
		// 验证内容包含标题
		content := body["content"].(string)
		if !strings.HasPrefix(content, "# 测试标题") {
			t.Errorf("content 应以标题开头，得到 %q", content)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"msg":"ok","data":{"note_id":"new_note_123"}}`))
	})

	if err := CreateNote(client, "测试标题", "这是正文"); err != nil {
		t.Fatalf("CreateNote 失败: %v", err)
	}
}

func TestAppendNote(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["note_id"] != "note123" {
			t.Errorf("note_id = %v", body["note_id"])
		}
		if body["content"] != "追加内容" {
			t.Errorf("content = %v", body["content"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"msg":"ok","data":{"note_id":"note123"}}`))
	})

	if err := AppendNote(client, "note123", "追加内容"); err != nil {
		t.Fatalf("AppendNote 失败: %v", err)
	}
}

func TestApiErrorPropagation(t *testing.T) {
	// 验证 API 业务错误会传递到调用方
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":210001,"msg":"参数错误","data":{}}`))
	})

	err := ListNotebooks(client)
	if err == nil {
		t.Fatal("API 错误应返回错误")
	}
	if !strings.Contains(err.Error(), "210001") {
		t.Errorf("错误信息应包含错误码，得到: %v", err)
	}
}
