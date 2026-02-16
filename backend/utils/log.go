//go:build !debug

package utils

import "fmt"

func DebugLog(format string, args ...interface{}) {
	fmt.Printf("[LOG] "+format+"\n", args...)
}
