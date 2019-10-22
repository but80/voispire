package series

import "math"

// ExtendFloatSlice は、数列 a の長さを n だけ拡張します。
func ExtendFloatSlice(a []float64, n int) []float64 {
	l := len(a) + n
	for len(a) < l {
		a = append(a, 0)
	}
	return a
}

// ExtendFloatSliceCeil は、数列 a の長さが n の整数倍になるように拡張します。
func ExtendFloatSliceCeil(a []float64, n int) []float64 {
	r := float64(len(a)) / float64(n)
	l := int(math.Ceil(r)) * n
	for len(a) < l {
		a = append(a, 0)
	}
	return a
}

// CmplxMulFloatConst は、複素数数列 src に対し実数 r を乗じたものを dst に格納します。
func CmplxMulFloatConst(dst, src []complex128, r float64) {
	rc := complex(r, 0)
	for i, v := range src {
		dst[i] = v * rc
	}
}

// CmplxDivFloatConst は、複素数数列 src に対し実数 r で除したものを dst に格納します。
func CmplxDivFloatConst(dst, src []complex128, r float64) {
	CmplxMulFloatConst(dst, src, 1/r)
}
