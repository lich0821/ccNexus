package storage

import "github.com/lich0821/ccNexus/internal/config"

// ConfigStorageAdapter adapts SQLiteStorage to config.StorageAdapter interface
type ConfigStorageAdapter struct {
	storage *SQLiteStorage
}

// NewConfigStorageAdapter creates a new adapter
func NewConfigStorageAdapter(storage *SQLiteStorage) *ConfigStorageAdapter {
	return &ConfigStorageAdapter{storage: storage}
}

// GetEndpoints returns endpoints in config format
func (a *ConfigStorageAdapter) GetEndpoints() ([]config.StorageEndpoint, error) {
	endpoints, err := a.storage.GetEndpoints()
	if err != nil {
		return nil, err
	}

	result := make([]config.StorageEndpoint, len(endpoints))
	for i, ep := range endpoints {
		result[i] = config.StorageEndpoint{
			Name:        ep.Name,
			APIUrl:      ep.APIUrl,
			APIKey:      ep.APIKey,
			Enabled:     ep.Enabled,
			Transformer: ep.Transformer,
			Model:       ep.Model,
			Remark:      ep.Remark,
		}
	}
	return result, nil
}

// SaveEndpoint saves an endpoint
func (a *ConfigStorageAdapter) SaveEndpoint(ep *config.StorageEndpoint) error {
	endpoint := &Endpoint{
		Name:        ep.Name,
		APIUrl:      ep.APIUrl,
		APIKey:      ep.APIKey,
		Enabled:     ep.Enabled,
		Transformer: ep.Transformer,
		Model:       ep.Model,
		Remark:      ep.Remark,
	}
	return a.storage.SaveEndpoint(endpoint)
}

// UpdateEndpoint updates an endpoint
func (a *ConfigStorageAdapter) UpdateEndpoint(ep *config.StorageEndpoint) error {
	endpoint := &Endpoint{
		Name:        ep.Name,
		APIUrl:      ep.APIUrl,
		APIKey:      ep.APIKey,
		Enabled:     ep.Enabled,
		Transformer: ep.Transformer,
		Model:       ep.Model,
		Remark:      ep.Remark,
	}
	return a.storage.UpdateEndpoint(endpoint)
}

// DeleteEndpoint deletes an endpoint
func (a *ConfigStorageAdapter) DeleteEndpoint(name string) error {
	return a.storage.DeleteEndpoint(name)
}

// GetConfig gets a config value
func (a *ConfigStorageAdapter) GetConfig(key string) (string, error) {
	return a.storage.GetConfig(key)
}

// SetConfig sets a config value
func (a *ConfigStorageAdapter) SetConfig(key, value string) error {
	return a.storage.SetConfig(key, value)
}
