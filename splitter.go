package voispire

import (
	"log"
	"math"

	"github.com/but80/voispire/internal/buffer"
)

const (
	minFreq = 1.0
)

type splitter struct {
	output chan buffer.Shape
	src    []float64
	f0     []float64
	fs     float64
}

func newSplitter(src, f0 []float64, fs float64) *splitter {
	return &splitter{
		src:    src,
		f0:     f0,
		fs:     fs,
		output: make(chan buffer.Shape, 4096),
	}
}

func (s *splitter) Output() chan buffer.Shape {
	return s.output
}

func (s *splitter) Start() {
	go func() {
		log.Print("debug: splitter goroutine is started")
		t := .0
		dt := 1.0 / s.fs
		iBegin := 0
		phase := .0
		lastFreq := 440.0
		for i := range s.src {
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
				s.output <- buffer.MakeShapeTrimmed(s.src, iBegin, i)
				iBegin = i
			}
			lastFreq = freq
			t += dt
		}
		close(s.output)
	}()
}
