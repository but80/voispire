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
}

func (s *cepstralShifter) Fs() int {
	return s.fs
}

func (s *cepstralShifter) LastEnvelope() []float64 {
	return s.envelope
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
	}
	s.FFTProcessor = fft.NewFFTProcessor(src, width, func(spec []complex128, wave []float64) []complex128 {
		if len(spec) <= 4 {
			return spec
		}
		if len(spec) != len(s.specDb) {
			panic("wrong length")
		}

		n := 128

		for i, v := range spec {
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

		applyEnvelopeShift(spec, s.envelope, shift)
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
