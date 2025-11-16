//go:build !windows
// +build !windows

package singleinstance

// EnsureSingleInstance 在非 windows 平台上为 stub，不阻止程序运行。
func EnsureSingleInstance() error {
	// 如果需要，可在这里实现 Unix/macOS 的单实例逻辑（例如使用 pidfile 或 unix domain socket）。
	return nil
}
