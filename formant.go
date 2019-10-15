package voispire

import (
	"math/cmplx"
	"sort"
)

// https://synsinger.wordpress.com/2015/11/21/pitch-shifting-using-a-spectral-envelope/

type peak struct {
	index int
	level float64
}

type formantShifter struct {
	*fftProcessor
	width      int
	fs         int
	shiftInv   float64
	maxPeakNum int
	ampBuf     []float64
	envBuf     []float64
	peaksBuf   []peak
}

var onFormantFFTProcess func(*formantShifter, []float64, []float64, []complex128, []complex128)
var onFormantFFTFinish func(*formantShifter)

func newFormantShifter(src []float64, fs, width int, shift float64) *formantShifter {
	shiftInv := 1.0 / shift
	maxPeakNum := 100
	s := &formantShifter{
		width:    width,
		fs:       fs,
		ampBuf:   make([]float64, width/2+1),
		envBuf:   make([]float64, width/2+1),
		peaksBuf: make([]peak, 0, maxPeakNum),
	}
	s.fftProcessor = newFFTProcessor(src, width, func(spec []complex128) []complex128 {
		if len(spec) <= 4 {
			return spec
		}
		n := len(spec)
		peaks := s.findPeaks(spec, maxPeakNum)
		env := s.peaksToEnvelope(peaks)
		for i := 0; i < n; i++ {
			j := int(float64(i) * shiftInv)
			if n <= j {
				j = n - 1
			}
			spec[i] *= complex(env[j]/env[i], .0)
		}
		return spec
	})
	s.fftProcessor.OnProcess = func(wave0, wave1 []float64, spec0, spec1 []complex128) {
		if onFormantFFTProcess != nil {
			onFormantFFTProcess(s, wave0, wave1, spec0, spec1)
		}
	}
	s.fftProcessor.OnFinish = func() {
		if onFormantFFTFinish != nil {
			onFormantFFTFinish(s)
		}
	}
	return s
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

func (s *formantShifter) findPeaks(spec []complex128, peakNum int) []peak {
	n := len(spec)
	amp := s.ampBuf
	for i, v := range spec {
		amp[i] = cmplx.Abs(v)
	}
	peaks := s.peaksBuf[0:0]
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

func (s *formantShifter) peaksToEnvelope(peaks []peak) []float64 {
	result := s.envBuf
	p0 := peak{index: 0, level: peaks[0].level}
	var p1 peak
	for i := 0; i <= len(peaks); i++ {
		if i < len(peaks) {
			p1 = peaks[i]
		} else {
			p1 = peak{index: s.width/2 + 1, level: p0.level}
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
