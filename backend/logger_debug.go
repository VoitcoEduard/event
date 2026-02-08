//go:build debug

package main

import (
	"log"
)

func DebugLog(format string, args ...interface{}) {
	log.Printf("[DEBUG] "+format, args...)
}
