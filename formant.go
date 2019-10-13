package voispire

import (
	"github.com/but80/voispire/internal/ldurbin"
	"golang.org/x/xerrors"
)

// https://synsinger.wordpress.com/2015/11/21/pitch-shifting-using-a-spectral-envelope/

type peak struct {
	index int
	level float64
}

type formantShifter struct {
	*fftProcessor
	width       int
	shiftInv    float64
	maxPeakNum  int
	ampBuf      []float64
	envBuf      []float64
	peaksBuf    []peak
	envDetector *ldurbin.SpectralEnvelopeDetector
}

func newFormantShifter(src []float64, width int, shift float64) *formantShifter {
	shiftInv := 1.0 / shift
	maxPeakNum := 100
	s := &formantShifter{
		width:       width,
		ampBuf:      make([]float64, width),
		envBuf:      make([]float64, width),
		peaksBuf:    make([]peak, 0, maxPeakNum),
		envDetector: ldurbin.NewSpectralEnvelopeDetector(width, 128),
	}
	s.fftProcessor = newFFTProcessor(src, width, func(spec []complex128, wave []float64) []complex128 {
		if len(spec) <= 4 {
			return spec
		}
		n := len(spec)
		env := s.envDetector.Detect(wave)
		if len(spec) != len(env) {
			panic(xerrors.Errorf("Spectrum size mismatch (%d != %d)", len(spec) != len(env)))
		}
		for i := 0; i < n; i++ {
			j := int(float64(i)*shiftInv + .5)
			if n <= j {
				j = n - 1
			}
			spec[i] *= complex(env[j]/env[i], .0)
		}
		return spec
	})
	return s
}
