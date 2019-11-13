package common

import (
	"math"
)

// MaxInt is max() function for "int" type.
func MaxInt(x, y int) int {
	if x < y {
		return y
	}
	return x
}

// MinInt is min() function for "int" type.
func MinInt(x, y int) int {
	if x < y {
		return x
	}
	return y
}

// GetSuitableFFTSize calculates the suitable FFT size.
// The size is defined as the minimum length whose length is longer than
// the input sample.
func GetSuitableFFTSize(sample int) int {
	e := uint(math.Log2(float64(sample)))
	return 1 << (e + 1)
}

// NuttallWindow calculates the coefficients of Nuttall window whose length
// is y_length and is used in Dio, Harvest and D4C.
func NuttallWindow(y []float64) {
	for i := range y {
		tmp := float64(i) / float64(len(y)-1)
		y[i] = 0.355768 - 0.487396*math.Cos(2.0*math.Pi*tmp) +
			0.144232*math.Cos(4.0*math.Pi*tmp) -
			0.012604*math.Cos(6.0*math.Pi*tmp)
	}
}
