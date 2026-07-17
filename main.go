// ima-cli 是一个通过命令行管理 IMA 笔记和知识库的工具。
//
// 命令结构：
//   - 知识库命令直接挂在顶层：ima list, ima info, ima browse, ima search, ima upload, ima url, ima get-media
//   - 笔记命令通过 notes 子命令：ima notes list-notebooks, ima notes list, ima notes search, etc.
//   - 别名管理通过 alias 子命令：ima alias add, ima alias list, ima alias remove
//
// 所有 API 调用通过内部的 ima.Client 完成，认证凭证通过环境变量或配置文件加载。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/user/ima-cli/internal/alias"
	"github.com/user/ima-cli/internal/config"
	"github.com/user/ima-cli/internal/ima"
	"github.com/user/ima-cli/internal/kb"
	"github.com/user/ima-cli/internal/notes"
)

// Version 在编译时通过 ldflags 注入，如：
//
//	go build -ldflags "-X main.Version=1.0.0" -o ima .
var Version = "1.0.1"

// usage 打印 CLI 帮助信息。
func usage() {
		fmt.Fprintf(os.Stderr, `IMA CLI %s — 通过命令行管理 IMA 笔记和知识库

用法:
  ima <command> [选项]

知识库命令:
  list                         列出可添加的知识库
  info <kb-id>                 获取知识库信息
  browse <kb-id> [--folder-id]     浏览知识库内容
  search <kb-id> <query>       搜索知识库
  upload <kb-id> <file-path> [--folder-id]   上传文件到知识库
  url <kb-id> <url> [--folder-id]            添加网页到知识库
  get-media <media-id>         获取媒体信息

别名管理:
  alias add <name> <kb-id>    添加知识库别名
  alias list                  列出所有别名
  alias remove <name>         删除别名

笔记命令:
  ima notes list-notebooks              列出笔记本
  ima notes list [--folder-id]     列出笔记（可按笔记本筛选）
  ima notes search <keyword>            搜索笔记
  ima notes get <note-id>               获取笔记内容
  ima notes create <title> <content>    创建笔记
  ima notes append <note-id> <content>  追加内容到笔记

其他:
  version 或 -version          显示版本信息

环境变量:
  IMA_OPENAPI_CLIENTID  或 IMA_CLIENT_ID     Client ID
  IMA_OPENAPI_APIKEY    或 IMA_API_KEY       API Key

配置目录: ~/.config/ima/client_id 和 ~/.config/ima/api_key
`, Version)
}

// printErr 输出错误信息并退出程序。
func printErr(msg string) {
	fmt.Fprintln(os.Stderr, "错误:", msg)
	os.Exit(1)
}

func main() {
	versionFlag := flag.Bool("version", false, "显示版本信息")
	flag.Usage = usage
	flag.Parse()

	if *versionFlag {
		fmt.Printf("ima-cli version %s\n", Version)
		return
	}

	args := flag.Args()
	if len(args) < 1 {
		usage()
		os.Exit(1)
	}

	// 加载认证凭证
	creds, err := config.Load()
	if err != nil {
		printErr(err.Error())
	}

	// 创建 API 客户端
	cli := ima.NewClient(creds)

	// 路由顶层命令
	switch args[0] {
	case "help":
		usage()
	case "version":
		fmt.Printf("ima-cli version %s\n", Version)
	case "notes":
		runNotes(cli, args[1:])
	case "alias":
		runAlias(args[1:])
	case "list":
		runList(cli)
	case "info":
		if len(args) < 2 {
			printErr("用法: ima info <kb-id>")
		}
		id, _ := alias.Resolve(args[1])
		if err := kb.GetKBInfo(cli, id); err != nil {
			printErr(err.Error())
		}
	case "browse":
		if len(args) < 2 {
			printErr("用法: ima browse <kb-id> [--folder-id <id>]")
		}
		id, _ := alias.Resolve(args[1])
		fs := flag.NewFlagSet("browse", flag.ExitOnError)
		folderID := fs.String("folder-id", "", "文件夹 ID")
		fs.Parse(args[2:])
		if err := kb.BrowseKB(cli, id, *folderID); err != nil {
			printErr(err.Error())
		}
	case "search":
		if len(args) < 3 {
			printErr("用法: ima search <kb-id> <query>")
		}
		id, _ := alias.Resolve(args[1])
		if err := kb.SearchKB(cli, id, strings.Join(args[2:], " ")); err != nil {
			printErr(err.Error())
		}
	case "upload":
		if len(args) < 3 {
			printErr("用法: ima upload <kb-id> <file-path> [--folder-id <id>]")
		}
		id, _ := alias.Resolve(args[1])
		fs := flag.NewFlagSet("upload", flag.ExitOnError)
		folderID := fs.String("folder-id", "", "文件夹 ID")
		fs.Parse(args[3:])
		if err := kb.UploadFile(cli, id, args[2], *folderID); err != nil {
			printErr(err.Error())
		}
	case "url":
		if len(args) < 3 {
			printErr("用法: ima url <kb-id> <url> [--folder-id <id>]")
		}
		id, _ := alias.Resolve(args[1])
		fs := flag.NewFlagSet("url", flag.ExitOnError)
		folderID := fs.String("folder-id", "", "文件夹 ID")
		fs.Parse(args[3:])
		if err := kb.AddURL(cli, id, *folderID, args[2]); err != nil {
			printErr(err.Error())
		}
	case "get-media":
		if len(args) < 2 {
			printErr("用法: ima get-media <media-id>")
		}
		if err := kb.GetMediaInfo(cli, args[1]); err != nil {
			printErr(err.Error())
		}
	default:
		printErr(fmt.Sprintf("未知命令: %s", args[0]))
	}
}

// runAlias 处理别名管理命令。
func runAlias(args []string) {
	if len(args) < 1 {
		printErr("用法: ima alias <add|list|remove> [...]")
	}

	switch args[0] {
	case "add":
		if len(args) < 3 {
			printErr("用法: ima alias add <name> <kb-id>")
		}
		if err := alias.Add(args[1], args[2]); err != nil {
			printErr(err.Error())
		}
		fmt.Printf("别名 '%s' 添加成功\n", args[1])
	case "list":
		s, err := alias.List()
		if err != nil {
			printErr(err.Error())
		}
		if len(s) == 0 {
			fmt.Println("暂无别名")
			return
		}
		for name, id := range s {
			fmt.Printf("%-20s %s\n", name, id)
		}
	case "remove":
		if len(args) < 2 {
			printErr("用法: ima alias remove <name>")
		}
		if err := alias.Remove(args[1]); err != nil {
			printErr(err.Error())
		}
		fmt.Printf("别名 '%s' 已删除\n", args[1])
	default:
		printErr(fmt.Sprintf("未知子命令: %s", args[0]))
	}
}

// runList 列出知识库列表，并拼接别名信息后以 JSON 格式输出。
func runList(cli *ima.Client) {
	list, err := kb.GetKBList(cli)
	if err != nil {
		printErr(err.Error())
	}

	// 构建别名查谘表（知识库 ID → 别名）
	aliases, _ := alias.Load()
	aliasByName := make(map[string]string)
	for name, id := range aliases {
		aliasByName[id] = name
	}

	type itemWithAlias struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Alias string `json:"alias,omitempty"`
	}
	var out []itemWithAlias
	for _, item := range list {
		ia := itemWithAlias{ID: item.ID, Name: item.Name}
		if a, ok := aliasByName[item.ID]; ok {
			ia.Alias = a
		}
		out = append(out, ia)
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(b))
}

// runNotes 处理笔记管理命令。
func runNotes(cli *ima.Client, args []string) {
	if len(args) < 1 {
		printErr("用法: ima notes <subcommand>，可用: list-notebooks, list, search, get, create, append")
	}

	switch args[0] {
	case "list-notebooks":
		if err := notes.ListNotebooks(cli); err != nil {
			printErr(err.Error())
		}
	case "list":
		fs := flag.NewFlagSet("notes list", flag.ExitOnError)
		folderID := fs.String("folder-id", "", "笔记本 ID")
		fs.Parse(args[1:])
		if err := notes.ListNotes(cli, *folderID); err != nil {
			printErr(err.Error())
		}
	case "search":
		keyword := strings.Join(args[1:], " ")
		if keyword == "" {
			printErr("用法: ima notes search <keyword>")
		}
		if err := notes.SearchNote(cli, keyword); err != nil {
			printErr(err.Error())
		}
	case "get":
		if len(args) < 2 {
			printErr("用法: ima notes get <note-id>")
		}
		if err := notes.GetNoteContent(cli, args[1]); err != nil {
			printErr(err.Error())
		}
	case "create":
		if len(args) < 3 {
			printErr("用法: ima notes create <title> <content>")
		}
		if err := notes.CreateNote(cli, args[1], strings.Join(args[2:], " ")); err != nil {
			printErr(err.Error())
		}
	case "append":
		if len(args) < 3 {
			printErr("用法: ima notes append <note-id> <content>")
		}
		if err := notes.AppendNote(cli, args[1], strings.Join(args[2:], " ")); err != nil {
			printErr(err.Error())
		}
	default:
		printErr(fmt.Sprintf("未知子命令: %s", args[0]))
	}
}
