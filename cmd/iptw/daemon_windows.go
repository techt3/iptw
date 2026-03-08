//go:build windows

package main

// maybeDaemonize is a no-op on Windows.  Binaries built with -H windowsgui
// already have no console window, so there is nothing to detach from.
func maybeDaemonize(_ bool) {}
