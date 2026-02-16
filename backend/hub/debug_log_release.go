//go:build !debug

package hub

// DebugLog is a no-op in production builds (without -tags debug)
func DebugLog(format string, args ...interface{}) {
	// No-op in production
}
