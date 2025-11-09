package webdav

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/lich0821/ccNexus/internal/config"

	"github.com/studio-b12/gowebdav"
)

// Client WebDAV 客户端
type Client struct {
	client *gowebdav.Client
	config *config.WebDAVConfig
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

	// 设置默认路径
	if cfg.ConfigPath == "" {
		cfg.ConfigPath = "/ccNexus/config"
	}
	if cfg.StatsPath == "" {
		cfg.StatsPath = "/ccNexus/stats"
	}

	return &Client{
		client: client,
		config: cfg,
	}, nil
}

// TestConnection 测试 WebDAV 连接
func (c *Client) TestConnection() *TestResult {
	// 尝试连接并读取根目录
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
	// 检查目录是否存在
	info, err := c.client.Stat(dirPath)
	if err == nil {
		// 目录存在
		if !info.IsDir() {
			return fmt.Errorf("路径 %s 已存在但不是目录", dirPath)
		}
		return nil
	}

	// 目录不存在，创建它
	err = c.client.MkdirAll(dirPath, 0755)
	if err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	return nil
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

	// 读取目录内容
	files, err := c.client.ReadDir(backupPath)
	if err != nil {
		// 如果目录不存在，返回空列表
		if strings.Contains(err.Error(), "404") {
			return []BackupFile{}, nil
		}
		return nil, fmt.Errorf("读取目录失败: %v", err)
	}

	// 转换为 BackupFile 列表
	var backups []BackupFile
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		// 只列出 .json 文件
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		backups = append(backups, BackupFile{
			Filename: file.Name(),
			Size:     file.Size(),
			ModTime:  file.ModTime(),
		})
	}

	// 按修改时间降序排序（最新的在前）
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].ModTime.After(backups[j].ModTime)
	})

	return backups, nil
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
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("删除失败: %s", strings.Join(errors, "; "))
	}

	return nil
}
