package voispire

import (
	"log"

	"github.com/mjibson/go-dsp/fft"
	"github.com/mjibson/go-dsp/window"
)

type fftProcessor struct {
	output    chan float64
	src       []float64
	width     int
	processor func([]complex128) []complex128
}

func newFFTProcessor(src []float64, width int, processor func([]complex128) []complex128) *fftProcessor {
	if width < 4 {
		width = 4
	}
	return &fftProcessor{
		src:       src,
		width:     (width >> 1) << 1,
		processor: processor,
		output:    make(chan float64, 4096),
	}
}

func (s *fftProcessor) Start() {
	go func() {
		log.Print("debug: fftProcessor goroutine is started")
		step := s.width >> 1
		buffer := make([]complex128, step)
		for i := 0; i < len(s.src); i += step {
			// log.Printf("debug: fftProcessor %d", i)
			j := i + s.width
			var src []float64
			if j <= len(s.src) {
				src = s.src[i:j]
			} else {
				src = s.src[i:]
				for len(src) < s.width {
					src = append(src, .0)
				}
			}
			wave := make([]complex128, s.width)
			win := window.Hann(s.width)
			for i, w := range win {
				wave[i] = complex(src[i]*w, .0)
			}
			spec := fft.FFT(wave)
			spec2 := s.processor(spec)
			result := fft.IFFT(spec2)
			for i := 0; i < step; i++ {
				s.output <- real(buffer[i]) + real(result[i])
			}
			buffer = result[step:]
		}
		close(s.output)
	}()
}
