package webdav

import (
	"time"

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/lich0821/ccNexus/internal/proxy"
)

// BackupFile 备份文件信息
type BackupFile struct {
	Filename string    `json:"filename"` // 文件名
	Size     int64     `json:"size"`     // 文件大小（字节）
	ModTime  time.Time `json:"modTime"`  // 修改时间
}

// BackupData 备份数据结构（包含配置和统计）
type BackupData struct {
	Config     *config.Config `json:"config"`     // 配置数据
	Stats      *proxy.Stats   `json:"stats"`      // 统计数据
	BackupTime time.Time      `json:"backupTime"` // 备份时间
	Version    string         `json:"version"`    // ccNexus 版本
}

// ConflictInfo 冲突信息
type ConflictInfo struct {
	HasConflict        bool      `json:"hasConflict"`        // 是否存在冲突
	LocalEndpointCount int       `json:"localEndpointCount"` // 本地端点数量
	RemoteEndpointCount int      `json:"remoteEndpointCount"` // 远程端点数量
	LocalModTime       time.Time `json:"localModTime"`       // 本地修改时间
	RemoteModTime      time.Time `json:"remoteModTime"`      // 远程修改时间
	LocalPort          int       `json:"localPort"`          // 本地端口
	RemotePort         int       `json:"remotePort"`         // 远程端口
}

// TestResult WebDAV 连接测试结果
type TestResult struct {
	Success bool   `json:"success"` // 是否成功
	Message string `json:"message"` // 消息
}
