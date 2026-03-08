package gui

import "fyne.io/systray"

func setTrayTitleAndTooltip(title, tooltip string) {
	systray.SetTitle(title)
	systray.SetTooltip(tooltip)
}
