package main

import (
	"fmt"
	_ "embed"
	"github.com/getlantern/systray"
)

//go:embed icon.png
var iconData []byte


func main() {
	fmt.Println("Hello, 世界")
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetTooltip("SomeHelp Assistant")
	systray.SetIcon(iconData)
}

func onExit() {
	// Nothing here
}

