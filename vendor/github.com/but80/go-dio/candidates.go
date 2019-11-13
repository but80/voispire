package dio

import (
	"math"

	"github.com/but80/go-dio/constant"
	"github.com/but80/go-dio/internal/common"
	"github.com/but80/go-dio/internal/matlab"
)

func (s *Estimator) nuttallWindowSpectrum(result []complex128, halfAverageLength int) {
	if lpfSpectrum, ok := s.lpfSpectrum[halfAverageLength]; ok {
		copy(result, lpfSpectrum)
		return
	}

	lpf := make([]float64, s.fftSize)
	// Nuttall window is used as a low-pass filter.
	// Cutoff frequency depends on the window length.
	common.NuttallWindow(lpf[:halfAverageLength*4])
	lpfSpectrum := make([]complex128, s.fftSize/2+1)
	s.fft.Coefficients(lpfSpectrum, lpf)
	s.lpfSpectrum[halfAverageLength] = lpfSpectrum
	copy(result, lpfSpectrum)
}

// getFilteredSignal calculates the signal that is the convolution of the
// input signal and low-pass filter.
// This function is only used in rawEventByDio()
func (s *Estimator) getFilteredSignal(halfAverageLength int, filteredSignal []float64) {
	spectrum := make([]complex128, s.fftSize/2+1)
	s.nuttallWindowSpectrum(spectrum, halfAverageLength)

	// Convolution
	spectrum[0] *= s.ySpectrum[0]
	for i := 1; i <= s.fftSize/2; i++ {
		spectrum[i] *= s.ySpectrum[i]
	}

	s.fft.Sequence(filteredSignal, spectrum)

	// Compensation of the delay.
	indexBias := halfAverageLength * 2
	for i := 0; i < s.yLength; i++ {
		filteredSignal[i] = filteredSignal[i+indexBias]
	}
}

// getFourZeroCrossingIntervals() calculates four zero-crossing intervals.
// (1) Zero-crossing going from negative to positive.
// (2) Zero-crossing going from positive to negative.
// (3) Peak, and (4) dip. (3) and (4) are calculated from the zero-crossings of
// the differential of waveform.
func (s *Estimator) getFourZeroCrossingIntervals(filteredSignal []float64) {
	// xLength / 4 (old version) is fixed at 2013/07/14
	zeroCrossingEngine(filteredSignal[:s.yLength], s.fs, &s.zeroCrossings.negatives)

	for i, v := range filteredSignal {
		filteredSignal[i] = -v
	}
	zeroCrossingEngine(filteredSignal[:s.yLength], s.fs, &s.zeroCrossings.positives)

	for i := 0; i < s.yLength-1; i++ {
		filteredSignal[i] -= filteredSignal[i+1]
	}
	zeroCrossingEngine(filteredSignal[:s.yLength-1], s.fs, &s.zeroCrossings.peaks)

	for i, v := range filteredSignal {
		filteredSignal[i] = -v
	}
	zeroCrossingEngine(filteredSignal[:s.yLength-1], s.fs, &s.zeroCrossings.dips)
}

// getF0CandidateContourSub calculates the f0 candidates and deviations.
// This is the sub-function of getF0Candidates() and assumes the calculation.
func (s *Estimator) getF0CandidateContourSub(interpolatedF0Set [4][]float64, boundaryF0 float64) {
	for i := 0; i < s.f0Length; i++ {
		c := &s.f0Candidate[i]

		c0 := interpolatedF0Set[0][i]
		c1 := interpolatedF0Set[1][i]
		c2 := interpolatedF0Set[2][i]
		c3 := interpolatedF0Set[3][i]
		c.f0 = (c0 + c1 + c2 + c3) / 4.0

		if c.f0 < boundaryF0/2.0 || boundaryF0 < c.f0 ||
			c.f0 < s.option.F0Floor || s.option.F0Ceil < c.f0 {
			c.f0 = 0.0
			c.score = constant.MaximumValue
			continue
		}

		d0 := c0 - c.f0
		d1 := c1 - c.f0
		d2 := c2 - c.f0
		d3 := c3 - c.f0
		c.score = math.Sqrt((d0*d0 + d1*d1 + d2*d2 + d3*d3) / 3.0)
	}
}

// getF0CandidateContour() calculates the F0 candidates based on the
// zero-crossings.
func (s *Estimator) getF0CandidateContour(boundaryF0 float64) {
	if len(s.zeroCrossings.negatives) <= 2 ||
		len(s.zeroCrossings.positives) <= 2 ||
		len(s.zeroCrossings.peaks) <= 2 ||
		len(s.zeroCrossings.dips) <= 2 {
		for i := 0; i < s.f0Length; i++ {
			s.f0Candidate[i] = candidate{
				f0:    0.0,
				score: constant.MaximumValue,
			}
		}
		return
	}

	var interpolatedF0Set [4][]float64
	for i := 0; i < 4; i++ {
		interpolatedF0Set[i] = make([]float64, s.f0Length)
	}

	interp1(s.zeroCrossings.negatives, s.temporalPositions, interpolatedF0Set[0])
	interp1(s.zeroCrossings.positives, s.temporalPositions, interpolatedF0Set[1])
	interp1(s.zeroCrossings.peaks, s.temporalPositions, interpolatedF0Set[2])
	interp1(s.zeroCrossings.dips, s.temporalPositions, interpolatedF0Set[3])

	s.getF0CandidateContourSub(interpolatedF0Set, boundaryF0)
}

// getF0CandidateFromRawEvent() calculates F0 candidate contour in 1-ch signal
func (s *Estimator) getF0CandidateFromRawEvent(boundaryF0 float64) {
	filteredSignal := make([]float64, s.fftSize)
	s.getFilteredSignal(matlab.Round(s.fs/boundaryF0/2.0), filteredSignal)
	s.getFourZeroCrossingIntervals(filteredSignal)
	s.getF0CandidateContour(boundaryF0)
}

// getF0CandidatesAndScores calculates all f0 candidates and their scores.
func (s *Estimator) getF0CandidatesAndScores() {
	// Calculation of the acoustics events (zero-crossing)
	for i := 0; i < s.numberOfBands; i++ {
		s.getF0CandidateFromRawEvent(s.boundaryF0List[i])
		for j := 0; j < s.f0Length; j++ {
			c := s.f0Candidate[j]
			s.f0Candidates[i][j] = candidate{
				f0:    c.f0,
				score: c.score / (c.f0 + constant.MySafeGuardMinimum),
			}
		}
	}
}
