package voispire

import (
	"log"
	"math"

	"github.com/but80/voispire/internal/buffer"
	"github.com/but80/voispire/internal/wav"
	"github.com/but80/voispire/internal/world"
	"github.com/gordonklaus/portaudio"
	"github.com/pkg/errors"
	"github.com/xlab/closer"
)

func join(input chan buffer.Shape) chan float64 {
	out := make(chan float64, 4096)
	go func() {
		for s := range input {
			for _, v := range s.Data() {
				out <- v
			}
		}
		close(out)
	}()
	return out
}

func render(in chan float64) (chan struct{}, error) {
	portaudio.Initialize()
	closer.Bind(func() {
		portaudio.Terminate()
	})

	hostapi, err := portaudio.DefaultHostApi()
	if err != nil {
		return nil, err
	}
	log.Printf("info: Audio device: %s\n", hostapi.DefaultOutputDevice.Name)
	params := portaudio.HighLatencyParameters(nil, hostapi.DefaultOutputDevice)
	endCh := make(chan struct{})
	stream, err := portaudio.OpenStream(params, func(out [][]float32) {
		for i := range out[0] {
			select {
			case v, ok := <-in:
				if !ok {
					if endCh != nil {
						close(endCh)
						endCh = nil
					}
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
		return nil, err
	}
	log.Printf("info: Sample rate: %f\n", stream.Info().SampleRate)
	log.Printf("info: Output latency: %s\n", stream.Info().OutputLatency.String())

	if err := stream.Start(); err != nil {
		return nil, err
	}
	return endCh, nil
}

const (
	framePeriod = .005
)

// Demo は、デモ実装です。
func Demo(transpose float64, infile, outfile string) error {
	src, fs, err := wav.Load(infile)
	if err != nil {
		return errors.Wrap(err, "音声ファイルの読み込みに失敗しました")
	}

	log.Print("info: 基本周波数を推定中...")
	f0, spectro := world.Harvest(src, fs, framePeriod)
	_ = spectro

	log.Print("info: 変換中...")
	pitchCoef := math.Pow(2.0, transpose/12.0)
	ch1 := splitShapes(src, f0, float64(fs))
	ch2 := stretch(ch1, pitchCoef, 1.0)
	ch3 := shiftFormant(ch2, .0)
	outCh := join(ch3)

	if outfile == "" {
		endCh, err := render(outCh)
		if err != nil {
			return errors.Wrap(err, "出力ストリームのオープンに失敗しました")
		}
		<-endCh
	} else {
		log.Print("info: 保存中...")
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
		wav.Save(outfile, fs, result)
		log.Print("info: 完了")
	}
	return nil
}
