package voispire

import (
	"log"
	"math/cmplx"
	"sort"

	"github.com/but80/voispire/internal/buffer"
	"github.com/mjibson/go-dsp/fft"
)

// https://synsinger.wordpress.com/2015/11/21/pitch-shifting-using-a-spectral-envelope/

type formantShifter struct {
	output     chan buffer.Shape
	input      chan buffer.Shape
	shiftInv   float64
	maxPeakNum int
}

func newFormantShifter(shift float64) *formantShifter {
	return &formantShifter{
		output:     make(chan buffer.Shape, 16),
		shiftInv:   1.0 / shift,
		maxPeakNum: 100,
	}
}

func (s *formantShifter) Input(input chan buffer.Shape) {
	s.input = input
}

func (s *formantShifter) Output() chan buffer.Shape {
	return s.output
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

func (s *formantShifter) Start() {
	go func() {
		log.Print("debug: formantShifter goroutine is started")
		for shape := range s.input {
			data := shape.Data()
			specSrc := fft.FFTReal(data)
			specDst := make([]complex128, len(specSrc))
			n := (len(specDst) + 1) / 2
			peaks := findPeaks(specSrc[:n], s.maxPeakNum)
			env := peaksToEnvelope(n, peaks)
			for i := 1; i < n; i++ {
				j := int(float64(i) * s.shiftInv)
				if n <= j {
					j = n - 1
				}
				specDst[i] = specSrc[i] * complex(env[j]/env[i], .0)
			}
			resultc := fft.IFFT(specDst)
			result := make([]float64, len(data))
			for i := 0; i < len(data); i++ {
				result[i] = real(resultc[i]) * 2.0
			}
			s.output <- buffer.MakeShape(result)
		}
		close(s.output)
	}()
}
