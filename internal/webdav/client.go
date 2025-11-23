package webdav

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/lich0821/ccNexus/internal/config"

	"github.com/studio-b12/gowebdav"
)

// Client WebDAV 客户端
type Client struct {
	client     *gowebdav.Client
	config     *config.WebDAVConfig
	httpClient *http.Client
}

// XML structures for PROPFIND response
type propfindResponse struct {
	XMLName   xml.Name   `xml:"multistatus"`
	Responses []response `xml:"response"`
}

type response struct {
	Href     string   `xml:"href"`
	Propstat propstat `xml:"propstat"`
}

type propstat struct {
	Prop prop   `xml:"prop"`
	Status string `xml:"status"`
}

type prop struct {
	GetLastModified  string       `xml:"getlastmodified"`
	GetContentLength int64        `xml:"getcontentlength"`
	ResourceType     resourceType `xml:"resourcetype"`
}

type resourceType struct {
	Collection *struct{} `xml:"collection"`
}

// NewClient 创建 WebDAV 客户端
func NewClient(cfg *config.WebDAVConfig) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("WebDAV config is nil")
	}

	if cfg.URL == "" {
		return nil, fmt.Errorf("WebDAV URL is empty")
	}

	// 创建 WebDAV 客户端
	client := gowebdav.NewClient(cfg.URL, cfg.Username, cfg.Password)

	// 设置超时时间（30秒）
	client.SetTimeout(30 * time.Second)

	// 创建独立的 HTTP 客户端用于自定义请求
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 设置默认路径
	if cfg.ConfigPath == "" {
		cfg.ConfigPath = "/ccNexus/config"
	}
	if cfg.StatsPath == "" {
		cfg.StatsPath = "/ccNexus/stats"
	}

	return &Client{
		client:     client,
		config:     cfg,
		httpClient: httpClient,
	}, nil
}

// TestConnection 测试 WebDAV 连接
func (c *Client) TestConnection() *TestResult {
	err := c.client.Connect()
	if err != nil {
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("连接失败: %v", err),
		}
	}

	return &TestResult{
		Success: true,
		Message: "连接成功",
	}
}

// ensureDirectory 确保目录存在
func (c *Client) ensureDirectory(dirPath string) error {
	err := c.client.MkdirAll(dirPath, 0755)

	if err == nil {
		return nil
	}

	// 检查错误类型
	errStr := err.Error()

	// 405 Method Not Allowed 或包含 "exists" 表示目录已存在，这是正常的
	if strings.Contains(errStr, "405") ||
		strings.Contains(errStr, "exists") ||
		strings.Contains(errStr, "Method Not Allowed") {
		return nil
	}

	return fmt.Errorf("创建目录失败: %v", err)
}

// UploadBackup 上传备份文件
func (c *Client) UploadBackup(filename string, data []byte, isConfig bool) error {
	// 选择备份路径
	backupPath := c.config.StatsPath
	if isConfig {
		backupPath = c.config.ConfigPath
	}

	// 确保目录存在
	if err := c.ensureDirectory(backupPath); err != nil {
		return err
	}

	// 构建完整路径
	remotePath := path.Join(backupPath, filename)

	// 上传文件
	err := c.client.Write(remotePath, data, 0644)
	if err != nil {
		return fmt.Errorf("上传文件失败: %v", err)
	}

	return nil
}

// ListBackups 列出备份文件
func (c *Client) ListBackups(isConfig bool) ([]BackupFile, error) {
	// 选择备份路径
	backupPath := c.config.StatsPath
	if isConfig {
		backupPath = c.config.ConfigPath
	}

	// 使用自定义 PROPFIND 请求

	// 构建完整 URL
	fullURL := c.config.URL + strings.TrimPrefix(backupPath, "/")
	if !strings.HasSuffix(fullURL, "/") {
		fullURL += "/"
	}

	// 创建 PROPFIND 请求
	req, err := http.NewRequest("PROPFIND", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Depth", "1")
	req.Header.Set("Content-Type", "application/xml")
	req.SetBasicAuth(c.config.Username, c.config.Password)

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()


	// 检查状态码
	if resp.StatusCode == 404 {
		return []BackupFile{}, nil
	}

	if resp.StatusCode != 207 { // 207 Multi-Status
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}


	// 解析 XML
	var propfind propfindResponse
	if err := xml.Unmarshal(body, &propfind); err != nil {
		return nil, fmt.Errorf("解析XML失败: %v", err)
	}


	// 转换为 BackupFile 列表
	var backups []BackupFile
	for _, resp := range propfind.Responses {
		// 跳过目录本身
		if resp.Propstat.Prop.ResourceType.Collection != nil {
			continue
		}

		// 提取文件名
		filename := path.Base(resp.Href)
		if filename == "" || filename == "/" {
			continue
		}

		// 只列出 .json 和 .db 文件（但排除 .meta.json 文件）
		if strings.HasSuffix(filename, ".meta.json") {
			continue
		}
		if !strings.HasSuffix(filename, ".json") && !strings.HasSuffix(filename, ".db") {
			continue
		}

		// 解析修改时间
		modTime, err := time.Parse(time.RFC1123, resp.Propstat.Prop.GetLastModified)
		if err != nil {
			// 尝试其他时间格式
			modTime, _ = time.Parse(time.RFC1123Z, resp.Propstat.Prop.GetLastModified)
		}

		backups = append(backups, BackupFile{
			Filename: filename,
			Size:     resp.Propstat.Prop.GetContentLength,
			ModTime:  modTime,
		})

	}

	// 按修改时间降序排序（最新的在前）
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].ModTime.After(backups[j].ModTime)
	})

	return backups, nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// DownloadBackup 下载备份文件
func (c *Client) DownloadBackup(filename string, isConfig bool) ([]byte, error) {
	// 选择备份路径
	backupPath := c.config.StatsPath
	if isConfig {
		backupPath = c.config.ConfigPath
	}

	// 构建完整路径
	remotePath := path.Join(backupPath, filename)

	// 下载文件
	data, err := c.client.Read(remotePath)
	if err != nil {
		return nil, fmt.Errorf("下载文件失败: %v", err)
	}

	return data, nil
}

// DeleteBackups 删除备份文件
func (c *Client) DeleteBackups(filenames []string, isConfig bool) error {
	if len(filenames) == 0 {
		return nil
	}

	// 选择备份路径
	backupPath := c.config.StatsPath
	if isConfig {
		backupPath = c.config.ConfigPath
	}

	var errors []string
	for _, filename := range filenames {
		remotePath := path.Join(backupPath, filename)
		err := c.client.Remove(remotePath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", filename, err))
		} else {
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("删除失败: %s", strings.Join(errors, "; "))
	}

	return nil
}
