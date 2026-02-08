//go:build !debug

package main

import "fmt"

func DebugLog(format string, args ...interface{}) {
	fmt.Printf("[LOG] "+format+"\n", args...)
}
