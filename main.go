package main

import (
	_ "embed"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/widget"
	"github.com/eliza-guseva/some-help/internal"
)

//go:embed icon.png
var iconData []byte
var fyneApp fyne.App
var clipboardEntry = widget.NewEntry()



// fyne
func main() {
	// general setup
	fmt.Println("Hello, 世界")
	fyneApp = app.New()
	var recorderState internal.RecorderState
	clipboardEntry.MultiLine = true
	clipboardEntry.Wrapping = fyne.TextWrapWord
	clipboardEntry.SetText("!")

	internal.CreateFyneApp(&recorderState, fyneApp, iconData, clipboardEntry)

	fyneApp.Run()
}
