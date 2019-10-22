package levinsondurbin

import (
	"math"
	"math/cmplx"

	"gonum.org/v1/gonum/fourier"
)

// SpectralEnvelopeDetector は、時間領域信号から周波数スペクトル包絡線を計算します。
type SpectralEnvelopeDetector struct {
	size               int
	lpcOrder           int
	fft                *fourier.FFT
	predictionCoefBuf1 []float64
	predictionCoefBuf2 []float64
	autocorrBuf        []float64
	impulseSpec        []float64
	freqzSpecDenomBuf  []complex128
	freqzResultBuf     []float64
}

func makeImpulseSpec(fft *fourier.FFT, size int) []float64 {
	impulse := make([]float64, size)
	impulse[0] = 1.0
	spec := fft.Coefficients(nil, impulse)
	r := 1 / float64(fft.Len())
	result := make([]float64, len(spec))
	for i, v := range spec {
		result[i] = cmplx.Abs(v) * r
	}
	return result
}

// NewSpectralEnvelopeDetector は、新しい SpectralEnvelopeDetector を作成します。
func NewSpectralEnvelopeDetector(size, lpcOrder int) *SpectralEnvelopeDetector {
	fft := fourier.NewFFT(size)
	return &SpectralEnvelopeDetector{
		size:               size,
		lpcOrder:           lpcOrder,
		fft:                fft,
		predictionCoefBuf1: make([]float64, size), // 実際に更新されるのは [0:size] の部分のみ（残りは0）
		predictionCoefBuf2: make([]float64, size), // 同上
		autocorrBuf:        make([]float64, lpcOrder+1),
		impulseSpec:        makeImpulseSpec(fft, size),
		freqzSpecDenomBuf:  make([]complex128, size/2+1),
		freqzResultBuf:     make([]float64, size/2+1),
	}
}

// calcAutocorr は、自己相関関数を計算します。
func (d *SpectralEnvelopeDetector) calcAutocorr(x []float64) []float64 {
	n := len(x)
	r := d.autocorrBuf
	for lag := range r {
		r[lag] = x[0] * x[lag]
		for i := 1; i < n-lag; i++ {
			r[lag] += x[i] * x[i+lag]
		}
	}
	return r
}

// levinsonDurbin は、LPC（線形予測分析）係数を計算します。
func (d *SpectralEnvelopeDetector) levinsonDurbin(r []float64) ([]float64, float64) {
	a := d.predictionCoefBuf1

	a[0] = 1.0
	a[1] = -r[1] / r[0]
	e := r[0] + r[1]*a[1]

	a2 := d.predictionCoefBuf2
	for k := 1; k < d.lpcOrder; k++ {
		lam := .0
		for j := 0; j <= k; j++ {
			lam -= a[j] * r[k+1-j]
		}
		lam /= e

		a2[0] = 1.0
		for i := 0; i < k; i++ {
			a2[1+i] = a[i+1] + lam*a[k-i]
		}
		a2[1+k] = lam
		a, a2 = a2, a

		e *= 1.0 - lam*lam
	}

	return a, e
}

// freqz は、フィルタ係数の周波数応答を計算します。
func (d *SpectralEnvelopeDetector) freqz(b float64, a []float64) []float64 {
	q := d.fft.Coefficients(d.freqzSpecDenomBuf, a)
	h := d.freqzResultBuf
	for i := range h {
		h[i] = b * d.impulseSpec[i] / cmplx.Abs(q[i])
	}
	return h
}

// func predict(x, coefs []float64) []float64 {
// 	result := make([]float64, len(x))
// 	copy(result, x)
// 	m := len(coefs)
// 	for i := m; i < len(result); i++ {
// 		result[i] = 0.0
// 		for j := 0; j < m; j++ {
// 			result[i] -= coefs[j] * x[i-1-j]
// 		}
// 	}
// 	return result
// }

// Detect は、時間領域信号 signal の周波数スペクトル包絡線を計算します。
func (d *SpectralEnvelopeDetector) Detect(signal []float64) []float64 {
	r := d.calcAutocorr(signal)
	a, e := d.levinsonDurbin(r)
	return d.freqz(math.Sqrt(e), a)
}
