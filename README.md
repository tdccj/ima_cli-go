# ima-cli

通过命令行管理 IMA 笔记和知识库。Go 实现，零外部依赖。

## 安装

```bash
git clone <repo-url>
cd ima-cli
go build -o ima .
# 推荐将 ima 加入 PATH
# cp ima /usr/local/bin/
```

## 配置

### 方式一：环境变量

```bash
export IMA_OPENAPI_CLIENTID="your_client_id"
export IMA_OPENAPI_APIKEY="your_api_key"
```

### 方式二：配置文件

```bash
mkdir -p ~/.config/ima
echo "your_client_id" > ~/.config/ima/client_id
echo "your_api_key" > ~/.config/ima/api_key
```

> 凭证获取：打开 https://ima.qq.com/agent-interface

## 知识库命令

知识库命令直接挂在 `ima` 顶层。

### 列出知识库

```bash
ima list
```

输出包含别名信息（如有）。

### 获取知识库信息

```bash
ima info <kb-id>
```

### 浏览知识库内容

```bash
# 浏览根目录
ima browse <kb-id>

# 浏览子文件夹
ima browse <kb-id> --folder-id <folder-id>
```

### 搜索知识库

```bash
ima search <kb-id> <关键词>
```

### 上传文件到知识库

```bash
# 上传到根目录
ima upload <kb-id> <文件路径>

# 上传到指定文件夹
ima upload <kb-id> <文件路径> --folder-id <folder-id>
```

支持的文件类型：PDF、Word、PPT、Excel、Markdown、图片、TXT、Xmind、音频等。

### 添加网页到知识库

```bash
# 添加到根目录
ima url <kb-id> <网页URL>

# 添加到指定文件夹
ima url <kb-id> <网页URL> --folder-id <folder-id>
```

### 获取媒体信息

```bash
ima get-media <media-id>
```

## 别名管理

为知识库 ID 设置别名，避免记忆长串 ID。

```bash
# 添加别名
ima alias add <名称> <kb-id>

# 列出所有别名
ima alias list

# 删除别名
ima alias remove <名称>
```

别名限制：最长 10 个字符，仅限字母、数字、下划线。

设置后所有知识库命令（info/browse/search/upload/url）的 `kb-id` 参数均可使用别名代替。

## 笔记命令

笔记命令以 `notes` 作为子命令前缀。

### 列出笔记本

```bash
ima notes list-notebooks
```

### 列出笔记

```bash
# 列出所有笔记
ima notes list

# 按笔记本筛选
ima notes list --folder-id <folder-id>
```

### 搜索笔记

```bash
ima notes search <关键词>
```

### 获取笔记内容

```bash
ima notes get <note-id>
```

### 创建笔记

```bash
ima notes create <标题> <内容>
```

### 追加内容到笔记

```bash
ima notes append <note-id> <追加内容>
```

## 示例

```bash
# 列出知识库（带别名）
$ ima list
[
  { "id": "xxx", "name": "开发", "alias": "dev" },
  { "id": "yyy", "name": "收藏" }
]

# 使用别名
$ ima info dev

# 上传文件到知识库
$ ima upload xxx report.pdf

# 搜索笔记
$ ima notes search Go语言

# 创建笔记
$ ima notes create 今日计划 完成ima-cli开发
```
