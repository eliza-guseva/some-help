package internal

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/atotto/clipboard"
)

var MainWindowWidth float32 = 600
var GapHeight float32 = 50
var currentCommandChan chan RecCommand

type CommandType int

const (
	Stop CommandType = iota
	Kill
)

type RecCommand struct {
	Type CommandType
}

func CreateFyneApp(
	commandChan chan RecCommand,
	fyneApp fyne.App,
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
	setupSystemTray(commandChan, fyneApp, mainWindow, iconData, clipboardEntry)
}

func setupSystemTray(
	commandChan chan RecCommand,
	fyneApp fyne.App,
	mainWindow fyne.Window,
	iconData []byte,
	clipboardEntry *widget.Entry,
) {
	desk, _ := fyneApp.(desktop.App)
	slog.Info("Setting up system tray")
	menu := fyne.NewMenu("SomeHelp?",
		fyne.NewMenuItem("Record your question", func() {
			ShowRecordingWindow(commandChan, mainWindow, clipboardEntry)
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
	commandChan chan RecCommand,
	mainWindow fyne.Window,
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

	// context
	var ctxPtr *context.Context
	var cancel context.CancelFunc
	initialCtx, cancel := context.WithCancel(context.Background())

	ctxPtr = &initialCtx

	isListening := true
	slog.Info("Starting Listen, isListening", "isListening", isListening)
	go Listen(commandChan, &recording, &isListening)

	clipboardContent := widget.NewEntry()
	clipboardContent.MultiLine = true
	clipboardContent.Wrapping = fyne.TextWrapWord
	clipboardContent.SetText(clipboardEntry.Text)
	scrollContainer := container.NewScroll(clipboardContent)
	scrollContainer.SetMinSize(fyne.NewSize(MainWindowWidth-20, height-GapHeight+25))
	mainWindow.Resize(fyne.NewSize(500, height))

	var transcribeSendButton *widget.Button
	transcribeSendButton = widget.NewButton("🤖 Transcribe and Send to AI", func() {
		slog.Info("Transcribe and Send to AI Button, isListening", "isListening", isListening)
		if isListening {
			commandChan <- RecCommand{Type: Stop}
		}
		go processRecordingAndSendToAI(
			ctxPtr,
			&recording,
			&transcribed,
			clipboardEntry.Text,
			statusLabel,
			mainWindow,
			transcribeSendButton,
		)
	})
	var stopButton *widget.Button
	stopButton = widget.NewButton("⏹️ Stop!", func() {
		slog.Info("Stopping Button, isListening", "isListening", isListening)
		if isListening {
			commandChan <- RecCommand{Type: Stop}
			isListening = false
		}
		cancel()
		statusLabel.SetText("❌ Cancelled")
		stopButton.SetText("🔄 Re-record")
		newCtx, NewCancel := context.WithCancel(context.Background())
		*ctxPtr = newCtx
		cancel = NewCancel
		slog.Info("Stopped with context", "context", *ctxPtr)
	})

	buttons := container.NewHBox(stopButton, transcribeSendButton)

	content := container.NewVBox(
		statusLabel,
		buttons,
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

func processRecordingAndSendToAI(
	ctxPtr *context.Context,
	recording *[]int16,
	transcribed *string,
	contetxText string,
	statusLabel *widget.Label,
	mainWindow fyne.Window,
	transcribeSendButton *widget.Button,
) {
	slog.Info("Processing recording and sending to AI")
	// after we are done let's re-enable the button
	defer func() {
		fyne.DoAndWait(func() {
			transcribeSendButton.Enable()
		})
	}()

	// but first we disable it
	fyne.DoAndWait(func() {
		transcribeSendButton.Disable()
	})

	result, ok := transcribeWithCancellation(ctxPtr, recording, statusLabel)

	if !ok {
		return
	}
	*transcribed = result

	fyne.DoAndWait(func() {
		statusLabel.SetText("🤖 Sending to Claude...")
	})

	content := *transcribed + "\n\n" + "CONTEXT:\n" + contetxText
	clipboard.WriteAll(content)

	err := PasteToClaudeApp()
	if err != nil {
		slog.Error("Failed to paste to Claude", "error", err)
		fyne.DoAndWait(func() {
			statusLabel.SetText("❌ Failed to send to Claude (copied to clipboard)")
		})
	} else {
		fyne.DoAndWait(func() {
			statusLabel.SetText("✅ Sent to Claude successfully!")
		})
	}

	time.AfterFunc(100*time.Millisecond, func() {
		fyne.DoAndWait(func() {
			mainWindow.Hide()
		})
	})
}

func transcribeWithCancellation(
	ctxPtr *context.Context,
	recording *[]int16,
	statusLabel *widget.Label,
) (string, bool) {
	updateStatus(statusLabel, "⏳ Transcribing...")

	transcrResult := make(chan string, 1)
	go func() {
		result := Transcribe(ctxPtr)
		transcrResult <- result
	}()

	select {
	case <-(*ctxPtr).Done():
		slog.Info("Cancelled")
		updateStatus(statusLabel, "❌ Cancelled")
		return "", false
	case result := <-transcrResult:
		slog.Info("Transcribed")
		return result, true
	}
}

func updateStatus(statusLabel *widget.Label, text string) {
	fyne.DoAndWait(func() {
		statusLabel.SetText(text)
	})
}
