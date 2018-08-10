package world

/*
#cgo LDFLAGS: -L../../cmodules/world/build -lworld -lstdc++
#cgo CFLAGS: -I../../cmodules/world/src
#include "world/dio.h"
#include "world/stonemask.h"
#include "world/cheaptrick.h"
#include <stdlib.h>

static double** _alloc_doubleptr_array(int size) {
	return (double**)malloc(sizeof(double*) * size);
}

static void _free_doubleptr_array(double** p) {
	free(p);
}

static void _write_doubleptr_array(double** p, int index, double* v) {
	p[index] = v;
}

*/
import "C"
import (
	"math"
)

const (
	useStoneMask = false
	useCheapTrick = false
)

type cDoublePtr = *C.double

// Harvest は、波形 x の基本周波数を framePeriod 秒間隔で推定します。
func Harvest(x []float64, fs int, framePeriod float64) ([]float64, [][]float64) {
	n := len(x)
	m := n / int(math.Floor(float64(fs)*framePeriod))
	tmppos := make([]float64, m)
	f0 := make([]float64, m)
	dopts := &C.DioOption{}
	C.InitializeDioOption(dopts)
	dopts.frame_period = C.double(framePeriod * 1000.0)
	C.Dio(
		cDoublePtr(&x[0]),
		C.int(n),
		C.int(fs),
		dopts,
		cDoublePtr(&tmppos[0]),
		cDoublePtr(&f0[0]),
	)

	if useStoneMask {
		f0r := make([]float64, m)
		C.StoneMask(
			cDoublePtr(&x[0]),
			C.int(n),
			C.int(fs),
			cDoublePtr(&tmppos[0]),
			cDoublePtr(&f0[0]),
			C.int(m),
			cDoublePtr(&f0r[0]),
		)
		f0 = f0r
	}

	copts := &C.struct___0{}
	C.InitializeCheapTrickOption(C.int(fs), copts)

	var spectro [][]float64
	if useCheapTrick {
		spectro = make([][]float64, m)
		cspectro := C._alloc_doubleptr_array(C.int(m))
		for i := range spectro {
			s := make([]float64, int(copts.fft_size))
			spectro[i] = s
			C._write_doubleptr_array(cspectro, C.int(i), cDoublePtr(&s[0]))
		}
		C.CheapTrick(
			cDoublePtr(&x[0]),
			C.int(n),
			C.int(fs),
			cDoublePtr(&tmppos[0]),
			cDoublePtr(&f0[0]),
			C.int(m),
			copts,
			cspectro,
		)
		C._free_doubleptr_array(cspectro)
	}

	return f0, spectro
}
