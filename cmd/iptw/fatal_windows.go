//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32      = syscall.MustLoadDLL("user32.dll")
	messageBoxW = user32.MustFindProc("MessageBoxW")
)

// fatalError shows a Windows MessageBox with the error and writes it to a log
// file so it is visible even when the binary is built with -H windowsgui.
func fatalError(title, message string) {
	writeErrorLog(title, message)
	titleStr, _ := syscall.UTF16PtrFromString("IPTW – " + title)
	msgStr, _ := syscall.UTF16PtrFromString(message)
	// MB_OK | MB_ICONERROR
	messageBoxW.Call(0, uintptr(unsafe.Pointer(msgStr)), uintptr(unsafe.Pointer(titleStr)), 0x10)
}

func writeErrorLog(title, message string) {
	logPath := errorLogPath()
	_ = os.MkdirAll(filepath.Dir(logPath), 0755)
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "[%s] %s: %s\n", time.Now().Format("2006-01-02 15:04:05"), title, message)
}

func errorLogPath() string {
	if appData := os.Getenv("APPDATA"); appData != "" {
		return filepath.Join(appData, "iptw", "iptw-error.log")
	}
	return filepath.Join(os.Getenv("USERPROFILE"), "iptw-error.log")
}
