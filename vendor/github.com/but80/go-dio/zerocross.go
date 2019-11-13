package dio

type interval struct {
	interval float64
	location float64
}

// struct for getFourZeroCrossingIntervals()
// "negative" means "zero-crossing point going from positive to negative"
// "positive" means "zero-crossing point going from negative to positive"
type zeroCrossings struct {
	negatives []interval
	positives []interval
	peaks     []interval
	dips      []interval
}

func newZeroCrossings(capacity int) *zeroCrossings {
	return &zeroCrossings{
		negatives: make([]interval, 0, capacity),
		positives: make([]interval, 0, capacity),
		peaks:     make([]interval, 0, capacity),
		dips:      make([]interval, 0, capacity),
	}
}

// zeroCrossingEngine calculates the zero crossing points from positive to
// negative. Thanks to Custom.Maid http://custom-made.seesaa.net/ (2012/8/19)
func zeroCrossingEngine(filteredSignal []float64, fs float64, intervals *[]interval) {
	*intervals = (*intervals)[:0]
	n := len(filteredSignal)
	negativeGoingPoints := make([]int, n)

	for i := 0; i < n-1; i++ {
		if 0.0 < filteredSignal[i] && filteredSignal[i+1] <= 0.0 {
			negativeGoingPoints[i] = i + 1
		}
	}
	negativeGoingPoints[n-1] = 0

	edges := make([]int, n)
	count := 0
	for i := 0; i < n; i++ {
		if negativeGoingPoints[i] > 0 {
			edges[count] = negativeGoingPoints[i]
			count++
		}
	}

	if count < 2 {
		return
	}

	fineEdges := make([]float64, count)
	for i := 0; i < count; i++ {
		d := filteredSignal[edges[i]] - filteredSignal[edges[i]-1]
		fineEdges[i] = float64(edges[i]) - filteredSignal[edges[i]-1]/d
	}

	for i := 0; i < count-1; i++ {
		*intervals = append(*intervals, interval{
			interval: fs / (fineEdges[i+1] - fineEdges[i]),
			location: (fineEdges[i] + fineEdges[i+1]) / 2.0 / fs,
		})
	}
}
