package internal

import (
	"os"
	"os/exec"
	"strings"
	"log/slog"
	
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/gordonklaus/portaudio"
)

func Listen(stopChan chan struct{}, recording *[]int16) error {
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

func Transcribe(data []int16) string {
	saveToWAV(data)
	defer os.Remove("recording.wav")

	cmd := exec.Command("whisper-cli", "recording.wav", "-np", "--no-timestamps")
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


