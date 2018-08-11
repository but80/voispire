package voispire

import (
	"log"
	"math"

	"github.com/but80/voispire/internal/world"
	"github.com/pkg/errors"
)

const (
	framePeriod = .005
	minFreq     = 1.0
)

// Demo は、デモ実装です。
func Demo(transpose int, infile, outfile string) error {
	src, fs, err := loadWav(infile)
	if err != nil {
		return errors.Wrap(err, "音声ファイルの読み込みに失敗しました")
	}

	log.Print("info: 基本周波数を推定中...")
	f0, spectro := world.Harvest(src, fs, framePeriod)
	_ = spectro

	log.Print("info: 解析中...")
	t := .0
	dt := 1.0 / float64(fs)
	iBegin := 0
	phase := .0
	lastFreq := 440.0
	sh := &shifter{}
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
			sh.addShape(src[iBegin:i])
			iBegin = i
		}
		lastFreq = freq
		t += dt
	}

	log.Print("info: 変換中...")
	pitchCoef := math.Pow(2.0, float64(transpose)/12.0)

	result := make([]float64, len(src))
	ch := sh.play(pitchCoef, 1.0)
	i := 0
	for i < len(result) {
		v, ok := <-ch
		if !ok {
			break
		}
		result[i] = v
		i++
	}

	log.Print("info: 保存中...")
	saveWav(outfile, fs, result)
	log.Print("info: 完了")
	return nil
}
