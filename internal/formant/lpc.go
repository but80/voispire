package formant

import (
	"github.com/but80/voispire/internal/fft"
	"github.com/but80/voispire/internal/levinsondurbin"
	"golang.org/x/xerrors"
)

type lpcShifter struct {
	fft.FFTProcessor
	width       int
	fs          int
	shiftInv    float64
	maxPeakNum  int
	envDetector *levinsondurbin.SpectralEnvelopeDetector
	envelope    []float64
	spec1       []complex128
}

// NewLPCShifter は、線形予測分析を用いたフォルマントシフタを作成します。
func NewLPCShifter(src []float64, fs, width int, shift float64) FormantShifter {
	s := &lpcShifter{
		width:       width,
		fs:          fs,
		envDetector: levinsondurbin.NewSpectralEnvelopeDetector(width, 96),
		envelope:    make([]float64, width/2+1),
		spec1:       make([]complex128, width/2+1),
	}
	s.FFTProcessor = fft.NewFFTProcessor(src, width, func(spec0 []complex128, wave0 []float64) []complex128 {
		if len(spec0) <= 4 {
			return spec0
		}
		n := len(spec0)
		env := s.envDetector.Detect(wave0)
		if n != len(env) {
			panic(xerrors.Errorf("Spectrum size mismatch (%d != %d)", n, len(env)))
		}
		flattenLowerCoefs(env, fs)
		s.envelope = env

		thr := int(float64(n) / shift)
		for i := thr; i < n; i++ {
			env[i] = env[thr]
		}

		applyEnvelopeShift(s.spec1, spec0, env, shift)
		analyzerFrame(&analyzerData{
			fs:       fs,
			fftWidth: width,
			wave0:    wave0,
			envelope: s.envelope,
			spec0:    spec0,
			spec1:    s.spec1,
		})
		return s.spec1
	})
	s.FFTProcessor.OnFinish(analyzerFinish)
	return s
}
