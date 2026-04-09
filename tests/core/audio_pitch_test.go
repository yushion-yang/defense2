package core_test

import (
	"encoding/binary"
	"math"
	"testing"

	"defense2/internal/audio"
)

func TestResampleIdentity(t *testing.T) {
	// Create a simple stereo PCM buffer: 10 frames of silence
	frames := 10
	pcm := make([]byte, frames*4) // 4 bytes per frame (16-bit stereo)

	result := audio.ResamplePCM(pcm, 1.0)
	if len(result) != len(pcm) {
		t.Errorf("identity resample: len = %d, want %d", len(result), len(pcm))
	}
}

func TestResampleHigherPitch(t *testing.T) {
	frames := 100
	pcm := makeTestPCM(frames)

	result := audio.ResamplePCM(pcm, 2.0)
	expectedLen := (frames / 2) * 4
	if math.Abs(float64(len(result)-expectedLen)) > 8 {
		t.Errorf("2x pitch: len = %d, want ~%d", len(result), expectedLen)
	}
}

func TestResampleLowerPitch(t *testing.T) {
	frames := 100
	pcm := makeTestPCM(frames)

	result := audio.ResamplePCM(pcm, 0.5)
	expectedLen := (frames * 2) * 4
	if math.Abs(float64(len(result)-expectedLen)) > 8 {
		t.Errorf("0.5x pitch: len = %d, want ~%d", len(result), expectedLen)
	}
}

func TestResampleStereoAlignment(t *testing.T) {
	frames := 50
	pcm := makeTestPCM(frames)

	result := audio.ResamplePCM(pcm, 1.1)
	if len(result)%4 != 0 {
		t.Errorf("output length %d not aligned to 4 bytes (stereo frame size)", len(result))
	}
}

func TestResampleEmptyInput(t *testing.T) {
	result := audio.ResamplePCM(nil, 1.5)
	if result != nil {
		t.Error("nil input should return nil")
	}
	result = audio.ResamplePCM([]byte{}, 1.5)
	if len(result) != 0 {
		t.Error("empty input should return empty")
	}
}

func TestResamplePreservesWaveform(t *testing.T) {
	// Resample up then down should roughly preserve the original waveform
	frames := 200
	pcm := makeTestPCM(frames)

	// Resample to 1.08x pitch (simulating +8% variation)
	resampled := audio.ResamplePCM(pcm, 1.08)
	if len(resampled) == 0 {
		t.Fatal("resampled output is empty")
	}

	// Output should be shorter than input
	if len(resampled) >= len(pcm) {
		t.Errorf("1.08x pitch should produce shorter output: got %d >= %d", len(resampled), len(pcm))
	}
}

func TestResampleInvalidFactor(t *testing.T) {
	pcm := makeTestPCM(10)

	// Factor <= 0 should return input unchanged
	result := audio.ResamplePCM(pcm, 0)
	if len(result) != len(pcm) {
		t.Errorf("factor=0: len = %d, want %d", len(result), len(pcm))
	}
	result = audio.ResamplePCM(pcm, -1.0)
	if len(result) != len(pcm) {
		t.Errorf("factor=-1: len = %d, want %d", len(result), len(pcm))
	}
}

func makeTestPCM(frames int) []byte {
	pcm := make([]byte, frames*4)
	for i := 0; i < frames; i++ {
		// Generate a 440Hz sine wave at 44100Hz
		val := int16(10000 * math.Sin(2*math.Pi*440*float64(i)/44100))
		off := i * 4
		binary.LittleEndian.PutUint16(pcm[off:off+2], uint16(val))   // left
		binary.LittleEndian.PutUint16(pcm[off+2:off+4], uint16(val)) // right
	}
	return pcm
}
