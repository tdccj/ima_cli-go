package alias

import (
	"os"
	"path/filepath"
	"testing"
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
	kbID := "l2CsYhCgf5ECYEcHoeZW8wtgbvYNbjR_yLhvnKqULxs="

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

	// 验证非别名原样返回
	got, err = Resolve("some-raw-id")
	if err != nil {
		t.Fatalf("Resolve 失败: %v", err)
	}
	if got != "some-raw-id" {
		t.Errorf("Resolve 应原样返回，得到 %q", got)
	}
}

func TestAddDuplicate(t *testing.T) {
	dir, cleanup := testStore(t)
	defer cleanup()
	_ = dir

	if err := Add("dev", "id1"); err != nil {
		t.Fatalf("首次 Add 失败: %v", err)
	}
	if err := Add("dev", "id2"); err == nil {
		t.Fatal("重复添加应返回错误，但未返回")
	}
}

func TestRemove(t *testing.T) {
	dir, cleanup := testStore(t)
	defer cleanup()
	_ = dir

	if err := Add("dev", "some-id"); err != nil {
		t.Fatalf("Add 失败: %v", err)
	}
	if err := Remove("dev"); err != nil {
		t.Fatalf("Remove 失败: %v", err)
	}
	// 删除后应不再能解析
	got, _ := Resolve("dev")
	if got != "dev" {
		t.Errorf("删除后 Resolve 应原样返回，得到 %q", got)
	}
	// 删除不存在的别名应报错
	if err := Remove("nonexistent"); err == nil {
		t.Fatal("删除不存在的别名应返回错误")
	}
}

func TestAddValidation(t *testing.T) {
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
			err := Add(tt.name, "some-id")
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

	if err := Add("dev", "id1"); err != nil {
		t.Fatalf("Add 失败: %v", err)
	}
	if err := Add("prod", "id2"); err != nil {
		t.Fatalf("Add 失败: %v", err)
	}
	s, err := List()
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if len(s) != 2 {
		t.Errorf("List 应返回 2 个别名，得到 %d", len(s))
	}
	if s["dev"] != "id1" {
		t.Errorf("dev 别名值应为 id1，得到 %q", s["dev"])
	}
	if s["prod"] != "id2" {
		t.Errorf("prod 别名值应为 id2，得到 %q", s["prod"])
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
	if err := Add("dev", "persist-id"); err != nil {
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
	if s["dev"] != "persist-id" {
		t.Errorf("持久化的别名为 %q，期望 persist-id", s["dev"])
	}
}
