package formant

import (
	"github.com/but80/voispire/internal/fft"
)

type FormantShifter interface {
	fft.FFTProcessor
	Fs() int
	LastEnvelope() []float64
}

var onFFTProcess func(interface{}, []float64, []float64, []complex128, []complex128)
var onFFTFinish func(interface{})

func lerp(a, b, t float64) float64 {
	return a*(1-t) + b*t
}

func applyEnvelopeShift(spec []complex128, env []float64, shift float64) {
	n := len(spec)
	for i := 1; i < n; i++ {
		j := float64(i) / shift
		if j<1 {
			j=1
		}
		ji := int(j)
		jf := j - float64(ji)
		if n-2 < ji {
			ji = n - 2
			jf = 1
		}
		e := lerp(env[ji], env[ji+1], jf)
		spec[i] *= complex(e/env[i], .0)
	}
}
