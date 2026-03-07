//go:build linux

package gui

func setTrayTitleAndTooltip(title, tooltip string) {
	// SetTitle and SetTooltip are often undefined on Linux in github.com/getlantern/systray
}
