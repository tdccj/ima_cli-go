// Package ima 提供 IMA OpenAPI 的统一 HTTP 客户端和数据结构。
//
// 核心组件：
//   - Client：封装 HTTP POST 请求、认证头、响应解析
//   - APIResponse：统一的 API 响应包装（code/msg/data）
//   - 各种业务数据类型（NoteBookInfo, KnowledgeBaseInfo 等）
//
// 所有 IMA API 均通过 Client.Post 发送 POST 请求，请求体为 JSON，
// 响应体均为 {"code":0, "msg":"成功", "data":{...}} 的格式。
package ima

// RequestBody 是 API 请求体的泛型类型，用 map 避免为每个 API 定义结构体。
type RequestBody map[string]any

// APIResponse 是所有 IMA API 的统一响应结构。
// 业务数据位于 Data 字段中，需按接口文档提取对应字段。
type APIResponse struct {
	Code int            `json:"code"`
	Msg  string         `json:"msg"`
	Data map[string]any `json:"data"`
}

// NoteBookInfo 笔记基本信息。
type NoteBookInfo struct {
	NoteID     string       `json:"note_id"`
	Title      string       `json:"title"`
	Summary    string       `json:"summary"`
	CreateTime int64        `json:"create_time"`
	ModifyTime int64        `json:"modify_time"`
	CoverImage string       `json:"cover_image"`
	ExtInfo    *NoteExtInfo `json:"note_ext_info,omitempty"`
}

// NoteExtInfo 笔记扩展字段，包含所属笔记本信息。
type NoteExtInfo struct {
	FolderID   string `json:"folder_id"`
	FolderName string `json:"folder_name"`
}

// NoteFolderInfo 笔记本信息。
type NoteFolderInfo struct {
	FolderID       string `json:"folder_id"`
	Name           string `json:"name"`
	CreateTime     int64  `json:"create_time"`
	ModifyTime     int64  `json:"modify_time"`
	NoteNumber     int64  `json:"note_number"`
	ParentFolderID string `json:"parent_folder_id"`
	FolderType     int    `json:"folder_type"`
}

// SearchNoteInfo 笔记搜索结果条目，包含高亮信息。
type SearchNoteInfo struct {
	NoteBookInfo  NoteBookInfo       `json:"note_book_info"`
	HighlightInfo map[string]string  `json:"highlightInfo"`
}

// KnowledgeBaseInfo 知识库信息。
type KnowledgeBaseInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	CoverURL    string   `json:"cover_url"`
	Description string   `json:"description"`
	RecommendedQuestions []string `json:"recommended_questions"`
}

// KnowledgeInfo 知识条目信息。
type KnowledgeInfo struct {
	MediaID        string `json:"media_id"`
	Title          string `json:"title"`
	ParentFolderID string `json:"parent_folder_id"`
}

// FolderInfo 知识库文件夹条目。
type FolderInfo struct {
	FolderID       string `json:"folder_id"`
	Name           string `json:"name"`
	FileNumber     int64  `json:"file_number"`
	FolderNumber   int64  `json:"folder_number"`
	ParentFolderID string `json:"parent_folder_id"`
	IsTop          bool   `json:"is_top"`
}

// AddableKnowledgeBaseInfo 用户可添加内容的知识库简略信息。
type AddableKnowledgeBaseInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SearchedKnowledgeBaseInfo 知识库搜索结果。
type SearchedKnowledgeBaseInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	CoverURL string `json:"cover_url"`
}

// SearchedKnowledgeInfo 知识库内容搜索结果。
type SearchedKnowledgeInfo struct {
	MediaID          string `json:"media_id"`
	Title            string `json:"title"`
	ParentFolderID   string `json:"parent_folder_id"`
	HighlightContent string `json:"highlight_content"`
}

// Credential 腾讯云 COS 临时上传凭证，由 create_media 接口返回。
type Credential struct {
	Token        string `json:"token"`
	SecretID     string `json:"secret_id"`
	SecretKey    string `json:"secret_key"`
	StartTime    int64  `json:"start_time"`
	ExpiredTime  int64  `json:"expired_time"`
	AppID        string `json:"appid"`
	BucketName   string `json:"bucket_name"`
	Region       string `json:"region"`
	CustomDomain string `json:"custom_domain"`
	CosKey       string `json:"cos_key"`
}

// FileInfo add_knowledge 接口中文件上传时的文件信息。
type FileInfo struct {
	CosKey         string `json:"cos_key"`
	FileSize       uint64 `json:"file_size"`
	LastModifyTime int64  `json:"last_modify_time"`
	FileName       string `json:"file_name"`
}

// ContentInfo 网页/笔记/会话等内容信息，用于 add_knowledge 接口。
type ContentInfo struct {
	ContentID string `json:"content_id"`
}

// URLInfo get_media_info 接口返回的访问链接信息。
type URLInfo struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

// NotebookExtInfo get_media_info 返回笔记类型媒体时的扩展信息。
type NotebookExtInfo struct {
	NotebookID string `json:"notebook_id"`
}

// MediaInfoData get_media_info 接口返回的媒体详情数据。
type MediaInfoData struct {
	MediaType       int32            `json:"media_type"`
	URLInfo         *URLInfo         `json:"url_info,omitempty"`
	NotebookExtInfo *NotebookExtInfo `json:"notebook_ext_info,omitempty"`
}

// ImportURLData import_urls 接口中单个 URL 的导入结果。
type ImportURLData struct {
	URL      string `json:"url"`
	RetCode  int32  `json:"ret_code"`
	MediaID  string `json:"media_id"`
}

// CheckRepeatedNamesParam 检查文件名重复时的请求参数。
type CheckRepeatedNamesParam struct {
	Name      string `json:"name"`
	MediaType int32  `json:"media_type"`
}

// CheckRepeatedNamesResult 文件名重复检查结果。
type CheckRepeatedNamesResult struct {
	Name       string `json:"name"`
	IsRepeated bool   `json:"is_repeated"`
}
