package voispire

import (
	"log"

	"github.com/but80/voispire/internal/buffer"
)

func shiftFormant(input chan buffer.Shape, shift float64) chan buffer.Shape {
	out := make(chan buffer.Shape, 16)
	go func() {
		log.Print("debug: shiftFormant goroutine is started")
		for s := range input {
			out <- s
		}
		close(out)
	}()
	return out
}
