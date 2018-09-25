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
	useStoneMask  = false
	useCheapTrick = false
)

type cDoublePtr = *C.double

const (
	kLog2 = 0.69314718055994529
)

func MyMaxInt(a, b int) int {
	if a < b {
		return b
	}
	return a
}

func MyMinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func GetSuitableFFTSize(sample int) int {
	return int(math.Pow(2.0, math.Floor(math.Log(float64(sample)) / kLog2) + 1.0))
}

// DioGeneralBody estimates the F0 based on Distributed Inline-filter Operation.
func DioGeneralBody(
	x []float64, x_length int, fs int,
	frame_period float64, f0_floor float64, f0_ceil float64,
	channels_in_octave float64, speed int, allowed_range float64,
	temporal_positions []float64, f0 []float64,
) {
	number_of_bands := 1 + int(math.Log(f0_ceil/f0_floor)/kLog2*channels_in_octave)
	boundary_f0_list := make([]float64, number_of_bands)
	for i := 0; i < number_of_bands; i++ {
		boundary_f0_list[i] = f0_floor * math.Pow(2.0, float64(i+1)/channels_in_octave)
	}

	// normalization
	decimation_ratio := MyMaxInt(MyMinInt(speed, 12), 1)
	y_length := 1 + int(x_length/decimation_ratio)
	actual_fs := float64(fs) / float64(decimation_ratio)
	fft_size := int(C.CallGetSuitableFFTSize(
		C.int(y_length + (4 * int(1.0+actual_fs/boundary_f0_list[0]/2.0))),
	))

	// Calculation of the spectrum used for the f0 estimation
	y_spectrum := make([]C.fft_complex, fft_size)
	C.CallGetSpectrumForEstimation(
		(*C.double)(&x[0]),
		C.int(x_length),
		C.int(y_length),
		C.double(actual_fs),
		C.int(fft_size),
		C.int(decimation_ratio),
		(*C.fft_complex)(&y_spectrum[0]),
	)

	f0_length := int(C.CallGetSamplesForDIO(
		C.int(fs),
		C.int(x_length),
		C.double(frame_period),
	))

	f0_candidates_go := make([][]float64, number_of_bands)
	f0_scores_go := make([][]float64, number_of_bands)
	f0_candidates := make([]*C.double, number_of_bands)
	f0_scores := make([]*C.double, number_of_bands)
	for i := 0; i < number_of_bands; i++ {
		f0_candidate := make([]float64, f0_length)
		f0_score := make([]float64, f0_length)
		f0_candidates_go[i] = f0_candidate
		f0_scores_go[i] = f0_score
		f0_candidates[i] = (*C.double)(&f0_candidate[0])
		f0_scores[i] = (*C.double)(&f0_score[0])
	}

	for i := 0; i < f0_length; i++ {
		temporal_positions[i] = float64(i) * frame_period / 1000.0
	}

	C.CallGetF0CandidatesAndScores(
		(*C.double)(&boundary_f0_list[0]),
		C.int(number_of_bands),
		C.double(actual_fs),
		C.int(y_length),
		(*C.double)(&temporal_positions[0]),
		C.int(f0_length),
		(*C.fft_complex)(&y_spectrum[0]),
		C.int(fft_size),
		C.double(f0_floor),
		C.double(f0_ceil),
		(**C.double)(&f0_candidates[0]),
		(**C.double)(&f0_scores[0]),
	)

	// Selection of the best value based on fundamental-ness.
	// This function is related with SortCandidates() in MATLAB.
	best_f0_contour := make([]float64, f0_length)
	C.CallGetBestF0Contour(
		C.int(f0_length),
		(**C.double)(&f0_candidates[0]),
		(**C.double)(&f0_scores[0]),
		C.int(number_of_bands),
		(*C.double)(&best_f0_contour[0]),
	)

	// Postprocessing to find the best f0-contour.
	C.CallFixF0Contour(
		C.double(frame_period),
		C.int(number_of_bands),
		C.int(fs),
		(**C.double)(&f0_candidates[0]),
		(*C.double)(&best_f0_contour[0]),
		C.int(f0_length),
		C.double(f0_floor),
		C.double(allowed_range),
		(*C.double)(&f0[0]),
	)
}

// Dio は、波形 x の基本周波数を framePeriod 秒間隔で推定します。
func Dio(x []float64, fs int, framePeriod float64) ([]float64, [][]float64) {
	n := len(x)
	m := n / int(math.Floor(float64(fs)*framePeriod))
	tmppos := make([]float64, m)
	f0 := make([]float64, m)
	dopts := &C.DioOption{}
	C.InitializeDioOption(dopts)
	dopts.frame_period = C.double(framePeriod * 1000.0)
	C.CallDioGeneralBody(
		cDoublePtr(&x[0]),
		C.int(n),
		C.int(fs),
		C.double(dopts.frame_period),
		C.double(dopts.f0_floor),
		C.double(dopts.f0_ceil),
		C.double(dopts.channels_in_octave),
		C.int(dopts.speed),
		C.double(dopts.allowed_range),
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
