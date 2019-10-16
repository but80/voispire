package voispire

import (
	"log"

	"github.com/mjibson/go-dsp/window"
	"gonum.org/v1/gonum/fourier"
)

type fftProcessor struct {
	fft       *fourier.FFT
	output    chan float64
	src       []float64
	width     int
	processor func([]complex128, []float64) []complex128
	OnProcess func([]float64, []float64, []complex128, []complex128)
	OnFinish  func()
}

func newFFTProcessor(src []float64, width int, processor func([]complex128, []float64) []complex128) *fftProcessor {
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

func (s *fftProcessor) Start() {
	go func() {
		log.Print("debug: fftProcessor goroutine is started")
		step := s.width >> 1
		win := window.Hann(s.width)
		spec := make([]complex128, s.width/2+1)
		resultPrev := make([]float64, s.width)
		result := make([]float64, s.width)
		ampCoef := .5 / float64(s.width)
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
			wave := s.src[i : i+s.width]
			s.fft.Coefficients(spec, wave)
			var spec0 []complex128
			if s.OnProcess != nil {
				spec0 = make([]complex128, len(spec))
				copy(spec0, spec)
			}
			spec = s.processor(spec, wave)
			s.fft.Sequence(result, spec)
			for i, w := range win {
				result[i] *= ampCoef * w
			}
			if s.OnProcess != nil {
				s.OnProcess(wave, result, spec0, spec)
			}
			prev := resultPrev[step:]
			for i := 0; i < step; i++ {
				s.output <- prev[i] + result[i]
			}
			result, resultPrev = resultPrev, result
		}
		if s.OnFinish != nil {
			s.OnFinish()
		}
		close(s.output)
	}()
}
