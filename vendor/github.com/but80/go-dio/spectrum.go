package dio

import (
	"math"

	"github.com/but80/go-dio/constant"
	"github.com/but80/go-dio/internal/matlab"
)

// designLowCutFilter calculates the coefficients the filter.
func designLowCutFilter(n, fftSize int, lowCutFilter []float64) {
	for i := 1; i <= n; i++ {
		lowCutFilter[i-1] = 0.5 - 0.5*math.Cos(float64(i)*2.0*math.Pi/float64(n+1))
	}
	for i := n; i < fftSize; i++ {
		lowCutFilter[i] = 0.0
	}
	sumOfAmplitude := 0.0
	for i := 0; i < n; i++ {
		sumOfAmplitude += lowCutFilter[i]
	}
	for i := 0; i < n; i++ {
		lowCutFilter[i] = -lowCutFilter[i] / sumOfAmplitude
	}
	for i := 0; i < (n-1)/2; i++ {
		lowCutFilter[fftSize-(n-1)/2+i] = lowCutFilter[i]
	}
	for i := 0; i < n; i++ {
		lowCutFilter[i] = lowCutFilter[i+(n-1)/2]
	}
	lowCutFilter[0] += 1.0
}

// getSpectrumForEstimation calculates the spectrum for estimation.
// This function carries out downsampling to speed up the estimation process
// and calculates the spectrum of the downsampled signal.
func (s *Estimator) getSpectrumForEstimation() {
	y := make([]float64, s.fftSize)
	copy(y[:len(s.x)], s.x)

	// Removal of the DC component (y = y - mean value of y)
	meanY := 0.0
	for i := 0; i < s.yLength; i++ {
		meanY += y[i]
	}
	meanY /= float64(s.yLength)
	for i := 0; i < s.yLength; i++ {
		y[i] -= meanY
	}
	for i := s.yLength; i < s.fftSize; i++ {
		y[i] = 0.0
	}

	s.fft.Coefficients(s.ySpectrum[:s.fftSize/2+1], y)

	// Low cut filtering (from 0.1.4). Cut off frequency is 50.0 Hz.
	cutoffInSample := matlab.Round(s.fs / constant.CutOff)
	designLowCutFilter(cutoffInSample*2+1, s.fftSize, y)

	filterSpectrum := make([]complex128, s.fftSize/2+1)
	s.fft.Coefficients(filterSpectrum, y)

	for i := 0; i <= s.fftSize/2; i++ {
		s.ySpectrum[i] *= filterSpectrum[i]
	}
}
