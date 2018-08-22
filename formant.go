package voispire

import (
	"log"

	"github.com/but80/voispire/internal/buffer"
)

type formantShifter struct {
	output chan buffer.Shape
	input  chan buffer.Shape
	shift  float64
}

func newFormantShifter(shift float64) *formantShifter {
	return &formantShifter{
		output: make(chan buffer.Shape, 16),
		shift:  shift,
	}
}

func (s *formantShifter) Input(input chan buffer.Shape) {
	s.input = input
}

func (s *formantShifter) Output() chan buffer.Shape {
	return s.output
}

func (s *formantShifter) Start() {
	go func() {
		log.Print("debug: formantShifter goroutine is started")
		for shape := range s.input {
			s.output <- shape
		}
		close(s.output)
	}()
}
