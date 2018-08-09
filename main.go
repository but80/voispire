package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"reflect"

	"github.com/but80/voispire/internal/world"
	"github.com/go-audio/wav"
	dsp "github.com/mjibson/go-dsp/wav"
	"github.com/pkg/errors"
)

const (
	framePeriod = .005
	minFreq     = 1.0
)

func loadWav(filename string) ([]float64, int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()
	w, err := dsp.New(file)
	if err != nil {
		return nil, 0, err
	}
	samples, err := w.ReadSamples(w.Samples)
	if err != nil {
		return nil, 0, err
	}
	result := make([]float64, w.Samples)
	switch s := samples.(type) {
	case []int16:
		for i, v := range s {
			result[i] = float64(v) / 32767.0
		}
	default:
		return nil, 0, fmt.Errorf("Unsupported sample size: %s", reflect.TypeOf(samples))
	}
	return result, int(w.SampleRate), nil
}

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

type Shifter struct {
	Fs       int
	totalLen int
	shapes   []shape
}

func (shifter *Shifter) Record(data []float64) {
	shifter.totalLen += len(data)
	shifter.shapes = append(shifter.shapes, shape{
		flen: float64(len(data)),
		data: data,
	})
}

func (shifter *Shifter) Play(pitchCoef, speedCoef float64) []float64 {
	result := make([]float64, 0, shifter.totalLen)
	phase := .0
	i := .0
	for iShape := sigmaWidth; iShape < len(shifter.shapes)-sigmaWidth; iShape++ {
		n := shifter.shapes[iShape].flen
		dphase := 1.0 / n * pitchCoef
		di := speedCoef / n
		for ; i < 1.0; i += di {
			v := .0
			sincPhase := phase - i
			for j := -sigmaWidth; j <= sigmaWidth; j++ {
				v += shifter.shapes[iShape+j].get(phase) * sinc(sincPhase+float64(j))
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

func foo(infile string) error {
	src, fs, err := loadWav(infile)
	if err != nil {
		return errors.Wrap(err, "音声ファイルの読み込みに失敗しました")
	}

	log.Print("info: 基本周波数を推定中...")
	f0 := world.Harvest(src, fs, framePeriod)

	log.Print("info: 解析中...")
	t := .0
	dt := 1.0 / float64(fs)
	iBegin := 0
	phase := .0
	lastFreq := 440.0
	sh := &Shifter{Fs: fs}
	for i := range src {
		j := int(math.Floor(t / float64(framePeriod)))
		freq := lastFreq
		if j < len(f0) && minFreq <= f0[j] {
			freq = f0[j]
		}
		phase += freq * dt
		if 1.0 <= phase {
			for 1.0 <= phase {
				phase -= 1.0
			}
			sh.Record(src[iBegin:i])
			iBegin = i
		}
		lastFreq = freq
		t += dt
	}

	log.Print("info: 変換中...")
	out := sh.Play(1.3, 1.0)

	log.Print("info: 保存中...")
	saveWav("test.out.wav", fs, out)
	log.Print("info: 完了")
	return nil
}

func saveWav(filename string, fs int, data []float64) error {
	out, err := os.Create(filename)
	if err != nil {
		return errors.Wrap(err, "出力音声ファイルのオープンに失敗しました")
	}
	enc := wav.NewEncoder(out, fs, 16, 1, 1)
	for _, v := range data {
		if err := enc.WriteFrame(uint16(v * 32767)); err != nil {
			return errors.Wrap(err, "出力音声ファイルの書き込みに失敗しました")
		}
	}
	if err := enc.Close(); err != nil {
		return errors.Wrap(err, "出力音声ファイルのクローズに失敗しました")
	}
	return out.Close()
}

func main() {
	if err := foo("test.wav"); err != nil {
		panic(err)
	}
}
