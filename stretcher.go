package voispire

import (
	"log"

	"github.com/but80/voispire/internal/buffer"
)

// stretcher は、指定したピッチ係数 pitchCoef、速度係数 speedCoef で再生した波形を返します。
// pitchCoef, speedCoef, resampleCoef がすべて 1 のとき、オリジナルと同じ波形となります。
type stretcher struct {
	output       chan buffer.Shape
	input        <-chan buffer.Shape
	pitchCoef    float64
	speedCoef    float64
	resampleCoef float64
	minChunkLen  int
}

func newStretcher(pitchCoef, speedCoef, resampleCoef float64) *stretcher {
	return &stretcher{
		output:       make(chan buffer.Shape, 16),
		pitchCoef:    pitchCoef,
		speedCoef:    speedCoef,
		resampleCoef: resampleCoef,
		minChunkLen:  1024,
	}
}

func (s *stretcher) Start() {
	history := &buffer.ShapeHistory{}
	go func() {
		log.Print("debug: stretcher goroutine is started")
		srcPhase := .0
		dstPhase := .0
		result := []float64{}
		msg := 0
		for shape := range s.input {
			history.Rotate(shape)
			freq := history.Freq()
			srcPhaseStep := freq * s.pitchCoef / s.resampleCoef
			dstPhaseStep := freq * s.speedCoef / s.resampleCoef
			for ; dstPhase < 1.0; dstPhase += dstPhaseStep {
				result = append(result, history.Get(srcPhase, dstPhase))
				srcPhase += srcPhaseStep
				for 1.0 <= srcPhase {
					srcPhase -= 1.0
				}
			}
			if s.minChunkLen <= len(result) {
				s.output <- buffer.MakeShape(result)
				msg++
				result = []float64{}
			}
			for 1.0 <= dstPhase {
				dstPhase -= 1.0
			}
		}
		if 0 < len(result) {
			s.output <- buffer.MakeShape(result)
			msg++
		}
		log.Printf("debug: stretcher %d messages", msg)
		close(s.output)
	}()
}
