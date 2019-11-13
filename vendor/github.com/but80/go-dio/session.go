package dio

import (
	"math"

	"github.com/but80/go-dio/internal/common"
)

type Session struct {
	*Estimator
	params *params
	time   int
}

// NewSession creates new Session.
func NewSession(fs float64, option *Option) *Session {
	if option == nil {
		option = NewOption()
	}
	t := math.Ceil(fs / option.F0Floor * 20) // 最長周期の20倍 [sample]
	fp := option.FramePeriod / 1000.0        // [sec]

	// f0Length = int(xLength/(fs*FramePeriod)) + 1
	// xLength をf0検出単位の整数倍（4の倍数倍）に取る
	b := math.Round(fs * fp * 4)       // [sample]
	xLength := int(math.Ceil(t/b) * b) // [sample]

	p := newParams(xLength, fs, option)
	s := &Session{
		params:    p,
		Estimator: newEstimator(p),
	}
	return s
}

func (s *Session) Len() int {
	return s.Estimator.params.xLength
}

func (s *Session) F0Length() int {
	return s.Estimator.params.f0Length
}

func (s *Session) FramePeriod() float64 {
	return s.Estimator.params.option.FramePeriod
}

func (s *Session) Estimate(x []float64) []float64 {
	if len(x) < s.Len() {
		x = append(x, make([]float64, s.Len()-len(x))...)
	}
	if s.Len() < len(x) {
		x = x[:s.Len()]
	}
	s.Estimator = newEstimator(s.params)
	s.Estimator.x = x
	_, f0 := s.Estimator.Estimate()
	return f0
}

func (s *Session) Start() (chan<- []float64, <-chan []float64) {
	input := make(chan []float64, 1000)
	output := make(chan []float64, 1000)
	go func() {
		var buffer []float64
		step := s.Len() / 2
		from := 0
		for {
			inputWave, inputOk := <-input
			if inputOk {
				buffer = append(buffer, inputWave...)
			}

			for s.Len() <= len(buffer) || 0 < len(buffer) && !inputOk {
				j := common.MinInt(s.Len(), len(buffer))
				f0 := s.Estimate(buffer[:j])
				buffer = buffer[common.MinInt(step, len(buffer)):]
				to := s.F0Length() * 3 / 4
				if j == len(buffer) && !inputOk {
					to = s.F0Length()
					buffer = buffer[:0]
				}
				output <- f0[from:to]
				from = s.F0Length() / 4
			}

			if !inputOk {
				break
			}
		}
		close(output)
	}()
	return input, output
}
