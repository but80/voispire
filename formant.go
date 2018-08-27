package voispire

import (
	"math/cmplx"
	"sort"
)

// https://synsinger.wordpress.com/2015/11/21/pitch-shifting-using-a-spectral-envelope/

type formantShifter struct {
	*fftProcessor
	shiftInv   float64
	maxPeakNum int
}

func newFormantShifter(src []float64, width int, shift float64) *formantShifter {
	shiftInv := 1.0 / shift
	maxPeakNum := 100
	result := &formantShifter{}
	result.fftProcessor = newFFTProcessor(src, width, func(spec []complex128) []complex128 {
		if len(spec) <= 2 {
			return spec
		}
		n := ((len(spec) | 1) + 1) >> 1
		peaks := findPeaks(spec[:n], maxPeakNum)
		env := peaksToEnvelope(n, peaks)
		for i := 0; i < n; i++ {
			j := int(float64(i) * shiftInv)
			if n <= j {
				j = n - 1
			}
			spec[i] *= complex(2.0*env[j]/env[i], .0)
		}
		for i := n; i < len(spec); i++ {
			spec[i] = .0
		}
		return spec
	})
	return result
}

type peak struct {
	index int
	level float64
}

func findPeak(spec []float64) peak {
	result := peak{index: -1, level: .0}
	for i, level := range spec {
		if result.level < level {
			result.index = i
			result.level = level
		}
	}
	return result
}

func findPeaks(spec []complex128, peakNum int) []peak {
	n := len(spec)
	amp := make([]float64, n)
	for i, v := range spec {
		amp[i] = cmplx.Abs(v)
	}
	peaks := []peak{}
	m := n/peakNum - 1
	if m < 0 {
		m = 1
	}
	for i := 0; i < peakNum; i++ {
		p := findPeak(amp)
		if p.index < 0 {
			break
		}
		peaks = append(peaks, p)
		for j := p.index - m; j <= p.index+m; j++ {
			if 0 <= j && j < n {
				amp[j] = .0
			}
		}
	}
	sort.Slice(peaks, func(i, j int) bool {
		return peaks[i].index < peaks[j].index
	})
	return peaks
}

func peaksToEnvelope(n int, peaks []peak) []float64 {
	result := make([]float64, n)
	p0 := peak{index: 0, level: peaks[0].level}
	var p1 peak
	for i := 0; i <= len(peaks); i++ {
		if i < len(peaks) {
			p1 = peaks[i]
		} else {
			p1 = peak{index: n, level: p0.level}
		}
		m := p1.index - p0.index
		level := p0.level
		diff := (p1.level - p0.level) / float64(m)
		for j := p0.index; j < p1.index; j++ {
			result[j] = level
			level += diff
		}
		p0 = p1
	}
	return result
}
