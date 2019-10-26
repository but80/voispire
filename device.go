package voispire

import (
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/but80/voispire/internal/buffer"
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

	bufferUnderrunAt := time.Unix(0, 0)
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
		if i < n && endCh != nil && time.Second <= time.Since(bufferUnderrunAt) {
			log.Printf("warn: buffer underrun")
			bufferUnderrunAt = time.Now()
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
