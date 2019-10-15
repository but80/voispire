// +build analyzer

package voispire

import (
	"bytes"
	"image"
	"image/color"
	"log"
	"math"
	"math/cmplx"

	"github.com/but80/simplevid-go"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

const filename = "analyzer.mp4"
const frameLenLimit = 60

var vidEncoder simplevid.Encoder
var imageCh chan image.Image
var vidOpts = simplevid.EncoderOptions{
	Width:   960,
	Height:  540,
	BitRate: 4 * 1024 * 1024,
	GOPSize: 10,
	FPS:     30,
}

func init() {
	wait := make(chan struct{})
	onFormantFFTProcess = onFormantFFTProcessImpl
	onFormantFFTFinish = func(s *formantShifter) {
		close(imageCh)
		// log.Printf("debug: waiting video encoder finishes")
		<-wait
	}
	vidEncoder = simplevid.NewCustomEncoder(vidOpts, onFrame)
	imageCh = make(chan image.Image, 100)
	go func() {
		if err := vidEncoder.EncodeToFile(filename); err != nil {
			panic(err)
		}
		close(wait)
	}()
}

func toPlotLine(data []float64, fs int, c color.Color) *plotter.Line {
	xys := make(plotter.XYs, len(data))
	for i, v := range data {
		xys[i].X = float64(i) * float64(fs) / 2 / float64(len(data))
		xys[i].Y = -96
		if 0 < v {
			xys[i].Y = 20 * math.Log10(v/127)
		}
	}
	line, _ := plotter.NewLine(xys)
	line.LineStyle.Color = c
	line.LineStyle.Width = 1
	line.LineStyle.Dashes = nil
	line.LineStyle.DashOffs = 0
	return line
}

func toImage(p *plot.Plot) (image.Image, error) {
	w := vg.Length(vidOpts.Width) * vg.Inch / 96
	h := vg.Length(vidOpts.Height) * vg.Inch / 96
	writer, err := p.WriterTo(w, h, "tiff")
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(nil)
	_, err = writer.WriteTo(buf)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(buf)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func complexToFloatSlice(data []complex128) []float64 {
	result := make([]float64, len(data))
	for i, v := range data {
		result[i] = cmplx.Abs(v)
	}
	return result
}

var fftProcessFrame = 0

func onFormantFFTProcessImpl(s *formantShifter, wave0, wave1 []float64, spec0, spec1 []complex128) {
	if frameLenLimit <= fftProcessFrame {
		return
	}
	log.Printf("debug: FFT process frame %d", fftProcessFrame)

	p, err := plot.New()
	if err != nil {
		panic(err)
	}

	p.Add(toPlotLine(complexToFloatSlice(spec0), s.fs, color.RGBA{R: 255, G: 128, B: 128, A: 255}))
	p.Add(toPlotLine(s.envBuf, s.fs, color.RGBA{R: 255, G: 0, B: 0, A: 255}))
	p.Add(toPlotLine(complexToFloatSlice(spec1), s.fs, color.RGBA{R: 0, G: 96, B: 255, A: 255}))

	p.Title.Text = "voispire"
	p.X.Label.Text = "Freq."
	p.Y.Label.Text = "Amp."
	p.Y.Min = -90
	p.Y.Max = 0

	img, err := toImage(p)
	if err != nil {
		panic(err)
	}
	imageCh <- img
	fftProcessFrame++
}

func onFrame(e simplevid.Encoder) bool {
	img0, ok := <-imageCh
	if !ok {
		return false
	}
	img := img0.(*image.RGBA)
	opts := e.Options()
	for y := 0; y < opts.Height; y++ {
		for x := 0; x < opts.Width; x++ {
			rgba := img.RGBAAt(x, y)
			e.SetRGB(x, y, int(rgba.R), int(rgba.G), int(rgba.B))
		}
	}
	return true
}
