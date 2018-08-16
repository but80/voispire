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
	sh1 := MakeShape([]float64{
		-.04, // phase=0.0/8.0
		.00,  // phase=2.0/8.0
		.04,  // phase=4.0/8.0
		.16,  // phase=6.0/8.0
	})
	assert.Equal(t, -.04, roundf(sh1.getLagrange(0.0/8.0), .001))
	assert.Equal(t, -.02, roundf(sh1.getLagrange(1.0/8.0), .001)) // linear
	assert.Equal(t, .00, roundf(sh1.getLagrange(2.0/8.0), .001))
	assert.Equal(t, .02, roundf(sh1.getLagrange(3.0/8.0), .001)) // linear
	assert.Equal(t, .04, roundf(sh1.getLagrange(4.0/8.0), .001))
	assert.Equal(t, .09, roundf(sh1.getLagrange(5.0/8.0), .001)) // lagrange
	assert.Equal(t, .16, roundf(sh1.getLagrange(6.0/8.0), .001))
	assert.Equal(t, .16, roundf(sh1.getLagrange(7.0/8.0), .001)) // keep last sample
	//
	sh2 := MakeShapeTrimmed([]float64{
		-.16, // phase=-4.0/8.0
		-.04, // phase=-2.0/8.0
		.00,  // phase=0.0/8.0
		.04,  // phase=2.0/8.0
		.16,  // phase=4.0/8.0
		.36,  // phase=6.0/8.0
		.64,  // phase=8.0/8.0
	}, 2, 2+4)
	assert.Equal(t, .00, roundf(sh2.getLagrange(0.0/8.0), .001))
	assert.Equal(t, .02, roundf(sh2.getLagrange(1.0/8.0), .001)) // ?
	assert.Equal(t, .04, roundf(sh2.getLagrange(2.0/8.0), .001))
	assert.Equal(t, .09, roundf(sh2.getLagrange(3.0/8.0), .001)) // lagrange
	assert.Equal(t, .16, roundf(sh2.getLagrange(4.0/8.0), .001))
	assert.Equal(t, .25, roundf(sh2.getLagrange(5.0/8.0), .001)) // lagrange
	assert.Equal(t, .36, roundf(sh2.getLagrange(6.0/8.0), .001))
	assert.Equal(t, .49, roundf(sh2.getLagrange(7.0/8.0), .001)) // lagrange
	//
}
