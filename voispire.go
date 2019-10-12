package voispire

import (
	"log"
	"math"
	"time"

	"github.com/but80/voispire/internal/buffer"
	"github.com/but80/voispire/internal/wav"
	"github.com/but80/voispire/internal/world"
	"github.com/hajimehoshi/oto"
	"github.com/pkg/errors"
	"github.com/xlab/closer"
)

func join(input chan buffer.Shape) chan float64 {
	out := make(chan float64, 4096)
	go func() {
		msg := 0
		for s := range input {
			for _, v := range s.Data() {
				out <- v
				msg++
			}
		}
		log.Printf("debug: join: %d messages", msg)
		close(out)
	}()
	return out
}

func render(rate int, in chan float64) (chan struct{}, error) {
	ctx, err := oto.NewContext(rate, 1, 2, 4096)
	if err != nil {
		return nil, err
	}
	closer.Bind(func() {
		ctx.Close()
	})
	player := ctx.NewPlayer()

	endCh := make(chan struct{})
	go func() {
		var buf = make([]byte, 2)
		for v := range in {
			d := uint16(int16(math.Round(32767.0 * v)))
			buf[0] = byte(d & 255)
			buf[1] = byte(d >> 8)
			_, _ = player.Write(buf)
		}
		close(endCh)
		endCh = nil
	}()
	return endCh, nil
}

const (
	f0Floor = 71.0
	f0Ceil  = 800.0
)

// Demo は、デモ実装です。
func Demo(transpose, formant, framePeriod float64, rate int, infile, outfile string) error {
	src, fs, err := wav.Load(infile)
	if err != nil {
		return errors.Wrap(err, "音声ファイルの読み込みに失敗しました")
	}
	log.Printf("debug: IN: %d samples, fs=%d", len(src), fs)

	var f0 []float64
	if transpose != 0 {
		log.Print("info: 基本周波数を推定中...")
		f0, _ = world.Harvest(src, fs, framePeriod, f0Floor, f0Ceil)
	}

	fsOut := fs
	if 0 < rate {
		fsOut = rate
	}

	pitchCoef := math.Pow(2.0, transpose/12.0)
	formantCoef := math.Pow(2.0, (formant-transpose)/12.0)

	mod1 := newFormantShifter(src, 1024, formantCoef)
	var mod2 *f0Splitter
	var mod3 *stretcher
	var lastmod interface{ Start() }
	var outCh chan float64
	if transpose == 0 {
		log.Print("info: フォルマントシフタのみを使用します")
		outCh = mod1.output
		lastmod = mod1
	} else {
		log.Print("info: フォルマントシフタとストレッチャを使用します")
		mod2 = newF0Splitter(f0, float64(fs), framePeriod)
		mod3 = newStretcher(pitchCoef, 1.0, float64(fsOut)/float64(fs))
		mod2.input = mod1.output
		mod3.input = mod2.output
		outCh = join(mod3.output)
		mod1.Start()
		mod2.Start()
		lastmod = mod3
	}

	if outfile == "" {
		rate := 44100
		endCh, err := render(rate, outCh)
		if err != nil {
			return errors.Wrap(err, "出力ストリームのオープンに失敗しました")
		}
		if mod3 != nil {
			mod3.resampleCoef = float64(rate) / float64(fs)
		}
		log.Print("info: 変換を開始しました")
		lastmod.Start()
		<-endCh
		time.Sleep(time.Second)
	} else {
		lastmod.Start()
		result := make([]float64, 0, len(src))
		log.Print("info: 変換中...")
		for {
			v, ok := <-outCh
			if !ok {
				break
			}
			result = append(result, v)
		}
		log.Printf("debug: OUT: %d samples, fs=%d", len(result), fsOut)
		log.Print("info: 保存中...")
		wav.Save(outfile, fsOut, result)
		log.Print("info: 完了")
	}
	return nil
}
