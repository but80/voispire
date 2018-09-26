package world

import "math"

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
	return int(math.Pow(2.0, math.Floor(math.Log(float64(sample))/kLog2)+1.0))
}
