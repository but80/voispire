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
	if err := start(o); err != nil {
		return err
	}
	closer.Close()
	closer.Hold()
	return nil
}

func start(o Options) error {
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

	// 入力ファイルのみ指定時
	if o.InFile != "" && o.OutFile == "" {
		o.OutDevID = -1 // デフォルト出力デバイスを選択
	}

	var params portaudio.StreamParameters
	if o.InDevID != 0 || o.OutDevID != 0 {
		var err error
		params, err = initAudio(o)
		if err != nil {
			return err
		}
	}

	waitOutput := make(chan struct{}, 1)
	closer.Bind(func() {
		log.Print("debug: binded <-waitOutput")
		<-waitOutput
		log.Print("debug: binded <-waitOutput finished")
	})

	var input *buffer.WaveSource
	var audioInput *buffer.WaveSource
	var fs int

	if o.InFile == "" {
		audioInput = buffer.NewWaveSource()
		input = audioInput
		// FIXME: 入力デバイスの周波数レートをfsに設定
		fs = 44100
	} else {
		var err error
		input, fs, err = wav.NewWavFileSource(o.InFile)
		if err != nil {
			return xerrors.Errorf("音声ファイルのオープンに失敗しました: %w", err)
		}
	}
	closer.Bind(func() {
		log.Print("debug: closing input")
		input.Close()
	})

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

	var fileOutCh chan<- []float64
	var fileOutWait <-chan struct{}
	waitFileOut := func() {}
	if o.OutFile != "" {
		var err error
		fileOutCh, fileOutWait, err = wav.StartSave(o.OutFile, fsOut)
		if err != nil {
			return xerrors.Errorf("出力ファイルのオープンに失敗しました: %w", err)
		}
		log.Print("info: ファイル出力を開始しました")
		waitFileOut = func() {
			log.Print("debug: close(fileOutCh)")
			close(fileOutCh)
			log.Print("debug: <-fileOutWait")
			<-fileOutWait
			log.Print("debug: <-fileOutWait finished")
			log.Print("info: ファイル出力完了")
		}
	}

	if o.InDevID != 0 || o.OutDevID != 0 {
		waitInput, stream, err := render(params, audioInput, outCh, fileOutCh)
		if err != nil {
			return xerrors.Errorf("出力ストリームのオープンに失敗しました: %w", err)
		}
		if mod3 != nil {
			mod3.resampleCoef = float64(stream.Info().SampleRate) / float64(fs)
		}
		log.Print("info: 変換を開始しました")
		go func() {
			lastmod.Start()
			log.Print("debug: <-waitInput")
			<-waitInput
			log.Print("debug: <-waitInput finished")
			time.Sleep(time.Second)
			waitFileOut()
			log.Print("debug: close(waitOutput)")
			close(waitOutput)
		}()
	} else {
		lastmod.Start()
		result := make([]float64, 0)
		log.Print("info: 変換中...")
		go func() {
			// TODO: 一定サイズごとに出力
			for {
				v, ok := <-outCh
				if !ok {
					break
				}
				result = append(result, v)
			}
			fileOutCh <- result
			log.Printf("debug: OUT: %d samples, fs=%d", len(result), fsOut)
			waitFileOut()
			log.Print("debug: close(waitOutput)")
			close(waitOutput)
		}()
	}
	log.Print("debug: <-waitOutput")
	<-waitOutput
	log.Print("debug: <-waitOutput finished")
	return nil
}
