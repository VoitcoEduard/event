//go:build debug

package hub

import "log"

// DebugLog prints debug messages when built with -tags debug
func DebugLog(format string, args ...interface{}) {
	log.Printf("[DEBUG] "+format, args...)
}
