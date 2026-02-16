//go:build debug

package utils

import (
	"log"
)

func DebugLog(format string, args ...interface{}) {
	log.Printf("[DEBUG] "+format, args...)
}
