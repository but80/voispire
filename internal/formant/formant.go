package formant

import (
	"github.com/but80/voispire/internal/fft"
	"golang.org/x/xerrors"
)

type FormantShifter interface {
	fft.FFTProcessor
	Fs() int
	LastEnvelope() []float64
}

var onFFTProcess func(interface{}, []float64, []float64, []complex128, []complex128)
var onFFTFinish func(interface{})

const f0Floor = 70

func flattenLowerCoefs(env []float64, fs int) {
	n := len(env)
	fn := float64(fs) / 2
	i0 := int(float64(n) * f0Floor / fn)
	if i0 == 0 {
		return
	}
	// 1st max
	for ; i0 < n; i0++ {
		if env[i0] < env[i0-1] {
			break
		}
	}
	// 1st min
	for ; i0 < n; i0++ {
		if env[i0-1] < env[i0] {
			break
		}
	}
	// // 2nd max
	// for ; i0 < n; i0++ {
	// 	if env[i0] < env[i0-1] {
	// 		break
	// 	}
	// }
	if n <= i0 {
		return
	}
	// flatten envelope at lower frequency
	for j := i0 - 1; 0 <= j; j-- {
		env[j] = env[i0]
	}
}

func lerp(a, b, t float64) float64 {
	return a*(1-t) + b*t
}

func applyEnvelopeShift(spec []complex128, env []float64, shift float64) {
	n := len(spec)
	if n != len(env) {
		panic(xerrors.Errorf("Envelope size mismatch (%d != %d)", n, len(env)))
	}
	for i := 1; i < n; i++ {
		j := float64(i) / shift
		if j < 1 {
			j = 1
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
