//go:build !linux

package gui

import "github.com/getlantern/systray"

func setTrayTitleAndTooltip(title, tooltip string) {
	systray.SetTitle(title)
	systray.SetTooltip(tooltip)
}
