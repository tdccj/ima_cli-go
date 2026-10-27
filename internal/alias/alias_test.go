package alias

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testKBID  = "ima_kb_Abc12345"
	testKBID2 = "ima_kb_Xyz98765"
)

// testStore 创建一个临时的别名存储目录，返回目录路径和清理函数。
func testStore(t *testing.T) (string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "ima-alias-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	// 备份并覆盖 home 目录
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	// 覆盖 path() 的行为我们需要 mock home 目录
	// 但 path() 用的是 os.UserHomeDir()，它会读取 HOME 环境变量
	return dir, func() {
		os.RemoveAll(dir)
		os.Setenv("HOME", origHome)
	}
}

func TestAddAndResolve(t *testing.T) {
	// 用临时目录避免污染实际配置文件
	dir, cleanup := testStore(t)
	defer cleanup()

	_ = dir
	kbID := testKBID

	// 添加别名
	err := Add("dev", kbID)
	if err != nil {
		t.Fatalf("Add 失败: %v", err)
	}

	// 验证解析
	got, err := Resolve("dev")
	if err != nil {
		t.Fatalf("Resolve 失败: %v", err)
	}
	if got != kbID {
		t.Errorf("Resolve 返回 %q，期望 %q", got, kbID)
	}

	// 验证非别名的合法 ID 原样返回
	got, err = Resolve(testKBID2)
	if err != nil {
		t.Fatalf("Resolve 失败: %v", err)
	}
	if got != testKBID2 {
		t.Errorf("Resolve 应原样返回，得到 %q", got)
	}
}

func TestAddDuplicate(t *testing.T) {
	dir, cleanup := testStore(t)
	defer cleanup()
	_ = dir

	if err := Add("dev", testKBID); err != nil {
		t.Fatalf("首次 Add 失败: %v", err)
	}
	if err := Add("dev", testKBID2); err == nil {
		t.Fatal("重复添加应返回错误，但未返回")
	}
}

func TestUpdate(t *testing.T) {
	_, cleanup := testStore(t)
	defer cleanup()

	if err := Add("dev", testKBID); err != nil {
		t.Fatalf("Add 失败: %v", err)
	}
	if err := Update("dev", testKBID2); err != nil {
		t.Fatalf("Update 失败: %v", err)
	}

	got, err := Resolve("dev")
	if err != nil {
		t.Fatalf("Resolve 失败: %v", err)
	}
	if got != testKBID2 {
		t.Errorf("更新后的知识库 ID 为 %q，期望 %q", got, testKBID2)
	}
}

func TestUpdateNonexistent(t *testing.T) {
	_, cleanup := testStore(t)
	defer cleanup()

	if err := Update("nonexistent", testKBID); err == nil {
		t.Fatal("更新不存在的别名应返回错误")
	}
}

func TestUpdateRejectsInvalidKnowledgeBaseID(t *testing.T) {
	_, cleanup := testStore(t)
	defer cleanup()

	if err := Add("dev", testKBID); err != nil {
		t.Fatalf("Add 失败: %v", err)
	}
	if err := Update("dev", "legacy-id"); err == nil {
		t.Fatal("更新为旧格式知识库 ID 应返回错误")
	}

	got, err := Resolve("dev")
	if err != nil {
		t.Fatalf("Resolve 失败: %v", err)
	}
	if got != testKBID {
		t.Errorf("更新失败后知识库 ID 不应改变，得到 %q", got)
	}
}

func TestRemove(t *testing.T) {
	dir, cleanup := testStore(t)
	defer cleanup()
	_ = dir

	if err := Add("dev", testKBID); err != nil {
		t.Fatalf("Add 失败: %v", err)
	}
	if err := Remove("dev"); err != nil {
		t.Fatalf("Remove 失败: %v", err)
	}
	// 删除后别名映射应不存在
	s, err := Load()
	if err != nil {
		t.Fatalf("Load 失败: %v", err)
	}
	if _, ok := s["dev"]; ok {
		t.Error("删除后别名映射仍然存在")
	}
	// 删除不存在的别名应报错
	if err := Remove("nonexistent"); err == nil {
		t.Fatal("删除不存在的别名应返回错误")
	}
}

func TestAddValidation(t *testing.T) {
	_, cleanup := testStore(t)
	defer cleanup()

	tests := []struct {
		name string
		want bool // true=应成功, false=应失败
	}{
		{"abc", true},
		{"a_b_c", true},
		{"hello123", true},
		{"test-1", false},      // 含连字符
		{"abc.def", false},     // 含点号
		{"a b", false},         // 含空格
		{"abcdefghij", true},   // 恰好 10 字符
		{"abcdefghijk", false}, // 超过 10 字符
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Add(tt.name, testKBID)
			if tt.want && err != nil {
				t.Errorf("Add(%q) 应成功，但得到: %v", tt.name, err)
			}
			if !tt.want && err == nil {
				t.Errorf("Add(%q) 应失败，但成功了", tt.name)
			}
		})
	}
}

func TestList(t *testing.T) {
	dir, cleanup := testStore(t)
	defer cleanup()
	_ = dir

	if err := Add("dev", testKBID); err != nil {
		t.Fatalf("Add 失败: %v", err)
	}
	if err := Add("prod", testKBID2); err != nil {
		t.Fatalf("Add 失败: %v", err)
	}
	s, err := List()
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if len(s) != 2 {
		t.Errorf("List 应返回 2 个别名，得到 %d", len(s))
	}
	if s["dev"] != testKBID {
		t.Errorf("dev 别名值应为 %s，得到 %q", testKBID, s["dev"])
	}
	if s["prod"] != testKBID2 {
		t.Errorf("prod 别名值应为 %s，得到 %q", testKBID2, s["prod"])
	}
}

func TestLoadFromScratch(t *testing.T) {
	// 不创建任何文件，直接 Load 应返回空 Store
	dir, cleanup := testStore(t)
	defer cleanup()
	_ = dir

	s, err := Load()
	if err != nil {
		t.Fatalf("Load 从空白状态失败: %v", err)
	}
	if len(s) != 0 {
		t.Errorf("空白状态应返回空 Store，得到 %d 项", len(s))
	}
}

func TestAliasPersistence(t *testing.T) {
	// 验证别名写入后能正确持久化
	dir, cleanup := testStore(t)
	defer cleanup()

	// 添加别名
	if err := Add("dev", testKBID); err != nil {
		t.Fatalf("Add 失败: %v", err)
	}

	// 验证文件存在
	cfgPath := filepath.Join(dir, ".config/ima", fileName)
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Fatal("别名文件未创建")
	}

	// 重新 Load 验证持久化
	s, err := Load()
	if err != nil {
		t.Fatalf("重新 Load 失败: %v", err)
	}
	if s["dev"] != testKBID {
		t.Errorf("持久化的别名为 %q，期望 %s", s["dev"], testKBID)
	}
}

func TestKnowledgeBaseIDFormat(t *testing.T) {
	_, cleanup := testStore(t)
	defer cleanup()

	tests := []struct {
		name  string
		id    string
		valid bool
	}{
		{"合法 ID", "ima_kb_Abc12345", true},
		{"旧格式 ID", "l2CsYhCgf5ECYEcHoeZW8wtgbvYNbjR_yLhvnKqULxs=", false},
		{"缺少前缀", "Abc12345", false},
		{"后缀过短", "ima_kb_Abc1234", false},
		{"后缀过长", "ima_kb_Abc123456", false},
		{"后缀含非法字符", "ima_kb_Abc1234_", false},
		{"空前缀后缀", "ima_kb_", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(tt.id)
			if tt.valid && (err != nil || got != tt.id) {
				t.Errorf("Resolve(%q) = %q, %v，应原样返回", tt.id, got, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("Resolve(%q) 应拒绝无效 ID", tt.id)
			}
		})
	}
}

func TestResolveRejectsStaleAlias(t *testing.T) {
	_, cleanup := testStore(t)
	defer cleanup()

	if err := save(Store{"dev": "legacy-id"}); err != nil {
		t.Fatalf("保存测试别名失败: %v", err)
	}
	_, err := Resolve("dev")
	if err == nil {
		t.Fatal("旧格式别名应返回错误")
	}
	if !strings.Contains(err.Error(), "ima list") || !strings.Contains(err.Error(), "ima alias") {
		t.Errorf("错误应说明刷新和重新绑定方法，得到: %v", err)
	}
}

func TestAddRejectsInvalidKnowledgeBaseID(t *testing.T) {
	_, cleanup := testStore(t)
	defer cleanup()

	if err := Add("dev", "legacy-id"); err == nil {
		t.Fatal("添加旧格式知识库 ID 应返回错误")
	}
}
