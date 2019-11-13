package matlab

// Round calculates rounding.
func Round(x float64) int {
	if 0 < x {
		return int(x + .5)
	}
	return int(x - .5)
}
