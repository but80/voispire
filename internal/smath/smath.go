package smath

import "math"

// Sinc は、sinc関数です。
func Sinc(t float64) float64 {
	if t == .0 {
		return 1.0
	}
	return math.Sin(t) / t
}

// SincNormalized は、正規化sinc関数です。
func SincNormalized(t float64) float64 {
	return Sinc(math.Pi * t)
}

// Sign は、v の符号を -1, 0, 1 のいずれかで返します。
func Sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	if 0 < v {
		return 1
	}
	return 0
}

// SignedSqrt は、符号を維持して絶対値部分のみ平方根を取ります。
func SignedSqrt(v float64) float64 {
	return Sign(v) * math.Sqrt(math.Abs(v))
}
