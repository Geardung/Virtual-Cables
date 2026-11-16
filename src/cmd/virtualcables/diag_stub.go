//go:build !windows

package main

import "fmt"

func portConflictHint(address string) string {
	return fmt.Sprintf("USB/IP address %s cannot be opened; it is held by another process. Virtual Cables will try another address.", address)
}
