package main

import (
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/atotto/clipboard"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/gordonklaus/portaudio"
)

//go:embed icon.png
var iconData []byte

// fyne
var fyneApp fyne.App
var clipboardEntry = widget.NewEntry()
var MainWindowWidth float32 = 600
var GapHeight float32 = 50

func main() {
	// general setup
	fmt.Println("Hello, 世界")
	fyneApp = app.New()
	fyneApp.SetIcon(fyne.NewStaticResource("icon", iconData))
	clipboardEntry.SetText("!")

	//audio
	stopChan := make(chan struct{})

	// main window setup
	mainWindow := fyneApp.NewWindow("SomeHelp Assistant")
	mainWindow.Resize(fyne.NewSize(MainWindowWidth, 120))
	mainWindow.SetFixedSize(false)
	setupMainWindow(mainWindow)
	mainWindow.Hide()

	//system tray and run
	setupSystemTray(mainWindow, stopChan)
	fyneApp.Run()
}

func setupSystemTray(mainWindow fyne.Window, stopChan chan struct{}) {
	desk, _ := fyneApp.(desktop.App)
	fmt.Println("Setting up system tray...")
	menu := fyne.NewMenu("SomeHelp?",
		fyne.NewMenuItem("Record your question", func() {
			showRecordingWindow(mainWindow, stopChan)
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

func showRecordingWindow(mainWindow fyne.Window, stopChan chan struct{}) {
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
	go listen(stopChan, &recording)

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
		transcribed = transcribe(recording)
		slog.Info("Transcribed", "transcribed", transcribed)
		clipboard.WriteAll(transcribed + "\n" + "CONTEXT: " + clipboardEntry.Text)
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

func listen(stopChan chan struct{}, recording *[]int16) error {
	portaudio.Initialize()
	defer portaudio.Terminate()

	inputDevice, err := portaudio.DefaultInputDevice()
	if err != nil {
		slog.Error("Error getting default input device", "error", err)
		return err
	}

	inputParams := portaudio.StreamParameters{
		Input: portaudio.StreamDeviceParameters{
			Device:   inputDevice,
			Channels: 1,
			Latency:  inputDevice.DefaultLowInputLatency,
		},
		SampleRate:      16000,
		FramesPerBuffer: 1024,
	}
	buffer := make([]int16, 1024)
	stream, err := portaudio.OpenStream(inputParams, buffer)
	if err != nil {
		slog.Error("Error opening stream", "error", err)
		return err
	}

	err = stream.Start()
	if err != nil {
		slog.Error("Error starting stream", "error", err)
		return err
	}

	for {
		select {
		case <-stopChan:
			stream.Stop()
			stream.Close()
			return nil
		default:
			err = stream.Read()
			if err != nil {
				slog.Error("Error reading stream", "error", err)
				return err
			}
			*recording = append(*recording, buffer...)
		}
	}
}

func transcribe(data []int16) string {
	saveToWAV(data)
	defer os.Remove("recording.wav")

	cmd := exec.Command("whisper-cli", "recording.wav", "-np")
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("Error transcribing", "error", err, "output", string(output))
		return ""
	}
	return strings.TrimSpace(string(output))
}

func saveToWAV(data []int16) error {
	intData := make([]int, len(data))
	for i, v := range data {
		intData[i] = int(v)
	}

	file, err := os.Create("recording.wav")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := wav.NewEncoder(file, 16000, 16, 1, 1)
	if err != nil {
		return err
	}
	defer encoder.Close()

	buffer := audio.IntBuffer{
		Data: intData,
		Format: &audio.Format{
			SampleRate:  16000,
			NumChannels: 1,
		},
	}
	return encoder.Write(&buffer)
}
