package voispire

import (
	"log"

	"github.com/but80/voispire/internal/buffer"
)

// stretch は、指定したピッチ係数 pitchCoef、速度係数 speedCoef で再生した波形を返します。
// pitchCoef、speedCoef ともに 1 のとき、オリジナルと同じ波形となります。
func stretch(input chan buffer.Shape, pitchCoef, speedCoef float64) chan float64 {
	history := &buffer.ShapeHistory{}
	out := make(chan float64, 4096)
	go func() {
		log.Print("debug: stretch goroutine is started")
		srcPhase := .0
		dstPhase := .0
		for s := range input {
			history.Rotate(s)
			freq := history.Freq()
			srcPhaseStep := freq * pitchCoef
			dstPhaseStep := freq * speedCoef
			for ; dstPhase < 1.0; dstPhase += dstPhaseStep {
				out <- history.Get(srcPhase, dstPhase)
				srcPhase += srcPhaseStep
				for 1.0 <= srcPhase {
					srcPhase -= 1.0
				}
			}
			for 1.0 <= dstPhase {
				dstPhase -= 1.0
			}
		}
		close(out)
	}()
	return out
}
