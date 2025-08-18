package main

import (
	_ "embed"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/atotto/clipboard"
)

//go:embed icon.png
var iconData []byte

// fyne
var fyneApp fyne.App
var clipboardEntry = widget.NewEntry()
var MainWindowWidth float32 = 600
var GapHeight float32 = 50

func main() {
	fmt.Println("Hello, 世界")
	fyneApp = app.New()
	fyneApp.SetIcon(fyne.NewStaticResource("icon", iconData))
	clipboardEntry.SetText("!")

	mainWindow := fyneApp.NewWindow("SomeHelp Assistant")
	mainWindow.Resize(fyne.NewSize(MainWindowWidth, 120))
	mainWindow.SetFixedSize(false)
	setupMainWindow(mainWindow)
	mainWindow.Hide()

	setupSystemTray(mainWindow)
	fyneApp.Run()
}

func setupSystemTray(mainWindow fyne.Window) {
	desk, _ := fyneApp.(desktop.App)
	fmt.Println("Setting up system tray...")
	menu := fyne.NewMenu("SomeHelp?",
		fyne.NewMenuItem("Record your question", func() {
			showRecordingWindow(mainWindow)
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Close record window", func() {
			mainWindow.Hide()
		}),
	)
	desk.SetSystemTrayMenu(menu)
	desk.SetSystemTrayIcon(fyne.NewStaticResource("icon", iconData))
}

func setupMainWindow(window fyne.Window) {
	topLabel := widget.NewLabel("SomeHelp? Assistant")
	window.SetCloseIntercept(func() {
		window.Hide()
	})
	clipboardEntry.MultiLine = true
	clipboardEntry.Wrapping = fyne.TextWrapWord
	scrollContainer := container.NewScroll(clipboardEntry)
	scrollContainer.SetMinSize(fyne.NewSize(MainWindowWidth-20, 30))

	content := container.NewVBox(
		topLabel,
		widget.NewSeparator(),
		scrollContainer,
	)
	window.SetContent(content)
}

func showRecordingWindow(mainWindow fyne.Window) {
	mainWindow.Hide()
	clipboardEntry.SetText(readClipboard())
	lines := strings.Count(clipboardEntry.Text, "\n") + 1
	if lines > 10 {
		lines = 10 // Max 10 lines
	} else if lines < 2 {
		lines = 2
	}
	height := GapHeight + float32(lines*22)
	statusLabel := widget.NewLabel("🎙️ Recording...")

	clipboardContent := widget.NewEntry()
	clipboardContent.MultiLine = true
	clipboardContent.Wrapping = fyne.TextWrapWord
	clipboardContent.SetText(clipboardEntry.Text)
	scrollContainer := container.NewScroll(clipboardContent)
	scrollContainer.SetMinSize(fyne.NewSize(MainWindowWidth-20, height-GapHeight+25))
	mainWindow.Resize(fyne.NewSize(500, height))
	content := container.NewVBox(
		statusLabel,
		widget.NewSeparator(),
		scrollContainer,
	)

	mainWindow.SetContent(content)

	mainWindow.Show()
}

func readClipboard() string {
	text, err := clipboard.ReadAll()
	if err != nil {
		fmt.Println("Error reading clipboard:", err)
		return ""
	}
	return text
}
