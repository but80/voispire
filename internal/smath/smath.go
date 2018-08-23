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
