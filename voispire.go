package voispire

import (
	"log"
	"math"
	"time"

	"github.com/but80/voispire/internal/buffer"
	"github.com/but80/voispire/internal/formant"
	"github.com/but80/voispire/internal/wav"
	"github.com/but80/voispire/internal/world"
	"github.com/gordonklaus/portaudio"
	"github.com/xlab/closer"
	"golang.org/x/xerrors"
)

func join(input <-chan buffer.Shape) <-chan float64 {
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

func render(in <-chan float64) (chan struct{}, *portaudio.StreamInfo, error) {
	portaudio.Initialize()
	closer.Bind(func() {
		portaudio.Terminate()
	})

	hostapi, err := portaudio.DefaultHostApi()
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}
	log.Printf("info: Sample rate: %f\n", stream.Info().SampleRate)
	log.Printf("info: Output latency: %s\n", stream.Info().OutputLatency.String())

	if err := stream.Start(); err != nil {
		return nil, nil, err
	}
	return endCh, stream.Info(), nil
}

const (
	f0Floor = 71.0
	f0Ceil  = 800.0
)

// Demo は、デモ実装です。
func Demo(transpose, formantShift, framePeriod float64, rate int, infile, outfile string) error {
	var f0 []float64
	if transpose != 0 {
		log.Print("info: 基本周波数を推定中...")

		src, fs, err := wav.Load(infile)
		if err != nil {
			return xerrors.Errorf("音声ファイルの読み込みに失敗しました: %w", err)
		}
		log.Printf("debug: IN: %d samples, fs=%d", len(src), fs)

		f0, _ = world.Harvest(src, fs, framePeriod, f0Floor, f0Ceil)
	}

	input, fs, err := wav.NewWavFileSource(infile)
	if err != nil {
		return xerrors.Errorf("音声ファイルのオープンに失敗しました: %w", err)
	}

	fsOut := fs
	if 0 < rate {
		fsOut = rate
	}

	pitchCoef := math.Pow(2.0, transpose/12.0)
	formantCoef := math.Pow(2.0, (formantShift-transpose)/12.0)

	mod1 := formant.NewCepstralShifter(input, fs, 1024, formantCoef)
	var mod2 *f0Splitter
	var mod3 *stretcher
	var lastmod interface{ Start() }
	var outCh <-chan float64
	if transpose == 0 {
		log.Print("info: フォルマントシフタのみを使用します")
		outCh = mod1.Output()
		lastmod = mod1
	} else {
		log.Print("info: フォルマントシフタとストレッチャを使用します")
		mod2 = newF0Splitter(f0, float64(fs), framePeriod)
		mod3 = newStretcher(pitchCoef, 1.0, float64(fsOut)/float64(fs))
		mod2.input = mod1.Output()
		mod3.input = mod2.output
		outCh = join(mod3.output)
		mod1.Start()
		mod2.Start()
		lastmod = mod3
	}

	if outfile == "" {
		endCh, info, err := render(outCh)
		if err != nil {
			return xerrors.Errorf("出力ストリームのオープンに失敗しました: %w", err)
		}
		if mod3 != nil {
			mod3.resampleCoef = float64(info.SampleRate) / float64(fs)
		}
		log.Print("info: 変換を開始しました")
		lastmod.Start()
		<-endCh
		time.Sleep(time.Second)
	} else {
		lastmod.Start()
		result := make([]float64, 0)
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
