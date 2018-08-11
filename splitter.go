package voispire

import (
	"log"
	"math"
)

const (
	minFreq = 1.0
)

func splitShapes(src, f0 []float64, fs float64) chan shape {
	out := make(chan shape, 4096)
	go func() {
		log.Print("debug: splitShapes goroutine is started")
		t := .0
		dt := 1.0 / fs
		iBegin := 0
		phase := .0
		lastFreq := 440.0
		for i := range src {
			j := int(math.Floor(t / float64(framePeriod)))
			freq := lastFreq
			if j < len(f0) && minFreq <= f0[j] {
				freq = f0[j]
			}
			phase += freq * dt
			if 1.0 <= phase {
				for 1.0 <= phase {
					phase -= 1.0
				}
				out <- makeShape(src[iBegin:i])
				iBegin = i
			}
			lastFreq = freq
			t += dt
		}
		close(out)
	}()
	return out
}
