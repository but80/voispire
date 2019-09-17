package world

/*
#cgo LDFLAGS: -L../../cmodules/world/build -lworld -lstdc++ -lm
#cgo CFLAGS: -I../../cmodules/world/src
#include "world/dio.h"
#include "world/stonemask.h"
#include "world/cheaptrick.h"
#include "world/matlabfunctions.h"
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

static void _write_doubleptr_array_elem(double** p, int i, int j, double v) {
	p[i][j] = v;
}

static double** _alloc_double_array_array(int n, int m) {
	double** result = _alloc_doubleptr_array(n);
	for (int i=0; i<n; i++) {
		result[i] = (double*)malloc(sizeof(double) * m);
	}
	return result;
}

static void _free_double_array_array(double** p, int n) {
	for (int i=0; i<n; i++) {
		free(p[i]);
	}
	free(p);
}

*/
import "C"
import (
	"math"
	"unsafe"
)

const (
	useStoneMask  = false
	useCheapTrick = false
)

func toDoublePtr(a []float64) *C.double {
	return (*C.double)(&a[0])
}

func toFFTComplexPtr(a []complex128) *C.fft_complex {
	return (*C.fft_complex)(unsafe.Pointer(&a[0]))
}

func GetSamplesForDIO(fs int, x_length int, frame_period float64) int {
	return int(1000.0*float64(x_length)/float64(fs)/frame_period) + 1
}

func GetF0CandidatesAndScores(
	boundary_f0_list []float64,
	number_of_bands int, actual_fs float64, y_length int,
	temporal_positions []float64, f0_length int,
	y_spectrum []complex128, fft_size int, f0_floor, f0_ceil float64,
	raw_f0_candidates, raw_f0_scores **C.double,
) {
	f0_candidate := make([]float64, f0_length)
	f0_score := make([]float64, f0_length)

	// Calculation of the acoustics events (zero-crossing)
	for i := 0; i < number_of_bands; i++ {
		C.CallGetF0CandidateFromRawEvent(
			C.double(boundary_f0_list[i]),
			C.double(actual_fs),
			(*[2]C.double)(toFFTComplexPtr(y_spectrum)),
			C.int(y_length),
			C.int(fft_size),
			C.double(f0_floor),
			C.double(f0_ceil),
			toDoublePtr(temporal_positions),
			C.int(f0_length),
			toDoublePtr(f0_score),
			toDoublePtr(f0_candidate),
		)
		for j := 0; j < f0_length; j++ {
			// A way to avoid zero division
			C._write_doubleptr_array_elem(raw_f0_scores, C.int(i), C.int(j), C.double(f0_score[j]/(f0_candidate[j]+kMySafeGuardMinimum)))
			C._write_doubleptr_array_elem(raw_f0_candidates, C.int(i), C.int(j), C.double(f0_candidate[j]))
		}
	}
}

func DesignLowCutFilter(N, fft_size int, low_cut_filter []float64) {
	for i := 1; i <= N; i++ {
		low_cut_filter[i-1] = 0.5 - 0.5*math.Cos(float64(i)*2.0*math.Pi/float64(N+1))
	}
	for i := N; i < fft_size; i++ {
		low_cut_filter[i] = 0.0
	}
	sum_of_amplitude := 0.0
	for i := 0; i < N; i++ {
		sum_of_amplitude += low_cut_filter[i]
	}
	for i := 0; i < N; i++ {
		low_cut_filter[i] = -low_cut_filter[i] / sum_of_amplitude
	}
	for i := 0; i < (N-1)/2; i++ {
		low_cut_filter[fft_size-(N-1)/2+i] = low_cut_filter[i]
	}
	for i := 0; i < N; i++ {
		low_cut_filter[i] = low_cut_filter[i+(N-1)/2]
	}
	low_cut_filter[0] += 1.0
}

func GetSpectrumForEstimation(
	x []float64, x_length int, y_length int, actual_fs float64,
	fft_size int, decimation_ratio int, y_spectrum []complex128,
) {
	y := make([]float64, fft_size)

	// Downsampling
	if decimation_ratio != 1 {
		decimate(x, x_length, decimation_ratio, y)
	} else {
		copy(y, x)
	}

	// Removal of the DC component (y = y - mean value of y)
	mean_y := 0.0
	for i := 0; i < y_length; i++ {
		mean_y += y[i]
	}
	mean_y /= float64(y_length)
	for i := 0; i < y_length; i++ {
		y[i] -= mean_y
	}
	for i := y_length; i < fft_size; i++ {
		y[i] = 0.0
	}

	forwardFFT := C.fft_plan_dft_r2c_1d(
		C.int(fft_size),
		toDoublePtr(y),
		toFFTComplexPtr(y_spectrum),
		C.FFT_ESTIMATE,
	)
	C.fft_execute(forwardFFT)

	// Low cut filtering (from 0.1.4)
	cutoff_in_sample := int(C.matlab_round(C.double(actual_fs / 50.0))) // Cutoff is 50.0 Hz
	DesignLowCutFilter(cutoff_in_sample*2+1, fft_size, y)

	filter_spectrum := make([]complex128, fft_size)
	forwardFFT.c_out = toFFTComplexPtr(filter_spectrum)
	C.fft_execute(forwardFFT)

	for i := 0; i <= fft_size/2; i++ {
		// Complex number multiplications.
		y_spectrum[i] = complex(
			real(y_spectrum[i])*real(filter_spectrum[i])-imag(y_spectrum[i])*imag(filter_spectrum[i]),
			real(y_spectrum[i])*imag(filter_spectrum[i])+imag(y_spectrum[i])*real(filter_spectrum[i]),
		)
	}

	C.fft_destroy_plan(forwardFFT)
	// delete[] y;
	// delete[] filter_spectrum;
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
	fft_size := GetSuitableFFTSize(y_length + (4 * int(1.0+actual_fs/boundary_f0_list[0]/2.0)))

	// Calculation of the spectrum used for the f0 estimation
	y_spectrum := make([]complex128, fft_size)
	GetSpectrumForEstimation(x, x_length, y_length, actual_fs, fft_size, decimation_ratio, y_spectrum)

	f0_length := GetSamplesForDIO(fs, x_length, frame_period)

	f0_candidates := C._alloc_double_array_array(C.int(number_of_bands), C.int(f0_length))
	f0_scores := C._alloc_double_array_array(C.int(number_of_bands), C.int(f0_length))

	for i := 0; i < f0_length; i++ {
		temporal_positions[i] = float64(i) * frame_period / 1000.0
	}

	GetF0CandidatesAndScores(
		boundary_f0_list, number_of_bands, actual_fs, y_length, temporal_positions,
		f0_length, y_spectrum, fft_size, f0_floor, f0_ceil, f0_candidates, f0_scores,
	)

	// Selection of the best value based on fundamental-ness.
	// This function is related with SortCandidates() in MATLAB.
	best_f0_contour := make([]float64, f0_length)
	C.CallGetBestF0Contour(
		C.int(f0_length),
		f0_candidates,
		f0_scores,
		C.int(number_of_bands),
		toDoublePtr(best_f0_contour),
	)

	// Postprocessing to find the best f0-contour.
	C.CallFixF0Contour(
		C.double(frame_period),
		C.int(number_of_bands),
		C.int(fs),
		f0_candidates,
		toDoublePtr(best_f0_contour),
		C.int(f0_length),
		C.double(f0_floor),
		C.double(allowed_range),
		toDoublePtr(f0),
	)

	C._free_double_array_array(f0_candidates, C.int(number_of_bands))
	C._free_double_array_array(f0_scores, C.int(number_of_bands))
}

type DioOption struct {
	f0_floor           float64
	f0_ceil            float64
	channels_in_octave float64
	frame_period       float64 // msec
	speed              int     // (1, 2, ..., 12)
	allowed_range      float64 // Threshold used for fixing the F0 contour.
}

func InitializeDioOption() *DioOption {
	option := &DioOption{}

	// You can change default parameters.
	option.channels_in_octave = 2.0
	option.f0_ceil = kCeilF0
	option.f0_floor = kFloorF0
	option.frame_period = 5

	// You can use the value from 1 to 12.
	// Default value 11 is for the fs of 44.1 kHz.
	// The lower value you use, the better performance you can obtain.
	option.speed = 1

	// You can give a positive real number as the threshold.
	// The most strict value is 0, and there is no upper limit.
	// On the other hand, I think that the value from 0.02 to 0.2 is reasonable.
	option.allowed_range = 0.1

	return option
}

// Dio は、波形 x の基本周波数を framePeriod 秒間隔で推定します。
func Dio(x []float64, fs int, framePeriod float64) ([]float64, [][]float64) {
	n := len(x)
	m := n / int(math.Floor(float64(fs)*framePeriod))
	tmppos := make([]float64, m)
	f0 := make([]float64, m)
	dopts := InitializeDioOption()
	dopts.frame_period = framePeriod * 1000.0
	DioGeneralBody(x, n, fs, dopts.frame_period, dopts.f0_floor, dopts.f0_ceil,
		dopts.channels_in_octave, dopts.speed, dopts.allowed_range, tmppos, f0)

	if useStoneMask {
		f0r := make([]float64, m)
		C.StoneMask(
			toDoublePtr(x),
			C.int(n),
			C.int(fs),
			toDoublePtr(tmppos),
			toDoublePtr(f0),
			C.int(m),
			toDoublePtr(f0r),
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
			C._write_doubleptr_array(cspectro, C.int(i), toDoublePtr(s))
		}
		C.CheapTrick(
			toDoublePtr(x),
			C.int(n),
			C.int(fs),
			toDoublePtr(tmppos),
			toDoublePtr(f0),
			C.int(m),
			copts,
			cspectro,
		)
		C._free_doubleptr_array(cspectro)
	}

	return f0, spectro
}
