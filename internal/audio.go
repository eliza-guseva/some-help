package internal

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/gordonklaus/portaudio"
)

func Listen(commandChan chan RecCommand, recording *[]int16, isListening *bool) error {
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
		case <-commandChan:
			*isListening = false
			err = stream.Stop()
			if err != nil {
				slog.Error("Error stopping stream", "error", err)
			}
			err = stream.Close()
			if err != nil {
				slog.Error("Error closing stream", "error", err)
			}
			err = saveToWAV(*recording)
			if err != nil {
				slog.Error("Error saving recording", "error", err)
			}
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

func Transcribe(ctxPtr *context.Context) string {
	select {
	case <-(*ctxPtr).Done():
		slog.Info("Killing whisper-cli")
		cmd := exec.Command("kill", "-9", "whisper-cli")
		output, err := cmd.CombinedOutput()
		if err != nil {
			slog.Error("Error killing whisper-cli", "error", err, "output", string(output))
		}
		cmd.Run()
		slog.Info("Killed whisper-cli")
		return ""
	default:
		slog.Info("Transcribing")
		cmd := exec.Command("whisper-cli", "recording.wav", "-np", "--no-timestamps")
		output, err := cmd.CombinedOutput()
		if err != nil {
			slog.Error("Error transcribing", "error", err, "output", string(output))
			return ""
		}
		slog.Info("Transcribed")
		err = os.Remove("recording.wav")
		if err != nil {
			slog.Error("Error removing recording.wav", "error", err)
		}
		return strings.TrimSpace(string(output))
	}
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
