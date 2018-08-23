package buffer

import (
	"log"
	"math"

	"github.com/but80/voispire/internal/smath"
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

// Shape は、1周期分の波形です。
type Shape struct {
	// begin は、波形データの有効部分の開始位置 [サンプル] です。
	begin int
	// size は、波形データの有効部分のサンプル数です。
	size int
	// fsize は、波形データの有効部分のサンプル数を float64 型で保持します。
	fsize float64
	// freq は、波形データのオリジナルの周波数 [1/サンプル] で、1/fsize に一致します。
	freq float64
	// data は、波形データです。各要素は振幅 -1≦v≦1 を表します。
	data []float64
}

// MakeShape は、新しい Shape を作成します。
func MakeShape(data []float64) Shape {
	return MakeShapeTrimmed(data, 0, len(data))
}

// MakeShapeTrimmed は、範囲を指定して新しい Shape を作成します。
func MakeShapeTrimmed(data []float64, begin, end int) Shape {
	size := end - begin
	fsize := float64(size)
	return Shape{
		begin: begin,
		size:  size,
		fsize: fsize,
		freq:  1.0 / fsize,
		data:  data,
	}
}

// Data は、波形データを返します。
func (sh *Shape) Data() []float64 {
	return sh.data[sh.begin : sh.begin+sh.size]
}

// get は、指定した位相 0≦phase＜1 におけるこの波形の振幅を取得します。
func (sh *Shape) get(phase float64) float64 {
	return sh.data[sh.begin+int(math.Floor(phase*sh.fsize))]
}

// getLinear は、指定した位相 0≦phase＜1 におけるこの波形の線形補間された振幅を取得します。
// TODO: 0, 1 付近で前後の波形のサンプルを参照
func (sh *Shape) getLinear(phase float64) float64 {
	i, f := math.Modf(phase * sh.fsize)
	i0 := sh.begin + int(i)
	i1 := i0 + 1
	if len(sh.data) <= i1 {
		return sh.data[i0]
	}
	return sh.data[i0]*(1.0-f) + sh.data[i1]*f
}

// getLagrange は、指定した位相 0≦phase＜1 におけるこの波形のラグランジュ補間された振幅を取得します。
func (sh *Shape) getLagrange(phase float64) float64 {
	i, f := math.Modf(phase * sh.fsize)
	i1 := sh.begin + int(i)
	if i1 == 0 {
		return sh.data[0]*(1.0-f) + sh.data[1]*f
	}
	i0 := i1 - 1
	i2 := i1 + 1
	if len(sh.data) <= i2 {
		return sh.data[i1]
	}
	y0 := sh.data[i0]
	y1 := sh.data[i1]
	y2 := sh.data[i2]
	return 0.5*f*((f-1.0)*y0+(f+1.0)*y2) - (f-1.0)*(f+1.0)*y1
}

// ShapeHistory は、波形の履歴を一定数保持する領域です。
type ShapeHistory struct {
	shapes []Shape
}

// Get は、現在バッファの中心にある波形の位相 0≦srcPhase＜1 における振幅を返します。
// 返される振幅は、前後各 sigmaWidth 個の波形の同じ位相における振幅を用いて補間された値で、
// 0≦dstPhase＜1 はその係数に用いられます（小さいほど前方、大きいほど後方の波形に比重が置かれます）。
func (buf *ShapeHistory) Get(srcPhase, dstPhase float64) float64 {
	v := .0
	sincPhase := srcPhase - dstPhase
	for i := 0; i < sigmaTotal; i++ {
		d := i - sigmaWidth
		v += buf.shapes[i].getLagrange(srcPhase) * smath.SincNormalized(sincPhase+float64(d))
	}
	return v
}

// Rotate は、波形の履歴に一つの Shape を追記し、一定数を超えた古い Shape を履歴から削除します。
// TODO: リングバッファ化
func (buf *ShapeHistory) Rotate(s Shape) {
	if len(buf.shapes) == 0 {
		buf.shapes = make([]Shape, sigmaTotal)
		for i := range buf.shapes {
			buf.shapes[i] = s
		}
		log.Print("debug: ShapeHistory is initialized")
		return
	}
	buf.shapes = append(buf.shapes[1:], s)
}

// Freq は、現在バッファの中心にある波形のオリジナルの周波数を返します。
func (buf *ShapeHistory) Freq() float64 {
	return buf.shapes[sigmaWidth].freq
}
