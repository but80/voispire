package voispire

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindPeak(t *testing.T) {
	p := findPeak([]float64{.0, 1.0, 2.0, 3.0, -3.0, -2.0, -1.0, .0})
	assert.Equal(t, peak{index: 3, level: 3.0}, p)
}

func TestFindPeaks(t *testing.T) {
	peaks := findPeaks([]complex128{
		.0, 13.0, 23.0, 33.0, 43.0, 53.0, 43.0, 33.0, 23.0, 13.0,
		.0i, 16.0i, 26.0i, 36.0i, 46.0i, 56.0i, 46.0i, 36.0i, 26.0i, 16.0i,
		.0, 10.0, 20.0, 30.0, 40.0, 50.0, 40.0, 30.0, 20.0, 10.0,
	}, 3)
	assert.Equal(t, []peak{
		{index: 5, level: 53.0},
		{index: 15, level: 56.0},
		{index: 25, level: 50.0},
	}, peaks)
}
