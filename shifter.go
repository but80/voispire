package voispire

import (
	"math"
)

// https://arxiv.org/abs/0911.5171
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

func sinc(t float64) float64 {
	if t == .0 {
		return 1.0
	}
	pt := math.Pi * t
	return math.Sin(pt) / pt
}

type shape struct {
	flen float64
	data []float64
}

func (sh *shape) get(phase float64) float64 {
	return sh.data[int(math.Floor(phase*sh.flen))]
}

var sigmaWidth = 2

type shifter struct {
	Fs       int
	totalLen int
	shapes   []shape
}

func (sh *shifter) addShape(data []float64) {
	sh.totalLen += len(data)
	sh.shapes = append(sh.shapes, shape{
		flen: float64(len(data)),
		data: data,
	})
}

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
