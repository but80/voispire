package dio

// histc counts the number of values in vector x that fall between the
// elements in the edges vector (which must contain monotonically
// nondecreasing values). n is a length(edges) vector containing these counts.
// No elements of x can be complex.
// http://www.mathworks.co.jp/help/techdoc/ref/histc.html
//
// Input:
//   x              : Input vector
//   edges          : Input matrix (1-dimension)
//
// Output:
//   index          : Result counted in vector x
// Caution:
//   Lengths of index and edges must be the same.
func histc(intervals []interval, edges []float64, index []int) {
	count := 1

	i := 0
	for ; i < len(edges); i++ {
		index[i] = 1
		if edges[i] >= intervals[0].location {
			break
		}
	}
	for ; i < len(edges); i++ {
		if edges[i] < intervals[count].location {
			index[i] = count
		} else {
			index[i] = count
			i--
			count++
		}
		if count == len(intervals) {
			break
		}
	}
	count--
	for i++; i < len(edges); i++ {
		index[i] = count
	}
}

// Interp1 interpolates to find yi, the values of the underlying function Y
// at the points in the vector or array xi. x must be a vector.
// http://www.mathworks.co.jp/help/techdoc/ref/Interp1.html
//
// Input:
//   x          : Input vector (Time axis)
//   y          : Values at x[n]
//   xi         : Required vector
//
// Output:
//   yi         : Interpolated vector
func interp1(intervals []interval, xi []float64, yi []float64) {
	h := make([]float64, len(intervals)-1)
	s := make([]float64, len(xi))
	k := make([]int, len(xi))

	for i := 0; i < len(intervals)-1; i++ {
		h[i] = intervals[i+1].location - intervals[i].location
	}

	histc(intervals, xi, k)

	for i := range xi {
		s[i] = (xi[i] - intervals[k[i]-1].location) / h[k[i]-1]
	}

	for i := range xi {
		d := intervals[k[i]].interval - intervals[k[i]-1].interval
		yi[i] = intervals[k[i]-1].interval + s[i]*d
	}
}
