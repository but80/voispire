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

const (
	f0Floor = 71.0
	f0Ceil  = 800.0
)

// Options は、 Start 関数のオプションです。
type Options struct {
	Formant         float64
	Transpose       float64
	FramePeriodMsec float64
	Rate            int
	InDevID         int
	OutDevID        int
	InFile          string
	OutFile         string
}

// Start は、音声変換を開始します。
func Start(o Options) error {
	var f0 []float64
	if o.Transpose != 0 {
		log.Print("info: 基本周波数を推定中...")

		src, fs, err := wav.Load(o.InFile)
		if err != nil {
			return xerrors.Errorf("音声ファイルの読み込みに失敗しました: %w", err)
		}
		log.Printf("debug: IN: %d samples, fs=%d", len(src), fs)

		f0, _ = world.Harvest(src, fs, o.FramePeriodMsec, f0Floor, f0Ceil)
	}

	var params portaudio.StreamParameters
	if o.InFile == "" || o.OutFile == "" {
		portaudio.Initialize()
		closer.Bind(func() {
			portaudio.Terminate()
		})
		hostapi, err := portaudio.DefaultHostApi()
		if err != nil {
			return xerrors.Errorf("オーディオデバイスのオープンに失敗しました: %w", err)
		}

		ins, outs, err := getDevices()
		if err != nil {
			return xerrors.Errorf("オーディオデバイス情報の取得に失敗しました: %w", err)
		}

		var inDev *portaudio.DeviceInfo
		if o.InFile == "" {
			inDev = hostapi.DefaultInputDevice
			if 1 <= o.InDevID && o.InDevID <= len(ins) {
				inDev = ins[o.InDevID-1]
			}
			log.Printf("info: Input device: %s\n", inDev.Name)
		}

		var outDev *portaudio.DeviceInfo
		if o.OutFile == "" {
			outDev = hostapi.DefaultOutputDevice
			if 1 <= o.OutDevID && o.OutDevID <= len(outs) {
				outDev = outs[o.OutDevID-1]
			}
			log.Printf("info: Output device: %s\n", outDev.Name)
		}

		params = portaudio.LowLatencyParameters(inDev, outDev)
	}

	var input *buffer.WaveSource
	var paInput *buffer.WaveSource
	var fs int

	if o.InFile == "" {
		paInput = buffer.NewWaveSource()
		input = paInput
	} else {
		var err error
		input, fs, err = wav.NewWavFileSource(o.InFile)
		if err != nil {
			return xerrors.Errorf("音声ファイルのオープンに失敗しました: %w", err)
		}
	}

	fsOut := fs
	if 0 < o.Rate {
		fsOut = o.Rate
	}

	pitchCoef := math.Pow(2.0, o.Transpose/12.0)
	formantCoef := math.Pow(2.0, (o.Formant-o.Transpose)/12.0)

	mod1 := formant.NewCepstralShifter(input, fs, 1024, formantCoef)
	var mod2 *f0Splitter
	var mod3 *stretcher
	var lastmod interface{ Start() }
	var outCh <-chan float64
	if o.Transpose == 0 {
		log.Print("info: フォルマントシフタのみを使用します")
		outCh = mod1.Output()
		lastmod = mod1
	} else {
		log.Print("info: フォルマントシフタとストレッチャを使用します")
		mod2 = newF0Splitter(f0, float64(fs), o.FramePeriodMsec)
		mod3 = newStretcher(pitchCoef, 1.0, float64(fsOut)/float64(fs))
		mod2.input = mod1.Output()
		mod3.input = mod2.output
		outCh = join(mod3.output)
		mod1.Start()
		mod2.Start()
		lastmod = mod3
	}

	if o.OutFile == "" {
		endCh, stream, err := render(params, paInput, outCh)
		if err != nil {
			return xerrors.Errorf("出力ストリームのオープンに失敗しました: %w", err)
		}
		if mod3 != nil {
			mod3.resampleCoef = float64(stream.Info().SampleRate) / float64(fs)
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
		wav.Save(o.OutFile, fsOut, result)
		log.Print("info: 完了")
	}
	return nil
}
