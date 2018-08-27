package voispire

import (
	"log"
	"math"

	"github.com/but80/voispire/internal/buffer"
)

const (
	minFreq = 1.0
)

type f0Splitter struct {
	input  chan float64
	output chan buffer.Shape
	f0     []float64
	fs     float64
}

func newF0Splitter(f0 []float64, fs float64) *f0Splitter {
	return &f0Splitter{
		f0:     f0,
		fs:     fs,
		output: make(chan buffer.Shape, 4096),
	}
}

func (s *f0Splitter) Start() {
	go func() {
		log.Print("debug: f0Splitter goroutine is started")
		t := .0
		dt := 1.0 / s.fs
		iBegin := 0
		phase := .0
		lastFreq := 440.0
		buf := []float64{}
		for v := range s.input {
			i := len(buf)
			buf = append(buf, v)
			j := int(math.Floor(t / float64(framePeriod)))
			freq := lastFreq
			if j < len(s.f0) && minFreq <= s.f0[j] {
				freq = s.f0[j]
			}
			phase += freq * dt
			if 1.0 <= phase {
				for 1.0 <= phase {
					phase -= 1.0
				}
				s.output <- buffer.MakeShapeTrimmed(buf, iBegin, i)
				iBegin = i
			}
			lastFreq = freq
			t += dt
		}
		close(s.output)
	}()
}
