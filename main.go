package main

import (
	_ "embed"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"fyne.io/fyne/v2/app"
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

	clipboardEntry.MultiLine = true
	clipboardEntry.Wrapping = fyne.TextWrapWord
	clipboardEntry.SetText("!")	

	//audio
	stopChan := make(chan struct{})

	internal.CreateFyneApp(fyneApp, stopChan, iconData, clipboardEntry)
	

	fyneApp.Run()
}



