package voispire

import (
	"log"

	"github.com/but80/voispire/internal/buffer"
)

// stretch は、指定したピッチ係数 pitchCoef、速度係数 speedCoef で再生した波形を返します。
// pitchCoef, speedCoef, resampleCoef がすべて 1 のとき、オリジナルと同じ波形となります。
func stretch(input chan buffer.Shape, pitchCoef, speedCoef, resampleCoef float64) chan buffer.Shape {
	history := &buffer.ShapeHistory{}
	out := make(chan buffer.Shape, 16)
	go func() {
		log.Print("debug: stretch goroutine is started")
		srcPhase := .0
		dstPhase := .0
		for s := range input {
			history.Rotate(s)
			freq := history.Freq()
			srcPhaseStep := freq * pitchCoef / resampleCoef
			dstPhaseStep := freq * speedCoef / resampleCoef
			buf := []float64{}
			for ; dstPhase < 1.0; dstPhase += dstPhaseStep {
				buf = append(buf, history.Get(srcPhase, dstPhase))
				srcPhase += srcPhaseStep
				for 1.0 <= srcPhase {
					srcPhase -= 1.0
				}
			}
			out <- buffer.MakeShape(buf)
			for 1.0 <= dstPhase {
				dstPhase -= 1.0
			}
		}
		close(out)
	}()
	return out
}
