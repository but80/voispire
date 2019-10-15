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
	fs          int
	shiftInv    float64
	maxPeakNum  int
	envBuf      []float64
	envDetector *ldurbin.SpectralEnvelopeDetector
}

var onFormantFFTProcess func(*formantShifter, []float64, []float64, []complex128, []complex128)
var onFormantFFTFinish func(*formantShifter)

func newFormantShifter(src []float64, fs, width int, shift float64) *formantShifter {
	shiftInv := 1.0 / shift
	s := &formantShifter{
		width:       width,
		fs:          fs,
		envBuf:      make([]float64, width),
		envDetector: ldurbin.NewSpectralEnvelopeDetector(width, 64),
	}
	s.fftProcessor = newFFTProcessor(src, width, func(spec []complex128, wave []float64) []complex128 {
		if len(spec) <= 4 {
			return spec
		}
		n := len(spec)
		env := s.envDetector.Detect(wave)
		s.envBuf = env
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
	s.fftProcessor.OnProcess = func(wave0, wave1 []float64, spec0, spec1 []complex128) {
		if onFormantFFTProcess != nil {
			onFormantFFTProcess(s, wave0, wave1, spec0, spec1)
		}
	}
	s.fftProcessor.OnFinish = func() {
		if onFormantFFTFinish != nil {
			onFormantFFTFinish(s)
		}
	}
	return s
}
