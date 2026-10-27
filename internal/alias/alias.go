// Package alias 提供知识库 ID 的别名管理功能。
//
// 用户可以为长知识库 ID 设置简短别名，所有知识库命令（info/browse/search/upload/url）
// 的 kb-id 参数均可使用别名代替。别名存储在 ~/.config/ima/aliases.json 中。
package alias

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"unicode"
)

// Store 存储别名映射，key 为别名名称，value 为知识库 ID。
type Store map[string]string

const fileName = "aliases.json"

var knowledgeBaseIDPattern = regexp.MustCompile(`^ima_kb_[A-Za-z0-9]{8}$`)

func validateKnowledgeBaseID(kbID string) error {
	if !knowledgeBaseIDPattern.MatchString(kbID) {
		return fmt.Errorf("知识库 ID 无效或已过期，有效格式为 ima_kb_XXXXXXXX（XXXXXXXX 为 8 位字母或数字）；请运行 ima list 从最新响应获取 ID")
	}
	return nil
}

// path 返回别名配置文件的完整路径。
func path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("无法获取 home 目录: %w", err)
	}
	return filepath.Join(home, ".config/ima", fileName), nil
}

// Load 从配置文件中读取所有别名。
// 如果配置文件不存在，返回空的 Store（而非错误）。
func Load() (Store, error) {
	p, err := path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return Store{}, nil
		}
		return nil, fmt.Errorf("读取别名文件失败: %w", err)
	}
	var s Store
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("解析别名文件失败: %w", err)
	}
	return s, nil
}

// save 将别名数据写入配置文件。
func save(s Store) error {
	p, err := path()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化别名失败: %w", err)
	}
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}
	if err := os.WriteFile(p, data, 0644); err != nil {
		return fmt.Errorf("写入别名文件失败: %w", err)
	}
	return nil
}

// Add 添加一个别名映射。限制：
//   - 名称不超过 10 个字符
//   - 只能包含字母、数字、下划线
//   - 不能与已有别名重名
func Add(name, kbID string) error {
	if err := validateKnowledgeBaseID(kbID); err != nil {
		return err
	}
	if len(name) > 10 {
		return fmt.Errorf("别名长度不能超过 10 个字符")
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return fmt.Errorf("别名只能包含字母、数字和下划线")
		}
	}

	s, err := Load()
	if err != nil {
		return err
	}
	if _, exists := s[name]; exists {
		return fmt.Errorf("别名 '%s' 已存在", name)
	}
	s[name] = kbID
	return save(s)
}

// Update 更新指定别名对应的知识库 ID。如果别名不存在则返回错误。
func Update(name, kbID string) error {
	if err := validateKnowledgeBaseID(kbID); err != nil {
		return err
	}

	s, err := Load()
	if err != nil {
		return err
	}
	if _, exists := s[name]; !exists {
		return fmt.Errorf("别名 '%s' 不存在", name)
	}
	s[name] = kbID
	return save(s)
}

// Remove 删除指定名称的别名。如果别名不存在则返回错误。
func Remove(name string) error {
	s, err := Load()
	if err != nil {
		return err
	}
	if _, exists := s[name]; !exists {
		return fmt.Errorf("别名 '%s' 不存在", name)
	}
	delete(s, name)
	return save(s)
}

// List 返回所有别名的映射。
func List() (Store, error) {
	return Load()
}

// Resolve 将输入的名称解析为知识库 ID，并拒绝旧格式或无效 ID。
// 过期 ID 不会被自动重试；调用方必须从最新知识库列表获取新 ID。
func Resolve(input string) (string, error) {
	s, err := Load()
	if err != nil {
		return "", err
	}
	if id, ok := s[input]; ok {
		if err := validateKnowledgeBaseID(id); err != nil {
			return "", fmt.Errorf("别名 %q 保存的知识库 ID 无效或已过期；请运行 ima list 获取最新 ID，再执行 ima alias update %s <kb-id> 重新绑定", input, input)
		}
		return id, nil
	}
	if err := validateKnowledgeBaseID(input); err != nil {
		return "", err
	}
	return input, nil
}
