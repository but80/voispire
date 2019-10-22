package window

import "math"

// Hann は、Hann窓を生成します。
func Hann(n int) []float64 {
	m := float64(n) - 1
	result := make([]float64, n)
	for i := range result {
		t := float64(i) / m
		a := 2 * math.Pi * t
		result[i] = .5 * (1 - math.Cos(a))
	}
	return result
}

// Hamming は、Hamming窓を生成します。
func Hamming(n int) []float64 {
	m := float64(n) - 1
	result := make([]float64, n)
	for i := range result {
		t := float64(i) / m
		a := 2 * math.Pi * t
		result[i] = .54 - .46*math.Cos(a)
	}
	return result
}

// BlackmanHarris は、BlackmanHarris窓を生成します。
func BlackmanHarris(n int) []float64 {
	m := float64(n) - 1
	result := make([]float64, n)
	for i := range result {
		t := float64(i) / m
		a := 2 * math.Pi * t
		result[i] = .35875 - .48829*math.Cos(a) + .14128*math.Cos(2*a) - .01168*math.Cos(3*a)
	}
	return result
}

// Nuttall は、Nuttall窓を生成します。
func Nuttall(n int) []float64 {
	m := float64(n) - 1
	result := make([]float64, n)
	for i := range result {
		t := float64(i) / m
		a := 2 * math.Pi * t
		result[i] = .355768 - .487396*math.Cos(a) + .144232*math.Cos(2*a) - .012604*math.Cos(3*a)
	}
	return result
}

// FlatTop は、FlatTop窓を生成します。
func FlatTop(n int) []float64 {
	m := float64(n) - 1
	result := make([]float64, n)
	for i := range result {
		t := float64(i) / m
		a := 2 * math.Pi * t
		result[i] = 1.0 - 1.93*math.Cos(a) + 1.29*math.Cos(2*a) - .388*math.Cos(3*a) + .028*math.Cos(4*a)
	}
	return result
}

// Bartlett は、Bartlett窓を生成します。
func Bartlett(n int) []float64 {
	h := float64(n) / 2
	c := float64(n-1) / 2
	result := make([]float64, n)
	for i := range result {
		d := float64(i) - c
		result[i] = (h - math.Abs(d)) / h
	}
	return result
}

// Gauss は、Gauss窓を生成します。
func Gauss(n int, sigma float64) []float64 {
	c := float64(n-1) / 2
	result := make([]float64, n)
	for i := range result {
		d := float64(i) - c
		t := d / c / sigma
		result[i] = math.Pow(math.E, -.5*t*t)
	}
	return result
}
