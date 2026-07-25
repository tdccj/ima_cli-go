// Package notes 封装 IMA 笔记相关的 API 调用。
//
// 支持的笔记操作：
//   - 笔记本列表（ListNotebooks）
//   - 笔记列表（ListNotes，可按笔记本筛选）
//   - 笔记搜索（SearchNote）
//   - 获取笔记内容（GetNoteContent）
//   - 创建笔记（CreateNote）
//   - 追加内容（AppendNote）
package notes

import (
	"encoding/json"
	"fmt"

	"ima_cli-go/internal/ima"
)

// ListNotebooks 列出当前用户的所有笔记本（note folder）。
func ListNotebooks(cli *ima.Client) error {
	resp, err := cli.Post("openapi/note/v1/list_notebook", ima.RequestBody{
		"cursor": "0",
		"limit":  20,
	})
	if err != nil {
		return err
	}

	folders, _ := json.MarshalIndent(resp.Data["note_folder_infos"], "", "  ")
	fmt.Println(string(folders))
	return nil
}

// ListNotes 列出笔记列表。如果指定了 folderID，只列出该笔记本下的笔记。
func ListNotes(cli *ima.Client, folderID string) error {
	body := ima.RequestBody{
		"cursor": "",
		"limit":  20,
	}
	if folderID != "" {
		body["folder_id"] = folderID
	}

	resp, err := cli.Post("openapi/note/v1/list_note", body)
	if err != nil {
		return err
	}

	notes, _ := json.MarshalIndent(resp.Data["note_book_list"], "", "  ")
	fmt.Println(string(notes))
	return nil
}

// SearchNote 按关键词搜索笔记内容。
func SearchNote(cli *ima.Client, keyword string) error {
	resp, err := cli.Post("openapi/note/v1/search_note", ima.RequestBody{
		"search_type": 1,
		"sort_type":   0,
		"query_info": map[string]string{
			"content": keyword,
		},
		"start": 0,
		"end":   20,
	})
	if err != nil {
		return err
	}

	results, _ := json.MarshalIndent(resp.Data["search_note_infos"], "", "  ")
	fmt.Println(string(results))
	return nil
}

// GetNoteContent 获取指定笔记的纯文本内容。
func GetNoteContent(cli *ima.Client, noteID string) error {
	resp, err := cli.Post("openapi/note/v1/get_doc_content", ima.RequestBody{
		"note_id":               noteID,
		"target_content_format": 0, // 0 = 纯文本
	})
	if err != nil {
		return err
	}

	fmt.Println(ima.GetString(resp.Data, "content"))
	return nil
}

// CreateNote 创建一篇新的 Markdown 笔记。
//   - title：笔记标题
//   - content：笔记正文（Markdown 格式，不含标题）
func CreateNote(cli *ima.Client, title, content string) error {
	resp, err := cli.Post("openapi/note/v1/import_doc", ima.RequestBody{
		"content_format": 1, // 1 = Markdown
		"content":        fmt.Sprintf("# %s\n\n%s", title, content),
	})
	if err != nil {
		return err
	}

	noteID := ima.GetString(resp.Data, "note_id")
	fmt.Printf("笔记创建成功，ID: %s\n", noteID)
	return nil
}

// AppendNote 在指定笔记末尾追加 Markdown 内容。
func AppendNote(cli *ima.Client, noteID, content string) error {
	resp, err := cli.Post("openapi/note/v1/append_doc", ima.RequestBody{
		"note_id":        noteID,
		"content_format": 1, // 1 = Markdown
		"content":        content,
	})
	if err != nil {
		return err
	}

	fmt.Printf("追加成功，笔记 ID: %s\n", ima.GetString(resp.Data, "note_id"))
	return nil
}
