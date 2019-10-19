package formant

import (
	"github.com/but80/voispire/internal/fft"
	"github.com/but80/voispire/internal/ldurbin"
	"golang.org/x/xerrors"
)

type lpcShifter struct {
	fft.FFTProcessor
	width       int
	fs          int
	shiftInv    float64
	maxPeakNum  int
	envelope    []float64
	envDetector *ldurbin.SpectralEnvelopeDetector
}

func (s *lpcShifter) Fs() int {
	return s.fs
}

func (s *lpcShifter) LastEnvelope() []float64 {
	return s.envelope
}

func NewLPCShifter(src []float64, fs, width int, shift float64) FormantShifter {
	s := &lpcShifter{
		width:       width,
		fs:          fs,
		envelope:    make([]float64, width/2+1),
		envDetector: ldurbin.NewSpectralEnvelopeDetector(width, 96),
	}
	s.FFTProcessor = fft.NewFFTProcessor(src, width, func(spec []complex128, wave []float64) []complex128 {
		if len(spec) <= 4 {
			return spec
		}
		n := len(spec)
		env := s.envDetector.Detect(wave)
		if n != len(env) {
			panic(xerrors.Errorf("Spectrum size mismatch (%d != %d)", n, len(env)))
		}
		flattenLowerCoefs(env, fs)
		s.envelope = env

		thr := int(float64(n) / shift)
		for i := thr; i < n; i++ {
			env[i] = env[thr]
		}

		applyEnvelopeShift(spec, env, shift)
		return spec
	})
	s.FFTProcessor.OnProcess(func(wave0, wave1 []float64, spec0, spec1 []complex128) {
		if onFFTProcess != nil {
			onFFTProcess(s, wave0, wave1, spec0, spec1)
		}
	})
	s.FFTProcessor.OnFinish(func() {
		if onFFTFinish != nil {
			onFFTFinish(s)
		}
	})
	return s
}
