package dio

import (
	"math"

	"github.com/but80/go-dio/constant"
	"github.com/but80/go-dio/internal/common"
	"github.com/but80/go-dio/internal/matlab"
	"gonum.org/v1/gonum/fourier"
)

// Option is the struct to order the parameter for Dio.
type Option struct {
	F0Floor          float64
	F0Ceil           float64
	ChannelsInOctave float64
	FramePeriod      float64 // msec
	Speed            int     // (1, 2, ..., 12)
	AllowedRange     float64 // Threshold used for fixing the F0 contour.
}

// NewOption creates new Option with the default parameters.
func NewOption() *Option {
	option := &Option{}

	// You can change default parameters.
	option.ChannelsInOctave = 2.0
	option.F0Ceil = constant.CeilF0
	option.F0Floor = constant.FloorF0
	option.FramePeriod = 5

	// You can use the value from 1 to 12.
	// Default value 11 is for the fs of 44.1 kHz.
	// The lower value you use, the better performance you can obtain.
	option.Speed = 1

	// You can give a positive real number as the threshold.
	// The most strict value is 0, and there is no upper limit.
	// On the other hand, I think that the value from 0.02 to 0.2 is reasonable.
	option.AllowedRange = 0.1

	return option
}

type candidate struct {
	f0    float64
	score float64
}

type params struct {
	fs      float64 // Sampling frequency
	xLength int
	option  *Option // Struct to order the parameter for DIO

	f0Length          int
	yLength           int
	numberOfBands     int // Max number of candidates
	voiceRangeMinimum int // Number of consecutive frames for stable estimation
	fftSize           int
	fft               *fourier.FFT
	boundaryF0List    []float64
}

func newParams(xLength int, fs float64, option *Option) *params {
	p := &params{
		fs:      fs,
		xLength: xLength,
		option:  option,

		f0Length:          int(1000.0*float64(xLength)/fs/option.FramePeriod) + 1,
		yLength:           1 + xLength,
		numberOfBands:     1 + int(math.Log2(option.F0Ceil/option.F0Floor)*option.ChannelsInOctave),
		voiceRangeMinimum: int(0.5+1000.0/option.FramePeriod/option.F0Floor)*2 + 1,
	}

	p.boundaryF0List = make([]float64, p.numberOfBands)
	for i := 0; i < p.numberOfBands; i++ {
		p.boundaryF0List[i] = option.F0Floor * math.Pow(2.0, float64(i+1)/option.ChannelsInOctave)
	}

	p.fftSize = common.GetSuitableFFTSize(p.yLength +
		matlab.Round(fs/constant.CutOff)*2 + 1 +
		4*int(1.0+fs/p.boundaryF0List[0]/2.0))
	p.fft = fourier.NewFFT(p.fftSize)
	return p
}

// Estimator is the struct holds the variables needed to estimate f0 by Dio.
type Estimator struct {
	*params

	// Inputs
	x []float64 // Input signal

	// Temporaries
	ySpectrum     []complex128
	zeroCrossings *zeroCrossings
	f0Candidates  [][]candidate
	f0Candidate   []candidate
	lpfSpectrum   map[int][]complex128

	// Outputs
	temporalPositions []float64 // Temporal positions.
	f0                []float64 // F0 contour.
}

// New creates new Estimator.
func New(x []float64, fs float64, option *Option) *Estimator {
	if option == nil {
		option = NewOption()
	}
	p := newParams(len(x), fs, option)
	s := newEstimator(p)
	s.x = x
	return s
}

func newEstimator(p *params) *Estimator {
	s := &Estimator{
		params:            p,
		ySpectrum:         make([]complex128, p.fftSize),
		zeroCrossings:     newZeroCrossings(p.yLength),
		f0Candidates:      make([][]candidate, p.numberOfBands),
		f0Candidate:       make([]candidate, p.f0Length),
		lpfSpectrum:       make(map[int][]complex128),
		temporalPositions: make([]float64, p.f0Length),
		f0:                make([]float64, p.f0Length),
	}
	for i := range s.f0Candidates {
		s.f0Candidates[i] = make([]candidate, p.f0Length)
	}
	for i := range s.temporalPositions {
		s.temporalPositions[i] = float64(i) * p.option.FramePeriod / 1000.0
	}
	return s
}

// Estimate estimates the F0 based on Distributed Inline-filter Operation.
func (s *Estimator) Estimate() ([]float64, []float64) {
	// Calculation of the spectrum used for the f0 estimation
	s.getSpectrumForEstimation()

	s.getF0CandidatesAndScores()

	// Selection of the best value based on fundamental-ness.
	// This function is related with SortCandidates() in MATLAB.
	bestF0Contour := make([]float64, s.f0Length)
	s.getBestF0Contour(bestF0Contour)

	// Postprocessing to find the best f0-contour.
	s.fixF0Contour(bestF0Contour, s.f0)

	return s.temporalPositions, s.f0
}
