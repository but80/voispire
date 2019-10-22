package formant

import (
	"math/cmplx"
	"sort"

	"github.com/but80/voispire/internal/fft"
)

// https://synsinger.wordpress.com/2015/11/21/pitch-shifting-using-a-spectral-envelope/

type peak struct {
	index int
	level float64
}

type brokenlineShifter struct {
	fft.FFTProcessor
	width      int
	fs         int
	shiftInv   float64
	maxPeakNum int
	ampBuf     []float64
	peaksBuf   []peak
	envelope   []float64
	spec1      []complex128
}

// NewBrokenlineShifter は、折れ線近似を用いたフォルマントシフタを作成します。
func NewBrokenlineShifter(src []float64, fs, width int, shift float64) FormantShifter {
	maxPeakNum := 100
	s := &brokenlineShifter{
		width:    width,
		fs:       fs,
		ampBuf:   make([]float64, width/2+1),
		peaksBuf: make([]peak, 0, maxPeakNum),
		envelope: make([]float64, width/2+1),
		spec1:    make([]complex128, width/2+1),
	}
	s.FFTProcessor = fft.NewFFTProcessor(src, width, func(spec0 []complex128, wave0 []float64) []complex128 {
		if len(spec0) <= 4 {
			return spec0
		}
		peaks := s.findPeaks(spec0, maxPeakNum)
		env := s.peaksToEnvelope(peaks)
		applyEnvelopeShift(s.spec1, spec0, env, shift)
		analyzerFrame(&analyzerData{
			fs:       fs,
			fftWidth: width,
			wave0:    wave0,
			envelope: s.envelope,
			spec0:    spec0,
			spec1:    s.spec1,
		})
		return s.spec1
	})
	s.FFTProcessor.OnFinish(analyzerFinish)
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

func (s *brokenlineShifter) findPeaks(spec []complex128, peakNum int) []peak {
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

func (s *brokenlineShifter) peaksToEnvelope(peaks []peak) []float64 {
	result := s.envelope
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
