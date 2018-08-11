package voispire

import (
	"log"
	"math"
	"time"

	"github.com/but80/voispire/internal/world"
	"github.com/gordonklaus/portaudio"
	"github.com/pkg/errors"
	"github.com/xlab/closer"
)

func render(in chan float64) error {
	portaudio.Initialize()
	closer.Bind(func() {
		portaudio.Terminate()
	})

	hostapi, err := portaudio.DefaultHostApi()
	if err != nil {
		return err
	}
	log.Printf("info: Audio device: %s\n", hostapi.DefaultOutputDevice.Name)
	params := portaudio.HighLatencyParameters(nil, hostapi.DefaultOutputDevice)
	stream, err := portaudio.OpenStream(params, func(out [][]float32) {
		for i := range out[0] {
			select {
			case v, ok := <-in:
				if !ok {
					break
				}
				out[0][i] = float32(v)
				out[1][i] = float32(v)
			default:
				break
			}
		}
	})
	if err != nil {
		return err
	}
	log.Printf("info: Sample rate: %f\n", stream.Info().SampleRate)
	log.Printf("info: Output latency: %s\n", stream.Info().OutputLatency.String())

	if err := stream.Start(); err != nil {
		return err
	}
	return nil
}

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
	if err := render(outCh); err != nil {
		return errors.Wrap(err, "出力ストリームのオープンに失敗しました")
	}

	for {
		time.Sleep(time.Second)
	}

	// log.Print("info: 保存中...")
	// saveWav(outfile, fs, result)
	// log.Print("info: 完了")
	return nil
}
