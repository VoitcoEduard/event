//go:build debug

package hub

import "log"

func DebugLog(format string, args ...interface{}) {
	log.Printf("[DEBUG] "+format, args...)
}
