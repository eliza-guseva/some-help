package internal

import (
	"log/slog"
	"strings"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/atotto/clipboard"
)


var MainWindowWidth float32 = 600
var GapHeight float32 = 50


func CreateFyneApp(
	fyneApp fyne.App, 
	stopChan chan struct{}, 
	iconData []byte,
	clipboardEntry *widget.Entry,
) {
	
	fyneApp.SetIcon(fyne.NewStaticResource("icon", iconData))
	// main window setup
	mainWindow := fyneApp.NewWindow("SomeHelp Assistant")
	mainWindow.Resize(fyne.NewSize(MainWindowWidth, 120))
	mainWindow.SetFixedSize(false)
	setupMainWindow(mainWindow, clipboardEntry)
	mainWindow.Hide()

	//system tray and run
	setupSystemTray(fyneApp, mainWindow, stopChan, iconData, clipboardEntry)
}

func setupSystemTray(
	fyneApp fyne.App, 
	mainWindow fyne.Window, 
	stopChan chan struct{},
	iconData []byte,
	clipboardEntry *widget.Entry,
) {
	desk, _ := fyneApp.(desktop.App)
	slog.Info("Setting up system tray")
	menu := fyne.NewMenu("SomeHelp?",
		fyne.NewMenuItem("Record your question", func() {
			ShowRecordingWindow(mainWindow, stopChan, clipboardEntry)
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Close record window", func() {
			mainWindow.Hide()
		}),
	)
	desk.SetSystemTrayMenu(menu)
	desk.SetSystemTrayIcon(fyne.NewStaticResource("icon", iconData))
}

func setupMainWindow(window fyne.Window, clipboardEntry *widget.Entry) {
	topLabel := widget.NewLabel("SomeHelp? Assistant")
	window.SetCloseIntercept(func() {
		window.Hide()
	})
	scrollContainer := container.NewScroll(clipboardEntry)
	scrollContainer.SetMinSize(fyne.NewSize(MainWindowWidth-20, 30))

	content := container.NewVBox(
		topLabel,
		widget.NewSeparator(),
		scrollContainer,
	)
	window.SetContent(content)
}

func ShowRecordingWindow(
	mainWindow fyne.Window, 
	stopChan chan struct{},
	clipboardEntry *widget.Entry,
) {
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
	recording := make([]int16, 0)
	transcribed := ""
	go Listen(stopChan, &recording)

	clipboardContent := widget.NewEntry()
	clipboardContent.MultiLine = true
	clipboardContent.Wrapping = fyne.TextWrapWord
	clipboardContent.SetText(clipboardEntry.Text)
	scrollContainer := container.NewScroll(clipboardContent)
	scrollContainer.SetMinSize(fyne.NewSize(MainWindowWidth-20, height-GapHeight+25))
	mainWindow.Resize(fyne.NewSize(500, height))

	stopButton := widget.NewButton("⏹️ Stop Recording", func() {
		stopChan <- struct{}{}
		statusLabel.SetText("✅ Recording stopped")
		transcribed = Transcribe(recording)
		slog.Info("Transcribed", "transcribed", transcribed)
		clipboard.WriteAll(transcribed + "\n\n" + "CONTEXT:\n" + clipboardEntry.Text)
		PasteToClaudeApp()
		mainWindow.Hide()
	})

	content := container.NewVBox(
		statusLabel,
		stopButton,
		widget.NewSeparator(),
		scrollContainer,
	)

	mainWindow.SetContent(content)

	mainWindow.Show()
}

func readClipboard() string {
	text, err := clipboard.ReadAll()
	if err != nil {
		slog.Error("Error reading clipboard", "error", err)
		return ""
	}
	return text
}
