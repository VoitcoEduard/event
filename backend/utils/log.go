//go:build !debug

package utils

func DebugLog(format string, args ...interface{}) {
	// No-op
}
