package fft

import (
	"log"
	"math"

	"github.com/mjibson/go-dsp/window"
	"gonum.org/v1/gonum/fourier"
)

type FFTProcessor interface {
	Output() <-chan float64
	OnFinish(func())
	Start()
}

type fftProcessor struct {
	fft       *fourier.FFT
	output    chan float64
	src       []float64
	width     int
	processor func([]complex128, []float64) []complex128
	onFinish  func()
}

func NewFFTProcessor(src []float64, width int, processor func([]complex128, []float64) []complex128) FFTProcessor {
	if width < 4 {
		width = 4
	}
	width = (width >> 1) << 1
	return &fftProcessor{
		fft:       fourier.NewFFT(width),
		src:       src,
		width:     width,
		processor: processor,
		output:    make(chan float64, 4096),
	}
}

func (s *fftProcessor) Output() <-chan float64 {
	return s.output
}

func (s *fftProcessor) OnFinish(callback func()) {
	s.onFinish = callback
}

func sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	if 0 < v {
		return 1
	}
	return 0
}

func signedSqrt(v float64) float64 {
	return sign(v) * math.Sqrt(math.Abs(v))
}

func easing(width int) []float64 {
	result := make([]float64, width)
	n := width / 2
	for i := 0; i < n; i++ {
		t := float64(i) / float64(n)
		c := math.Cos(t * math.Pi)
		v := (1 + signedSqrt(c)) * .5
		result[n-1-i] = v
		result[n+i] = v
	}
	return result
}

func (s *fftProcessor) Start() {
	go func() {
		log.Print("debug: fftProcessor goroutine is started")
		step := s.width >> 1
		hann := window.Hann(s.width)
		wave0 := make([]float64, s.width)
		spec0 := make([]complex128, s.width/2+1)
		resultPrev := make([]float64, s.width)
		wave1 := make([]float64, s.width)

		merge := easing(s.width)
		for i, w := range window.Hamming(s.width) {
			merge[i] /= w
		}

		n0 := len(s.src)
		n := n0
		if n%step != 0 {
			n = (n/step + 1) * step
		}
		n += step
		for len(s.src) < n {
			s.src = append(s.src, 0)
		}
		for i := 0; i < n0; i += step {
			// log.Printf("debug: fftProcessor %d", i)
			for j, w := range hann {
				wave0[j] = s.src[i+j] * w
			}
			s.fft.Coefficients(spec0, wave0)
			r := complex(1/float64(s.fft.Len()), 0)
			for i, v := range spec0 {
				spec0[i] = v * r
			}
			spec1 := s.processor(spec0, wave0)
			s.fft.Sequence(wave1, spec1)
			for i, w := range merge {
				wave1[i] *= w
			}
			prev := resultPrev[step:]
			for i := 0; i < step; i++ {
				s.output <- prev[i] + wave1[i]
			}
			wave1, resultPrev = resultPrev, wave1
		}
		if s.onFinish != nil {
			s.onFinish()
		}
		close(s.output)
	}()
}
