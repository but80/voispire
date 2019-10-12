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
	processor func([]complex128) []complex128
}

func newFFTProcessor(src []float64, width int, processor func([]complex128) []complex128) *fftProcessor {
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
		wave := make([]float64, s.width)
		widthInv := 1.0 / float64(s.width)
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
			src := s.src[i : i+s.width]
			for i, w := range win {
				wave[i] = src[i] * w
			}
			s.fft.Coefficients(spec, wave)
			spec2 := s.processor(spec)
			s.fft.Sequence(result, spec2)
			for i, v := range result {
				result[i] = v * widthInv
			}
			prev := resultPrev[step:]
			for i := 0; i < step; i++ {
				s.output <- prev[i] + result[i]
			}
			result, resultPrev = resultPrev, result
		}
		close(s.output)
	}()
}
