package webdav

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/lich0821/ccNexus/internal/proxy"
)

// Manager WebDAV 同步管理器
type Manager struct {
	client *Client
}

// NewManager 创建同步管理器
func NewManager(client *Client) *Manager {
	return &Manager{
		client: client,
	}
}

// BackupConfig 备份配置到 WebDAV
func (m *Manager) BackupConfig(cfg *config.Config, stats *proxy.Stats, version string, filename string) error {
	// 创建备份数据
	backupData := &BackupData{
		Config:     cfg,
		Stats:      stats,
		BackupTime: time.Now(),
		Version:    version,
	}

	// 序列化为 JSON
	data, err := json.MarshalIndent(backupData, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化备份数据失败: %v", err)
	}

	// 上传到 WebDAV（config 备份）
	if err := m.client.UploadBackup(filename, data, true); err != nil {
		return err
	}

	return nil
}

// RestoreConfig 从 WebDAV 恢复配置
func (m *Manager) RestoreConfig(filename string, configPath, statsPath string) (*config.Config, *proxy.Stats, error) {
	// 下载备份文件
	data, err := m.client.DownloadBackup(filename, true)
	if err != nil {
		return nil, nil, err
	}

	// 解析备份数据
	var backupData BackupData
	if err := json.Unmarshal(data, &backupData); err != nil {
		return nil, nil, fmt.Errorf("解析备份数据失败: %v", err)
	}

	if backupData.Config == nil {
		return nil, nil, fmt.Errorf("备份数据中没有配置信息")
	}

	// 验证配置有效性
	if err := backupData.Config.Validate(); err != nil {
		return nil, nil, fmt.Errorf("备份配置无效: %v", err)
	}

	// 保存配置到文件
	if err := backupData.Config.Save(configPath); err != nil {
		return nil, nil, fmt.Errorf("保存配置失败: %v", err)
	}

	// 保存统计数据（如果有）
	if backupData.Stats != nil {
		backupData.Stats.SetStatsPath(statsPath)
		if err := backupData.Stats.Save(); err != nil {
			return nil, nil, fmt.Errorf("保存统计数据失败: %v", err)
		}
	}

	return backupData.Config, backupData.Stats, nil
}

// DetectConflict 检测本地配置和远程备份之间的冲突
func (m *Manager) DetectConflict(localConfig *config.Config, filename string) (*ConflictInfo, error) {
	// 下载远程备份
	data, err := m.client.DownloadBackup(filename, true)
	if err != nil {
		return nil, err
	}

	// 解析远程备份
	var backupData BackupData
	if err := json.Unmarshal(data, &backupData); err != nil {
		return nil, fmt.Errorf("解析备份数据失败: %v", err)
	}

	if backupData.Config == nil {
		return nil, fmt.Errorf("备份数据中没有配置信息")
	}

	// 获取本地配置信息
	localEndpoints := localConfig.GetEndpoints()
	localPort := localConfig.GetPort()

	// 获取本地配置文件修改时间
	configPath, err := config.GetConfigPath()
	var localModTime time.Time
	if err == nil {
		if info, err := os.Stat(configPath); err == nil {
			localModTime = info.ModTime()
		}
	}

	// 获取远程配置信息
	remoteEndpoints := backupData.Config.GetEndpoints()
	remotePort := backupData.Config.GetPort()
	remoteModTime := backupData.BackupTime

	// 判断是否存在冲突
	hasConflict := false
	if len(localEndpoints) != len(remoteEndpoints) {
		hasConflict = true
	} else if localPort != remotePort {
		hasConflict = true
	} else {
		// 检查端点是否有变化
		for i := range localEndpoints {
			if localEndpoints[i].Name != remoteEndpoints[i].Name ||
				localEndpoints[i].APIUrl != remoteEndpoints[i].APIUrl ||
				localEndpoints[i].Enabled != remoteEndpoints[i].Enabled {
				hasConflict = true
				break
			}
		}
	}

	return &ConflictInfo{
		HasConflict:         hasConflict,
		LocalEndpointCount:  len(localEndpoints),
		RemoteEndpointCount: len(remoteEndpoints),
		LocalModTime:        localModTime,
		RemoteModTime:       remoteModTime,
		LocalPort:           localPort,
		RemotePort:          remotePort,
	}, nil
}

// ListConfigBackups 列出配置备份
func (m *Manager) ListConfigBackups() ([]BackupFile, error) {
	return m.client.ListBackups(true)
}

// DeleteConfigBackups 删除配置备份
func (m *Manager) DeleteConfigBackups(filenames []string) error {
	return m.client.DeleteBackups(filenames, true)
}
