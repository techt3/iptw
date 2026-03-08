//go:build !windows

package main

import (
	"fmt"
	"os"
)

func fatalError(title, message string) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", title, message)
}
