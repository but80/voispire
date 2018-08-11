package voispire

import (
	"log"
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
	// sigmaTotal は、sinc関数による補間時に考慮する全波形の個数です。
	sigmaTotal = sigmaWidth*2 + 1
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
	// flen は、波形データのサンプル数 float64(len(data)) の値を保持します。
	flen float64
	// freq は、波形データのオリジナルの周波数 [1/サンプル] で、1/flen に一致します。
	freq float64
	// data は、波形データです。各要素は振幅 -1≦v≦1 を表します。
	data []float64
}

func makeShape(data []float64) shape {
	flen := float64(len(data))
	return shape{
		flen: flen,
		freq: 1.0 / flen,
		data: data,
	}
}

// get は、指定した位相 0≦phase＜1 におけるこの波形の振幅を取得します。
func (sh *shape) get(phase float64) float64 {
	// TODO: lerp補間？
	return sh.data[int(math.Floor(phase*sh.flen))]
}

// shapeBuffer は、波形の履歴を保持するバッファです。
type shapeBuffer struct {
	shapes []shape
}

// get は、現在バッファの中心にある波形の位相 0≦srcPhase＜1 における振幅を返します。
// 返される振幅は、前後各 sigmaWidth 個の波形の同じ位相における振幅を用いて補間された値で、
// 0≦dstPhase＜1 はその係数に用いられます（小さいほど前方、大きいほど後方の波形に比重が置かれます）。
func (buf *shapeBuffer) get(srcPhase, dstPhase float64) float64 {
	v := .0
	sincPhase := srcPhase - dstPhase
	for i := 0; i < sigmaTotal; i++ {
		d := i - sigmaWidth
		v += buf.shapes[i].get(srcPhase) * sinc(sincPhase+float64(d))
	}
	return v
}

func (buf *shapeBuffer) rotate(s shape) {
	if len(buf.shapes) == 0 {
		buf.shapes = make([]shape, sigmaTotal)
		for i := range buf.shapes {
			buf.shapes[i] = s
		}
		log.Print("debug: shapeBuffer is initialized")
		return
	}
	buf.shapes = append(buf.shapes[1:], s)
}

func (buf *shapeBuffer) freq() float64 {
	return buf.shapes[sigmaWidth].freq
}

// stretch は、指定したピッチ係数 pitchCoef、速度係数 speedCoef で再生した波形を返します。
// pitchCoef、speedCoef ともに 1 のとき、オリジナルと同じ波形となります。
func stretch(input chan shape, pitchCoef, speedCoef float64) chan float64 {
	buf := &shapeBuffer{}
	out := make(chan float64, 4096)
	go func() {
		log.Print("debug: stretch goroutine is started")
		srcPhase := .0
		dstPhase := .0
		for s := range input {
			buf.rotate(s)
			freq := buf.freq()
			srcPhaseStep := freq * pitchCoef
			dstPhaseStep := freq * speedCoef
			for ; dstPhase < 1.0; dstPhase += dstPhaseStep {
				out <- buf.get(srcPhase, dstPhase)
				srcPhase += srcPhaseStep
				for 1.0 <= srcPhase {
					srcPhase -= 1.0
				}
			}
			for 1.0 <= dstPhase {
				dstPhase -= 1.0
			}
		}
		close(out)
	}()
	return out
}
