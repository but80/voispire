package fft

import (
	"log"
	"math"

	"github.com/mjibson/go-dsp/window"
	"gonum.org/v1/gonum/fourier"
)

type FFTProcessor interface {
	Output() <-chan float64
	Width() int
	OnProcess(func(wave0, wave1 []float64, spec0, spec1 []complex128))
	OnFinish(func())
	Start()
}

type fftProcessor struct {
	fft       *fourier.FFT
	output    chan float64
	src       []float64
	width     int
	processor func([]complex128, []float64) []complex128
	onProcess func([]float64, []float64, []complex128, []complex128)
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

func (s *fftProcessor) Width() int {
	return s.width
}

func (s *fftProcessor) OnProcess(callback func(wave0, wave1 []float64, spec0, spec1 []complex128)) {
	s.onProcess = callback
}

func (s *fftProcessor) OnFinish(callback func()) {
	s.onFinish = callback
}

func (s *fftProcessor) Start() {
	go func() {
		log.Print("debug: fftProcessor goroutine is started")
		step := s.width >> 1
		win := window.Hann(s.width)
		for i, w := range win {
			win[i] = math.Sqrt(w)
		}
		wave := make([]float64, s.width)
		spec := make([]complex128, s.width/2+1)
		resultPrev := make([]float64, s.width)
		result := make([]float64, s.width)
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
			for j, w := range win {
				wave[j] = s.src[i+j] * w
			}
			s.fft.Coefficients(spec, wave)
			r := complex(1/float64(s.fft.Len()), 0)
			for i, v := range spec {
				spec[i] = v * r
			}
			var spec0 []complex128
			if s.onProcess != nil {
				spec0 = make([]complex128, len(spec))
				copy(spec0, spec)
			}
			spec = s.processor(spec, wave)
			s.fft.Sequence(result, spec)
			for i, w := range win {
				result[i] *= w
			}
			if s.onProcess != nil {
				s.onProcess(wave, result, spec0, spec)
			}
			prev := resultPrev[step:]
			for i := 0; i < step; i++ {
				s.output <- prev[i] + result[i]
			}
			result, resultPrev = resultPrev, result
		}
		if s.onFinish != nil {
			s.onFinish()
		}
		close(s.output)
	}()
}
