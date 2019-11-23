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
	input       <-chan float64
	output      chan buffer.Shape
	f0Input     <-chan []float64
	fs          float64
	framePeriod float64
}

func newF0Splitter(f0 <-chan []float64, fs, framePeriod float64) *f0Splitter {
	return &f0Splitter{
		f0Input:     f0,
		fs:          fs,
		framePeriod: framePeriod,
		output:      make(chan buffer.Shape, 4096),
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
		msg := 0
		jPrev := -1
		var f0Buffer []float64
		for v := range s.input {
			i := len(buf)
			buf = append(buf, v)
			j := int(math.Floor(t / float64(s.framePeriod)))
			freq := lastFreq
			if jPrev < j {
				jPrev = j
				if len(f0Buffer) == 0 {
					f0Buffer = <-s.f0Input
				}
				f0 := .0
				if 0 < len(f0Buffer) {
					f0 = f0Buffer[0]
					f0Buffer = f0Buffer[1:]
				}
				if minFreq <= f0 {
					freq = f0
				}
			}
			phase += freq * dt
			if 1.0 <= phase {
				for 1.0 <= phase {
					phase -= 1.0
				}
				s.output <- buffer.MakeShapeTrimmed(buf, iBegin, i)
				msg++
				iBegin = i
			}
			lastFreq = freq
			t += dt
		}
		log.Printf("debug: f0Splitter %d messages", msg)
		close(s.output)
	}()
}
