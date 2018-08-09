package voispire

import (
	"math"
)

// 参考論文: https://arxiv.org/abs/0911.5171
//
// Mx(t) = Σk{ x(at+k)·sinc((v−a)t − k) }
// sinc(t) = t=0 ? 1 : sin(πt) / πt
//
// x(t) の軸 t は「元の波形の位相(t周目)」＝周波数を無視して1周期分を1と数える
//   - k=±1 が隣の1周分の波形を指すと考えれば、このような解釈になるはず
//   - 再生周波数 a [Hz] が元波形の周波数と等しければそのままの音高で再生される
//   - 再生速度 v が a と等しければそのままの速度で再生される
// 元波形が4Hzなら
//   - v=4 a=4 : 等速・等音高
//   - v=8 a=4 : 倍速・等音高
//   - v=4 a=8 : 等速・+1oct
// 計算時は sinc が 1 に近い k に絞って Σ を取る
//   - k ∈ round((v−a)t) の前後±N

const (
	// sigmaWidth は、sinc関数による補間に前後各いくつの波形を考慮するかを表します。
	sigmaWidth = 2
)

// sinc は、正規化sinc関数です。
func sinc(t float64) float64 {
	if t == .0 {
		return 1.0
	}
	pt := math.Pi * t
	return math.Sin(pt) / pt
}

// shape は、1周期分の波形です。
type shape struct {
	// flen は、float64(len(data)) の値を保持します。
	// この逆数がオリジナルの周波数に一致します。
	flen float64
	// data は、波形データです。
	data []float64
}

// get は、指定した位相におけるこの波形の振幅を取得します。
func (sh *shape) get(phase float64) float64 {
	// TODO: lerp補間？
	return sh.data[int(math.Floor(phase*sh.flen))]
}

// shifter は、ピッチシフタです。
type shifter struct {
	fs       int
	totalLen int
	shapes   []shape
}

// addShape は、1周期分の波形を追加します。
func (sh *shifter) addShape(data []float64) {
	sh.totalLen += len(data)
	sh.shapes = append(sh.shapes, shape{
		flen: float64(len(data)),
		data: data,
	})
}

// play は、指定したピッチ係数・速度係数で再生した波形を返します。
func (sh *shifter) play(pitchCoef, speedCoef float64) []float64 {
	result := make([]float64, 0, sh.totalLen)
	phase := .0
	i := .0
	for iShape := sigmaWidth; iShape < len(sh.shapes)-sigmaWidth; iShape++ {
		n := sh.shapes[iShape].flen
		dphase := 1.0 / n * pitchCoef
		di := speedCoef / n
		for ; i < 1.0; i += di {
			v := .0
			sincPhase := phase - i
			for j := -sigmaWidth; j <= sigmaWidth; j++ {
				v += sh.shapes[iShape+j].get(phase) * sinc(sincPhase+float64(j))
			}
			result = append(result, v)
			phase += dphase
			for 1.0 <= phase {
				phase -= 1.0
			}
		}
		for 1.0 <= i {
			i -= 1.0
		}
	}
	return result
}
