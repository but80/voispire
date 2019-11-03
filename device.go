package voispire

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/but80/voispire/internal/buffer"
	"github.com/gordonklaus/portaudio"
	"github.com/mattn/go-runewidth"
	"github.com/olekukonko/tablewriter"
	"github.com/saintfish/chardet"
	"github.com/xlab/closer"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
	"golang.org/x/xerrors"
)

func autoToUTF8(s string) string {
	if i := strings.IndexRune(s, 0); 0 <= i {
		s = s[:i]
	}
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

		if a.HostApi.Name < b.HostApi.Name {
			return true
		}
		if b.HostApi.Name < a.HostApi.Name {
			return false
		}

		if a.Name < b.Name {
			return true
		}
		if b.Name < a.Name {
			return false
		}

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

		return false
	}
}

func getDevices() (ins, outs []*portaudio.DeviceInfo, err error) {
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

func deviceRow(dev *portaudio.DeviceInfo) []string {
	name := runewidth.Truncate(dev.Name, 50, "...")
	// name = runewidth.FillRight(name, 50)
	return []string{
		dev.HostApi.Name,
		name,
		dev.DefaultLowInputLatency.String(),
		fmt.Sprintf("%vHz", dev.DefaultSampleRate),
	}
}

func listDevices(caption string, devices []*portaudio.DeviceInfo, defaultDevice *portaudio.DeviceInfo) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.SetHeader([]string{"ID", "DRIVER", caption, "LATENCY", "RATE"})
	table.SetColumnAlignment([]int{
		tablewriter.ALIGN_RIGHT,
		tablewriter.ALIGN_LEFT,
		tablewriter.ALIGN_LEFT,
		tablewriter.ALIGN_RIGHT,
		tablewriter.ALIGN_RIGHT,
	})
	var data [][]string
	for i, dev := range devices {
		id := fmt.Sprintf("%d", i+1)
		if dev == defaultDevice {
			id = "* " + id
		}
		row := append([]string{id}, deviceRow(dev)...)
		data = append(data, row)
	}
	table.AppendBulk(data)
	table.Render()
}

// ListDevices は、オーディオデバイスの一覧を表示します。
func ListDevices() error {
	portaudio.Initialize()
	closer.Bind(func() {
		log.Print("debug: terminating PortAudio")
		portaudio.Terminate()
	})

	ins, outs, err := getDevices()
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
	fmt.Println()
	listDevices("INPUT DEVICE NAME", ins, defaultIn)
	fmt.Println()

	fmt.Println("OUTPUTS:")
	fmt.Println()
	listDevices("OUTPUT DEVICE NAME", outs, defaultOut)
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

func initAudio(o Options) (portaudio.StreamParameters, error) {
	portaudio.Initialize()
	closer.Bind(func() {
		log.Print("debug: terminating PortAudio")
		portaudio.Terminate()
	})
	hostapi, err := portaudio.DefaultHostApi()
	if err != nil {
		return portaudio.StreamParameters{}, xerrors.Errorf("オーディオデバイスのオープンに失敗しました: %w", err)
	}

	ins, outs, err := getDevices()
	if err != nil {
		return portaudio.StreamParameters{}, xerrors.Errorf("オーディオデバイス情報の取得に失敗しました: %w", err)
	}

	var inDev *portaudio.DeviceInfo
	if o.InDevID != 0 {
		inDev = hostapi.DefaultInputDevice
		if 1 <= o.InDevID && o.InDevID <= len(ins) {
			inDev = ins[o.InDevID-1]
		}
		log.Printf("info: Input device: %s\n", inDev.Name)
	}

	var outDev *portaudio.DeviceInfo
	if o.OutDevID != 0 {
		outDev = hostapi.DefaultOutputDevice
		if 1 <= o.OutDevID && o.OutDevID <= len(outs) {
			outDev = outs[o.OutDevID-1]
		}
		log.Printf("info: Output device: %s\n", outDev.Name)
	}

	return portaudio.LowLatencyParameters(inDev, outDev), nil
}

func render(params portaudio.StreamParameters, input *buffer.WaveSource, outCh <-chan float64, fileOutCh chan<- []float64) (<-chan struct{}, *portaudio.Stream, error) {
	waitCh := make(chan struct{})
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
	f64buf := make([]float64, 0)
	onOut := func(out [][]float32) {
		i := 0
		n := len(out[0])
		f64buf = f64buf[:0]
		for ; i < n; i++ {
			select {
			case v, ok := <-outCh:
				if !ok {
					if waitCh != nil {
						close(waitCh)
						waitCh = nil
					}
					break
				}
				out[0][i] = float32(v)
				out[1][i] = float32(v)
				f64buf = append(f64buf, v)
			default:
				break
			}
		}
		if i < n && waitCh != nil && time.Second <= time.Since(bufferUnderrunAt) {
			log.Printf("warn: buffer underrun")
			bufferUnderrunAt = time.Now()
		}
		for ; i < n; i++ {
			out[0][i] = 0
			out[1][i] = 0
		}
		if fileOutCh != nil {
			// FIXME: closeされている可能性、このコールバックからは処理を分離
			fileOutCh <- f64buf
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
	return waitCh, stream, nil
}
