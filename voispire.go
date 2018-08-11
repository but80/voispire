package voispire

import (
	"log"
	"math"

	"github.com/but80/voispire/internal/world"
	"github.com/pkg/errors"
)

const (
	framePeriod = .005
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

	log.Print("info: 変換中...")
	splittedCh := splitShapes(src, f0, float64(fs))
	pitchCoef := math.Pow(2.0, float64(transpose)/12.0)
	outCh := stretch(splittedCh, pitchCoef, 1.0)
	result := make([]float64, len(src))
	i := 0
	for i < len(result) {
		v, ok := <-outCh
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
