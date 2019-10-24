package voispire

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/but80/voispire/internal/buffer"
	"github.com/but80/voispire/internal/formant"
	"github.com/but80/voispire/internal/wav"
	"github.com/but80/voispire/internal/world"
	"github.com/gordonklaus/portaudio"
	"github.com/saintfish/chardet"
	"github.com/xlab/closer"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
	"golang.org/x/xerrors"
)

func autoToUTF8(s string) string {
	r, err := chardet.NewTextDetector().DetectBest([]byte(s))
	if err != nil {
		return s
	}

	var e encoding.Encoding
	switch r.Charset {
	case "UTF-8", "ISO-8859-1", "ISO-8859-9":
		return s
	case "EUCJP":
		e = japanese.EUCJP
	case "ISO-2022-JP":
		e = japanese.ISO2022JP
	case "Shift_JIS", "windows-1252", "windows-1254":
		e = japanese.ShiftJIS
	default:
		log.Printf("info: unsupported charset = %s", r.Charset)
		return s
	}

	t, err := ioutil.ReadAll(transform.NewReader(strings.NewReader(s), e.NewDecoder()))
	if err != nil {
		return s
	}
	return string(t)
}

func lessDevice(devs []*portaudio.DeviceInfo) func(i, j int) bool {
	return func(i, j int) bool {
		a := devs[i]
		b := devs[j]
		if a.DefaultLowInputLatency < b.DefaultLowInputLatency {
			return true
		}
		if b.DefaultLowInputLatency < a.DefaultLowInputLatency {
			return false
		}
		if a.DefaultHighInputLatency < b.DefaultHighInputLatency {
			return true
		}
		if b.DefaultHighInputLatency < a.DefaultHighInputLatency {
			return false
		}
		if b.DefaultSampleRate < a.DefaultSampleRate {
			return true
		}
		if a.DefaultSampleRate < b.DefaultSampleRate {
			return false
		}
		if b.MaxInputChannels < a.MaxInputChannels {
			return true
		}
		if a.MaxInputChannels < b.MaxInputChannels {
			return false
		}
		if b.MaxOutputChannels < a.MaxOutputChannels {
			return true
		}
		if a.MaxOutputChannels < b.MaxOutputChannels {
			return false
		}
		return a.Name < b.Name
	}
}

func listDevices() (ins, outs []*portaudio.DeviceInfo, err error) {
	devs, err := portaudio.Devices()
	if err != nil {
		return nil, nil, err
	}

	for _, dev := range devs {
		dev.Name = autoToUTF8(dev.Name)
		if 0 < dev.MaxInputChannels {
			ins = append(ins, dev)
		}
		if 0 < dev.MaxOutputChannels {
			outs = append(outs, dev)
		}
	}
	sort.Slice(ins, lessDevice(ins))
	sort.Slice(outs, lessDevice(outs))
	return
}

// ListDevices は、オーディオデバイスの一覧を表示します。
func ListDevices() error {
	portaudio.Initialize()
	closer.Bind(func() {
		portaudio.Terminate()
	})

	ins, outs, err := listDevices()
	if err != nil {
		return xerrors.Errorf("オーディオデバイス情報の取得に失敗しました: %w", err)
	}
	defaultIn, err := portaudio.DefaultInputDevice()
	if err != nil {
		return xerrors.Errorf("オーディオデバイス情報の取得に失敗しました: %w", err)
	}
	defaultOut, err := portaudio.DefaultOutputDevice()
	if err != nil {
		return xerrors.Errorf("オーディオデバイス情報の取得に失敗しました: %w", err)
	}

	fmt.Println("INPUTS:")
	for i, dev := range ins {
		if dev == defaultIn {
			fmt.Print("* ")
		} else {
			fmt.Print("  ")
		}
		fmt.Printf("[%2d]", i+1)
		fmt.Printf(" %-48s:", dev.Name)
		fmt.Printf(" %s", dev.HostApi.Name)
		fmt.Printf(" %s", dev.DefaultLowInputLatency)
		fmt.Printf(" %vHz", dev.DefaultSampleRate)
		fmt.Println()
	}
	fmt.Println()

	fmt.Println("OUTPUTS:")
	for i, dev := range outs {
		if dev == defaultOut {
			fmt.Print("* ")
		} else {
			fmt.Print("  ")
		}
		fmt.Printf("[%2d]", i+1)
		fmt.Printf(" %-48s:", dev.Name)
		fmt.Printf(" %s", dev.HostApi.Name)
		fmt.Printf(" %s", dev.DefaultLowOutputLatency)
		fmt.Printf(" %vHz", dev.DefaultSampleRate)
		fmt.Println()
	}
	fmt.Println()

	return nil
}

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

func render(params portaudio.StreamParameters, input *buffer.WaveSource, outCh <-chan float64) (chan struct{}, *portaudio.Stream, error) {
	endCh := make(chan struct{})
	onIn := func(in [][]float32) {
		if len(in) == 0 {
			return
		}
		n := len(in[0])
		buf := make([]float64, n)
		for i := 0; i < n; i++ {
			buf[i] = float64(in[0][i])
		}
		input.Append(buf)
	}
	onOut := func(out [][]float32) {
		i := 0
		n := len(out[0])
		for ; i < n; i++ {
			select {
			case v, ok := <-outCh:
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
		for ; i < n; i++ {
			out[0][i] = 0
			out[1][i] = 0
		}
	}

	var onProcess interface{} = onOut
	if input != nil {
		if outCh != nil {
			onProcess = func(in, out [][]float32) {
				onIn(in)
				onOut(out)
			}
		} else {
			onProcess = onIn
		}
	}

	stream, err := portaudio.OpenStream(params, onProcess)
	if err != nil {
		return nil, nil, err
	}
	log.Printf("info: Sample rate: %f\n", stream.Info().SampleRate)
	log.Printf("info: Input latency: %s\n", stream.Info().InputLatency.String())
	log.Printf("info: Output latency: %s\n", stream.Info().OutputLatency.String())

	if err := stream.Start(); err != nil {
		return nil, nil, err
	}
	return endCh, stream, nil
}

const (
	f0Floor = 71.0
	f0Ceil  = 800.0
)

// Start は、音声変換を開始します。
func Start(transpose, formantShift, framePeriod float64, rate, inDevID, outDevID int, infile, outfile string) error {
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

	var params portaudio.StreamParameters
	if infile == "" || outfile == "" {
		portaudio.Initialize()
		closer.Bind(func() {
			portaudio.Terminate()
		})
		hostapi, err := portaudio.DefaultHostApi()
		if err != nil {
			return xerrors.Errorf("オーディオデバイスのオープンに失敗しました: %w", err)
		}

		ins, outs, err := listDevices()
		if err != nil {
			return xerrors.Errorf("オーディオデバイス情報の取得に失敗しました: %w", err)
		}

		var inDev *portaudio.DeviceInfo
		if infile == "" {
			inDev = hostapi.DefaultInputDevice
			inDevID--
			if 0 <= inDevID && inDevID < len(ins) {
				inDev = ins[inDevID]
			}
			log.Printf("info: Input device: %s\n", inDev.Name)
		}

		var outDev *portaudio.DeviceInfo
		if outfile == "" {
			outDev = hostapi.DefaultOutputDevice
			outDevID--
			if 0 <= outDevID && outDevID < len(outs) {
				outDev = outs[outDevID]
			}
			log.Printf("info: Output device: %s\n", outDev.Name)
		}

		params = portaudio.LowLatencyParameters(inDev, outDev)
	}

	var input *buffer.WaveSource
	var paInput *buffer.WaveSource
	var fs int

	if infile == "" {
		paInput = buffer.NewWaveSource()
		input = paInput
	} else {
		var err error
		input, fs, err = wav.NewWavFileSource(infile)
		if err != nil {
			return xerrors.Errorf("音声ファイルのオープンに失敗しました: %w", err)
		}
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
		wav.Save(outfile, fsOut, result)
		log.Print("info: 完了")
	}
	return nil
}
