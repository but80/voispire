package buffer

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func roundf(v, p float64) float64 {
	return math.Round(v/p) * p
}

func TestShape_getLagrange(t *testing.T) {
	sh := MakeShape([]float64{
		-.04, // phase=0.0/8.0
		.00,  // phase=2.0/8.0
		.04,  // phase=4.0/8.0
		.16,  // phase=6.0/8.0
	})
	assert.Equal(t, -.04, roundf(sh.getLagrange(0.0/8.0), .001))
	assert.Equal(t, -.02, roundf(sh.getLagrange(1.0/8.0), .001)) // linear
	assert.Equal(t, .00, roundf(sh.getLagrange(2.0/8.0), .001))
	assert.Equal(t, .02, roundf(sh.getLagrange(3.0/8.0), .001)) // linear
	assert.Equal(t, .04, roundf(sh.getLagrange(4.0/8.0), .001))
	assert.Equal(t, .09, roundf(sh.getLagrange(5.0/8.0), .001)) // lagrange
	assert.Equal(t, .16, roundf(sh.getLagrange(6.0/8.0), .001))
	assert.Equal(t, .16, roundf(sh.getLagrange(7.0/8.0), .001)) // keep last sample
}
