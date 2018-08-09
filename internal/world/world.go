package world

/*
#cgo LDFLAGS: -L../../cmodules/world/build -lworld -lstdc++
#cgo CFLAGS: -I../../cmodules/world/src
#include "world/harvest.h"
#include "world/stonemask.h"
*/
import "C"
import (
	"math"
)

const (
	useStoneMask = false
)

func Harvest(x []float64, fs int, framePeriod float64) []float64 {
	n := len(x)
	m := n / int(math.Floor(float64(fs)*framePeriod))
	tmppos := make([]float64, m)
	f0 := make([]float64, m)
	opts := &C.struct___0{
		f0_floor:     C.double(71.0),
		f0_ceil:      C.double(800.0),
		frame_period: C.double(framePeriod * 1000.0),
	}
	C.Harvest(
		(*C.double)(&x[0]),
		C.int(n),
		C.int(fs),
		opts,
		(*C.double)(&tmppos[0]),
		(*C.double)(&f0[0]),
	)
	if !useStoneMask {
		return f0
	}
	f0r := make([]float64, m)
	C.StoneMask(
		(*C.double)(&x[0]),
		C.int(n),
		C.int(fs),
		(*C.double)(&tmppos[0]),
		(*C.double)(&f0[0]),
		C.int(m),
		(*C.double)(&f0r[0]),
	)
	return f0r
}
