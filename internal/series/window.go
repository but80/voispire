package series

import "math"

// Window は、窓関数から生成される数列です。
type Window []float64

// Apply は、スライス src に窓がけを行い、結果を dst に保存します。
// src, dst の長さは、この窓関数の長さと一致する必要があります。
// src, dst に同一のスライスを指定することで、上書きができます。
func (s Window) Apply(dst, src []float64) {
	for i, w := range s {
		dst[i] = src[i] * w
	}
}

// Sum は、この数列の和を返します。
func (s Window) Sum() float64 {
	result := .0
	for _, v := range s {
		result += v
	}
	return result
}

// ApplyAndRemoveOffset は、窓がけを行い、0Hz成分を除去します。
func (s Window) ApplyAndRemoveOffset(dst, src []float64) float64 {
	waveSum := .0
	winSum := .0
	for i, w := range s {
		v := src[i] * w
		dst[i] = v
		waveSum += v
		winSum += w
	}
	offset := waveSum / winSum
	for i, w := range s {
		dst[i] -= offset * w
	}
	return offset
}

// RestoreOffset は、 ApplyAndRemoveOffset で除去した0Hz成分を復元します。
func (s Window) RestoreOffset(dst, src []float64, offset float64) {
	for i, w := range s {
		dst[i] = src[i] + offset*w
	}
}

// New は、指定した窓関数を用いて 0≦t≦1 を n 等分する数列を生成します。
func New(n int, fn func(t float64) float64) Window {
	s := make(Window, n)
	m := float64(n) - 1
	for i := range s {
		t := float64(i) / m
		s[i] = fn(t)
	}
	return s
}

// Rect は、矩形窓を生成します。
func Rect(n int) Window {
	return New(n, func(t float64) float64 {
		return 1
	})
}

// Hann は、Hann窓を生成します。
func Hann(n int) Window {
	return New(n, func(t float64) float64 {
		a := 2 * math.Pi * t
		return .5 * (1 - math.Cos(a))
	})
}

// SqrtHann は、Hann窓の平方根を生成します。
func SqrtHann(n int) Window {
	return New(n, func(t float64) float64 {
		a := 2 * math.Pi * t
		return math.Sqrt(.5 * (1 - math.Cos(a)))
	})
}

// Hamming は、Hamming窓を生成します。
func Hamming(n int) Window {
	return New(n, func(t float64) float64 {
		a := 2 * math.Pi * t
		return .54 - .46*math.Cos(a)
	})
}

// BlackmanHarris は、BlackmanHarris窓を生成します。
func BlackmanHarris(n int) Window {
	return New(n, func(t float64) float64 {
		a := 2 * math.Pi * t
		return .35875 - .48829*math.Cos(a) + .14128*math.Cos(2*a) - .01168*math.Cos(3*a)
	})
}

// Nuttall は、Nuttall窓を生成します。
func Nuttall(n int) Window {
	return New(n, func(t float64) float64 {
		a := 2 * math.Pi * t
		return .355768 - .487396*math.Cos(a) + .144232*math.Cos(2*a) - .012604*math.Cos(3*a)
	})
}

// FlatTop は、FlatTop窓を生成します。
func FlatTop(n int) Window {
	return New(n, func(t float64) float64 {
		a := 2 * math.Pi * t
		return 1 - 1.93*math.Cos(a) + 1.29*math.Cos(2*a) - .388*math.Cos(3*a) + .028*math.Cos(4*a)
	})
}

// Bartlett は、Bartlett窓を生成します。
func Bartlett(n int) Window {
	return New(n, func(t float64) float64 {
		return 1 - math.Abs(t*2-1)
	})
}

// Gauss は、Gauss窓を生成します。
func Gauss(n int, sigma float64) Window {
	return New(n, func(t float64) float64 {
		u := (t*2 - 1) / sigma
		return math.Pow(math.E, -.5*u*u)
	})
}
