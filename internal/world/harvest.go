package world

/*
#cgo LDFLAGS: -L../../cmodules/world/build -lworld -lstdc++ -lm
#cgo CFLAGS: -I../../cmodules/world/src
#include "world/harvest.h"
#include <stdlib.h>

*/
import "C"

func toDoublePtr(a []float64) *C.double {
	return (*C.double)(&a[0])
}

func Harvest(x []float64, fs int, framePeriod, f0Floor, f0Ceil float64) ([]float64, []float64) {
	xLength := len(x)
	var option C.HarvestOption
	option.f0_floor = C.double(f0Floor)
	option.f0_ceil = C.double(f0Ceil)
	option.frame_period = C.double(framePeriod) * 1000.0
	f0Length := C.GetSamplesForHarvest(
		C.int(fs),
		C.int(xLength),
		C.double(option.frame_period),
	)
	f0 := make([]float64, f0Length)
	temporalPositions := make([]float64, f0Length)
	C.Harvest(
		toDoublePtr(x),
		C.int(xLength),
		C.int(fs),
		&option,
		toDoublePtr(temporalPositions),
		toDoublePtr(f0),
	)
	return f0, temporalPositions
}
