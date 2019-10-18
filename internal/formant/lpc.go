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

const f0Floor = 70

func NewLPCShifter(src []float64, fs, width int, shift float64) FormantShifter {
	s := &lpcShifter{
		width:       width,
		fs:          fs,
		envelope:    make([]float64, width),
		envDetector: ldurbin.NewSpectralEnvelopeDetector(width, 96),
	}
	s.FFTProcessor = fft.NewFFTProcessor(src, width, func(spec []complex128, wave []float64) []complex128 {
		if len(spec) <= 4 {
			return spec
		}
		n := len(spec)
		env := s.envDetector.Detect(wave)
		if n != len(env) {
			panic(xerrors.Errorf("Spectrum size mismatch (%d != %d)", n != len(env)))
		}
		s.envelope = env

		df := (float64(s.fs) / 2) / float64(n)
		i0 := int(f0Floor / df)
		// 1st max
		for ; i0 < n; i0++ {
			if env[i0] < env[i0-1] {
				break
			}
		}
		// 1st min
		for ; i0 < n; i0++ {
			if env[i0-1] < env[i0] {
				break
			}
		}
		// 2nd max
		for ; i0 < n; i0++ {
			if env[i0] < env[i0-1] {
				break
			}
		}
		// flatten envelope at lower frequency
		for j := i0 - 1; 0 <= j; j-- {
			env[j] = env[i0]
		}

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
