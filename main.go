package main

import (
	"fmt"
	"os"
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
	mRecord := systray.AddMenuItem("🎙️ Record your question", "Record your question")

	mQuit := systray.AddMenuItem("❌ Quit the assistant", "Quit the assistant")

	go func() {
		for {
			select {
			case <-mRecord.ClickedCh:
				fmt.Println("Make belief recording")
			case <-mQuit.ClickedCh:
				fmt.Println("Quitting the assistant")
				systray.Quit()
			}
		}
	}()
}

func onExit() {
	// Nothing here
}

