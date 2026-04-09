// resample.go — PCM pitch shifting via linear-interpolation resampling.
// Used to add subtle pitch variation to combat SFX, preventing auditory fatigue.
package audio

import (
	"encoding/binary"
	"math"
)

// ResamplePCM resamples 16-bit stereo PCM data by the given pitch factor.
// factor > 1.0 = higher pitch (shorter output), factor < 1.0 = lower pitch (longer output).
// Uses linear interpolation for smooth resampling.
func ResamplePCM(pcm []byte, factor float64) []byte {
	if len(pcm) == 0 || factor <= 0 {
		return pcm
	}
	// Skip if factor is effectively 1.0
	if math.Abs(factor-1.0) < 0.001 {
		return pcm
	}

	const bytesPerSample = 2 // 16-bit
	const channels = 2       // stereo
	const frameSize = bytesPerSample * channels // 4 bytes per frame

	numFrames := len(pcm) / frameSize
	if numFrames < 2 {
		return pcm
	}

	outFrames := int(float64(numFrames) / factor)
	if outFrames < 1 {
		outFrames = 1
	}
	out := make([]byte, outFrames*frameSize)

	for i := 0; i < outFrames; i++ {
		srcPos := float64(i) * factor
		srcIdx := int(srcPos)
		frac := srcPos - float64(srcIdx)

		if srcIdx >= numFrames-1 {
			srcIdx = numFrames - 2
			frac = 1.0
		}

		for ch := 0; ch < channels; ch++ {
			off0 := srcIdx*frameSize + ch*bytesPerSample
			off1 := (srcIdx+1)*frameSize + ch*bytesPerSample

			s0 := int16(binary.LittleEndian.Uint16(pcm[off0 : off0+2]))
			s1 := int16(binary.LittleEndian.Uint16(pcm[off1 : off1+2]))

			// Linear interpolation
			val := float64(s0)*(1-frac) + float64(s1)*frac
			sample := int16(math.Round(val))

			outOff := i*frameSize + ch*bytesPerSample
			binary.LittleEndian.PutUint16(out[outOff:outOff+2], uint16(sample))
		}
	}

	return out
}
