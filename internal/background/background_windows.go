//go:build windows

package background

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	user32                   = syscall.MustLoadDLL("user32.dll")
	procSystemParametersInfo = user32.MustFindProc("SystemParametersInfoW")

	advapi32            = syscall.MustLoadDLL("advapi32.dll")
	procRegOpenKeyEx    = advapi32.MustFindProc("RegOpenKeyExW")
	procRegQueryValueEx = advapi32.MustFindProc("RegQueryValueExW")
	procRegCloseKey     = advapi32.MustFindProc("RegCloseKey")
)

const (
	spiSetDeskWallpaper = 0x0014
	spifUpdateIniFile   = 0x0001
	spifSendChange      = 0x0002

	hkeyCurrentUser = uintptr(0x80000001)
	keyRead         = 0x20019
	regSZ           = 1
)

// setWindowsBackground calls SystemParametersInfoW directly — no PowerShell
// subprocess, no CMD window flash.
func setWindowsBackground(imagePath string) error {
	pathPtr, err := syscall.UTF16PtrFromString(imagePath)
	if err != nil {
		return fmt.Errorf("failed to encode wallpaper path: %w", err)
	}
	r1, _, e := procSystemParametersInfo.Call(
		spiSetDeskWallpaper,
		0,
		uintptr(unsafe.Pointer(pathPtr)),
		spifUpdateIniFile|spifSendChange,
	)
	if r1 == 0 {
		return fmt.Errorf("SystemParametersInfoW failed: %w", e)
	}
	return nil
}

// RestoreWallpaper restores a wallpaper from a backup file on Windows.
// It calls SystemParametersInfoW directly with the backup path so the file
// is never deleted before Windows reads it (unlike SetDesktopBackground which
// creates a timestamped copy and then deletes it).
func RestoreWallpaper(backupPath string) error {
	return setWindowsBackground(backupPath)
}

// getWindowsCurrentWallpaper reads the current wallpaper path from the registry
// directly — no PowerShell subprocess needed.
func getWindowsCurrentWallpaper() (string, error) {
	subkey, err := syscall.UTF16PtrFromString(`Control Panel\Desktop`)
	if err != nil {
		return "", err
	}
	valueName, err := syscall.UTF16PtrFromString("Wallpaper")
	if err != nil {
		return "", err
	}

	var hKey uintptr
	r1, _, e := procRegOpenKeyEx.Call(
		hkeyCurrentUser,
		uintptr(unsafe.Pointer(subkey)),
		0,
		keyRead,
		uintptr(unsafe.Pointer(&hKey)),
	)
	if r1 != 0 {
		return "", fmt.Errorf("RegOpenKeyEx failed: %w", e)
	}
	defer procRegCloseKey.Call(hKey)

	var dataType uint32
	var bufLen uint32 = 520 // MAX_PATH * 2 bytes
	buf := make([]uint16, bufLen/2)
	r1, _, e = procRegQueryValueEx.Call(
		hKey,
		uintptr(unsafe.Pointer(valueName)),
		0,
		uintptr(unsafe.Pointer(&dataType)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&bufLen)),
	)
	if r1 != 0 {
		return "", fmt.Errorf("RegQueryValueEx failed: %w", e)
	}
	if dataType != regSZ {
		return "", fmt.Errorf("unexpected registry value type: %d", dataType)
	}
	return syscall.UTF16ToString(buf), nil
}
