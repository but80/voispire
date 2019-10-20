package formant

import (
	"math"
	"math/cmplx"

	"github.com/but80/voispire/internal/fft"
	"gonum.org/v1/gonum/fourier"
)

type cepstralShifter struct {
	fft.FFTProcessor
	cfft       *fourier.FFT
	width      int
	fs         int
	shiftInv   float64
	maxPeakNum int
	envelope   []float64
	envelopeDb []complex128
	specDb     []complex128
	ceps       []float64
	spec1      []complex128
}

func NewCepstralShifter(src []float64, fs, width int, shift float64) FormantShifter {
	s := &cepstralShifter{
		cfft:       fourier.NewFFT(width),
		width:      width,
		fs:         fs,
		envelope:   make([]float64, width/2+1),
		envelopeDb: make([]complex128, width/2+1),
		specDb:     make([]complex128, width/2+1),
		ceps:       make([]float64, width),
		spec1:      make([]complex128, width/2+1),
	}
	s.FFTProcessor = fft.NewFFTProcessor(src, width, func(spec0 []complex128, wave0 []float64) []complex128 {
		if len(spec0) <= 4 {
			return spec0
		}
		if len(spec0) != len(s.specDb) {
			panic("wrong length")
		}

		n := 96

		for i, v := range spec0 {
			s.specDb[i] = complex(math.Log(cmplx.Abs(v)), 0)
		}
		s.cfft.Sequence(s.ceps, s.specDb)
		for i := n; i < len(s.ceps)-n; i++ {
			s.ceps[i] = 0
		}
		s.cfft.Coefficients(s.envelopeDb, s.ceps)
		r := 1.0 / float64(s.width)
		for i := range s.envelope {
			v := s.envelopeDb[i]
			s.envelope[i] = math.Pow(math.E, real(v)*r)
		}
		flattenLowerCoefs(s.envelope, s.fs)

		applyEnvelopeShift(s.spec1, spec0, s.envelope, shift)
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
