//go:build !debug

package main

func DebugLog(format string, args ...interface{}) {
	// No-op for non-debug builds
}
