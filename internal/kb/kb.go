// Package kb 封装 IMA 知识库相关的 API 调用和 COS 文件上传逻辑。
//
// 支持的知識库操作：
//   - 列出可添加的知识库（GetKBList）
//   - 获取知识库详情（GetKBInfo）
//   - 浏览知识库内容（BrowseKB）
//   - 搜索知识库（SearchKB）
//   - 上传文件（UploadFile）：三步流程（create_media → COS PUT → add_knowledge）
//   - 添加网页 URL（AddURL）
//   - 获取媒体信息（GetMediaInfo）
//
// COS 上传基于腾讯云 COS 的 HMAC-SHA1 签名算法，
// 参考：https://cloud.tencent.com/document/product/436/7778
package kb

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/user/ima-cli/internal/ima"
)

// cosDefaultTimeout COS 上传超时时间（5 分钟），大文件可通过此值调整。
const cosDefaultTimeout = 5 * time.Minute

// hmacSHA1 计算 HMAC-SHA1 签名，返回十六进制字符串。
func hmacSHA1(key, data []byte) string {
	mac := hmac.New(sha1.New, key)
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

// sha1Hash 计算 SHA1 哈希，返回十六进制字符串。
func sha1Hash(data []byte) string {
	h := sha1.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// buildCOSAuth 构建腾讯云 COS 的 PUT Object 请求的 Authorization 头。
//
// 签名算法（腾讯云 COS V5 版本）：
//  1. SignKey = HMAC-SHA1(SecretKey, KeyTime)
//  2. HttpString = method\npathname\nparams\nheaders\n
//  3. StringToSign = sha1\nKeyTime\nSHA1(HttpString)\n
//  4. Signature = HMAC-SHA1(SignKey, StringToSign)
//
// 参考：https://cloud.tencent.com/document/product/436/7778
func buildCOSAuth(secretID, secretKey, method, pathname string, headers map[string]string, startTime, expiredTime int64) string {
	keyTime := fmt.Sprintf("%d;%d", startTime, expiredTime)
	signKey := hmacSHA1([]byte(secretKey), []byte(keyTime))

	// 对 header key 排序以保证签名一致性
	var headerKeys []string
	for k := range headers {
		headerKeys = append(headerKeys, strings.ToLower(k))
	}
	sort.Strings(headerKeys)

	var pHParts []string
	for _, k := range headerKeys {
		pHParts = append(pHParts, fmt.Sprintf("%s=%s", k, urlEncode(headers[k])))
	}
	httpHeaders := strings.Join(pHParts, "&")

	httpString := fmt.Sprintf("%s\n%s\n\n%s\n", strings.ToLower(method), pathname, httpHeaders)
	stringToSign := fmt.Sprintf("sha1\n%s\n%s\n", keyTime, sha1Hash([]byte(httpString)))
	signature := hmacSHA1([]byte(signKey), []byte(stringToSign))

	headerList := strings.Join(headerKeys, ";")
	return fmt.Sprintf("q-sign-algorithm=sha1&q-ak=%s&q-sign-time=%s&q-key-time=%s&q-header-list=%s&q-url-param-list=&q-signature=%s",
		secretID, keyTime, keyTime, headerList, signature)
}

// urlEncode 对 COS header 中的特殊字符进行百分号编码。
// 仅编码 %、&、= 和空格，其余字符保持原样。
func urlEncode(s string) string {
	var out strings.Builder
	for _, r := range s {
		if r == '%' || r == '&' || r == '=' || r == ' ' {
			out.WriteString(fmt.Sprintf("%%%02X", r))
		} else {
			out.WriteRune(r)
		}
	}
	return out.String()
}

// uploadFileToCOS 使用 COS PUT Object API 将文件上传到腾讯云对象存储。
// 使用 create_media 返回的临时凭证进行身份验证。
func uploadFileToCOS(cred *ima.Credential, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	// 从路径中提取文件名
	fileName := filePath
	if idx := strings.LastIndex(filePath, "/"); idx >= 0 {
		fileName = filePath[idx+1:]
	}

	hostname := fmt.Sprintf("%s.cos.%s.myqcloud.com", cred.BucketName, cred.Region)
	pathname := "/" + cred.CosKey

	startTime := time.Now().Unix()
	expiredTime := startTime + 3600

	signHeaders := map[string]string{
		"content-length": fmt.Sprintf("%d", len(data)),
		"host":           hostname,
	}

	contentType := detectContentType(fileName)
	auth := buildCOSAuth(cred.SecretID, cred.SecretKey, "PUT", pathname, signHeaders, startTime, expiredTime)

	url := fmt.Sprintf("https://%s%s", hostname, pathname)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("创建 COS 请求失败: %w", err)
	}

	req.ContentLength = int64(len(data))
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", auth)
	req.Header.Set("x-cos-security-token", cred.Token)

	client := &http.Client{Timeout: cosDefaultTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("COS 上传失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("COS 上传失败 (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// mimeTypes 文件扩展名到 MIME 类型的映射表，用于 COS 上传和 API 调用。
var mimeTypes = map[string]string{
	".pdf":  "application/pdf",
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".ppt":  "application/vnd.ms-powerpoint",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".xls":  "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".csv":  "text/csv",
	".md":   "text/markdown",
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".webp": "image/webp",
	".txt":  "text/plain",
	".xmind": "application/x-xmind",
	".mp3":  "audio/mpeg",
	".m4a":  "audio/x-m4a",
	".wav":  "audio/wav",
	".aac":  "audio/aac",
}

// detectContentType 根据文件扩展名推断 MIME 类型。
// 如果无法识别，返回 application/octet-stream。
func detectContentType(fileName string) string {
	lower := strings.ToLower(fileName)
	for ext, mime := range mimeTypes {
		if strings.HasSuffix(lower, ext) {
			return mime
		}
	}
	return "application/octet-stream"
}

// detectMediaType 根据文件扩展名推断 IMA 媒体类型枚举值。
// 参考 IMA OpenAPI 文档中的 MediaType 枚举。
func detectMediaType(fileName string) int32 {
	lower := strings.ToLower(fileName)
	switch {
	case strings.HasSuffix(lower, ".pdf"):
		return 1
	case strings.HasSuffix(lower, ".doc"), strings.HasSuffix(lower, ".docx"):
		return 3
	case strings.HasSuffix(lower, ".ppt"), strings.HasSuffix(lower, ".pptx"):
		return 4
	case strings.HasSuffix(lower, ".xls"), strings.HasSuffix(lower, ".xlsx"), strings.HasSuffix(lower, ".csv"):
		return 5
	case strings.HasSuffix(lower, ".md"):
		return 7
	case strings.HasSuffix(lower, ".png"), strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"), strings.HasSuffix(lower, ".webp"):
		return 9
	case strings.HasSuffix(lower, ".txt"):
		return 13
	case strings.HasSuffix(lower, ".xmind"):
		return 14
	case strings.HasSuffix(lower, ".mp3"), strings.HasSuffix(lower, ".m4a"), strings.HasSuffix(lower, ".wav"), strings.HasSuffix(lower, ".aac"):
		return 15
	}
	return 1 // 默认视为 PDF
}

// GetKBList 获取当前用户可添加内容的知识库列表。
func GetKBList(cli *ima.Client) ([]ima.AddableKnowledgeBaseInfo, error) {
	resp, err := cli.Post("openapi/wiki/v1/get_addable_knowledge_base_list", ima.RequestBody{
		"cursor": "",
		"limit":  50,
	})
	if err != nil {
		return nil, err
	}

	raw, _ := json.Marshal(resp.Data["addable_knowledge_base_list"])
	var list []ima.AddableKnowledgeBaseInfo
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, fmt.Errorf("解析知识库列表失败: %w", err)
	}
	return list, nil
}

// GetKBInfo 获取一个或多个知识库的详细信息。
func GetKBInfo(cli *ima.Client, kbID string) error {
	resp, err := cli.Post("openapi/wiki/v1/get_knowledge_base", ima.RequestBody{
		"ids": []string{kbID},
	})
	if err != nil {
		return err
	}

	result, _ := json.MarshalIndent(resp.Data["infos"], "", "  ")
	fmt.Println(string(result))
	return nil
}

// BrowseKB 浏览知识库的内容（文件和文件夹）。
// folderID 为空时列出根目录。
func BrowseKB(cli *ima.Client, kbID, folderID string) error {
	body := ima.RequestBody{
		"cursor":            "",
		"limit":             20,
		"knowledge_base_id": kbID,
	}
	if folderID != "" {
		body["folder_id"] = folderID
	}

	resp, err := cli.Post("openapi/wiki/v1/get_knowledge_list", body)
	if err != nil {
		return err
	}

	result, _ := json.MarshalIndent(resp.Data, "", "  ")
	fmt.Println(string(result))
	return nil
}

// SearchKB 在知识库中按关键词搜索。
func SearchKB(cli *ima.Client, kbID, query string) error {
	resp, err := cli.Post("openapi/wiki/v1/search_knowledge", ima.RequestBody{
		"cursor":            "",
		"knowledge_base_id": kbID,
		"query":             query,
	})
	if err != nil {
		return err
	}

	result, _ := json.MarshalIndent(resp.Data["info_list"], "", "  ")
	fmt.Println(string(result))
	return nil
}

// AddURL 添加网页 URL 到知识库。folderID 为空时添加到根目录。
func AddURL(cli *ima.Client, kbID, folderID, url string) error {
	body := ima.RequestBody{
		"knowledge_base_id": kbID,
		"urls":              []string{url},
	}
	if folderID != "" {
		body["folder_id"] = folderID
	}

	resp, err := cli.Post("openapi/wiki/v1/import_urls", body)
	if err != nil {
		return err
	}

	result, _ := json.MarshalIndent(resp.Data["results"], "", "  ")
	fmt.Println(string(result))
	return nil
}

// UploadFile 上传本地文件到知识库。
//
// 上传流程（三步）：
//  1. create_media：获取 COS 临时上传凭证
//  2. COS PUT：将文件直传到腾讯云对象存储
//  3. add_knowledge：将媒体关联到知识库
//
// folderID 为空时上传到根目录。
func UploadFile(cli *ima.Client, kbID, filePath, folderID string) error {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("无法访问文件: %w", err)
	}

	// 解析文件名和扩展名
	fileName := filePath
	if idx := strings.LastIndex(filePath, "/"); idx >= 0 {
		fileName = filePath[idx+1:]
	}

	fileSize := uint64(fileInfo.Size())
	contentType := detectContentType(fileName)
	fileExt := ""
	if idx := strings.LastIndex(fileName, "."); idx >= 0 {
		fileExt = fileName[idx+1:]
	}

	// Step 1: 创建媒体，获取 COS 上传凭证
	resp, err := cli.Post("openapi/wiki/v1/create_media", ima.RequestBody{
		"file_name":         fileName,
		"file_size":         fileSize,
		"content_type":      contentType,
		"knowledge_base_id": kbID,
		"file_ext":          fileExt,
	})
	if err != nil {
		return fmt.Errorf("创建媒体失败: %w", err)
	}

	mediaID := ima.GetString(resp.Data, "media_id")
	if mediaID == "" {
		return fmt.Errorf("创建媒体未返回 media_id")
	}

	// 解析 COS 临时凭证
	credRaw, _ := json.Marshal(resp.Data["cos_credential"])
	var cred ima.Credential
	if err := json.Unmarshal(credRaw, &cred); err != nil {
		return fmt.Errorf("解析 COS 凭证失败: %w", err)
	}

	// Step 2: 上传文件到 COS
	fmt.Println("正在上传文件到 COS...")
	if err := uploadFileToCOS(&cred, filePath); err != nil {
		return fmt.Errorf("COS 上传失败: %w", err)
	}
	fmt.Println("COS 上传成功")

	// Step 3: 将媒体关联到知识库
	body := ima.RequestBody{
		"media_type":        detectMediaType(fileName),
		"media_id":          mediaID,
		"title":             fileName,
		"knowledge_base_id": kbID,
		"file_info": ima.FileInfo{
			CosKey:         cred.CosKey,
			FileSize:       fileSize,
			LastModifyTime: fileInfo.ModTime().Unix(),
			FileName:       fileName,
		},
	}
	if folderID != "" {
		body["folder_id"] = folderID
	}
	_, err = cli.Post("openapi/wiki/v1/add_knowledge", body)
	if err != nil {
		return fmt.Errorf("添加知识失败: %w", err)
	}

	fmt.Printf("文件 %s 上传到知识库成功\n", fileName)
	return nil
}

// GetMediaInfo 获取指定媒体的访闻信息（下载链接、笔记 ID 等）。
func GetMediaInfo(cli *ima.Client, mediaID string) error {
	resp, err := cli.Post("openapi/wiki/v1/get_media_info", ima.RequestBody{
		"media_id": mediaID,
	})
	if err != nil {
		return err
	}

	result, _ := json.MarshalIndent(resp.Data, "", "  ")
	fmt.Println(string(result))
	return nil
}
