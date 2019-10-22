package window

import "math"

// Series は、窓関数から生成される数列です。
type Series []float64

// Apply は、指定のスライスに窓がけを行います。
func (s Series) Apply(dst []float64) {
	for i, v := range dst {
		dst[i] = v * s[i]
	}
}

// New は、指定した窓関数を用いて 0≦t≦1 を n 等分する数列を生成します。
func New(n int, fn func(t float64) float64) Series {
	s := make(Series, n)
	m := float64(n) - 1
	for i := range s {
		t := float64(i) / m
		s[i] = fn(t)
	}
	return s
}

// Hann は、Hann窓を生成します。
func Hann(n int) Series {
	return New(n, func(t float64) float64 {
		a := 2 * math.Pi * t
		return .5 * (1 - math.Cos(a))
	})
}

// Hamming は、Hamming窓を生成します。
func Hamming(n int) Series {
	return New(n, func(t float64) float64 {
		a := 2 * math.Pi * t
		return .54 - .46*math.Cos(a)
	})
}

// BlackmanHarris は、BlackmanHarris窓を生成します。
func BlackmanHarris(n int) Series {
	return New(n, func(t float64) float64 {
		a := 2 * math.Pi * t
		return .35875 - .48829*math.Cos(a) + .14128*math.Cos(2*a) - .01168*math.Cos(3*a)
	})
}

// Nuttall は、Nuttall窓を生成します。
func Nuttall(n int) Series {
	return New(n, func(t float64) float64 {
		a := 2 * math.Pi * t
		return .355768 - .487396*math.Cos(a) + .144232*math.Cos(2*a) - .012604*math.Cos(3*a)
	})
}

// FlatTop は、FlatTop窓を生成します。
func FlatTop(n int) Series {
	return New(n, func(t float64) float64 {
		a := 2 * math.Pi * t
		return 1 - 1.93*math.Cos(a) + 1.29*math.Cos(2*a) - .388*math.Cos(3*a) + .028*math.Cos(4*a)
	})
}

// Bartlett は、Bartlett窓を生成します。
func Bartlett(n int) Series {
	return New(n, func(t float64) float64 {
		return 1 - math.Abs(t*2-1)
	})
}

// Gauss は、Gauss窓を生成します。
func Gauss(n int, sigma float64) Series {
	return New(n, func(t float64) float64 {
		u := (t*2 - 1) / sigma
		return math.Pow(math.E, -.5*u*u)
	})
}
