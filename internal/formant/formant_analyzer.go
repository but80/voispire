// +build analyzer

package formant

import (
	"bytes"
	"image"
	"image/color"
	"log"
	"math"
	"math/cmplx"
	"sync"

	"github.com/but80/simplevid-go"
	"gonum.org/v1/gonum/fourier"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

const (
	analyzerFilename     = "analyzer.mp4"
	analyzerFrameLimit   = 20
	analyzerSlowPlayRate = 4
)

var (
	analyzerVidOpts = simplevid.EncoderOptions{
		Width:   960,
		Height:  540,
		BitRate: 8 * 1024 * 1024,
		GOPSize: 30,
		FPS:     30,
	}
	analyzerWait    = make(chan struct{})
	analyzerImageCh = make(chan image.Image, 100)
)

func init() {
	// plot.DefaultFont = "Times-Bold"
}

func toTimePlotLine(data []float64, fs int, c color.Color) *plotter.Line {
	xys := make(plotter.XYs, len(data))
	xr := 1 / float64(fs)
	for i, v := range data {
		xys[i].X = float64(i) * xr
		xys[i].Y = v
	}
	line, _ := plotter.NewLine(xys)
	line.LineStyle.Color = c
	line.LineStyle.Width = 1
	line.LineStyle.Dashes = nil
	line.LineStyle.DashOffs = 0
	return line
}

func toSpecPlotLine(data []float64, fs int, c color.Color) *plotter.Line {
	data = data[1:]
	xys := make(plotter.XYs, len(data))
	xr := (float64(fs) / 2) / float64(len(data))
	for i, v := range data {
		xys[i].X = float64(i+1) * xr
		xys[i].Y = -1e100
		if 0 < v {
			xys[i].Y = 20 * math.Log10(v)
		}
	}
	line, _ := plotter.NewLine(xys)
	line.LineStyle.Color = c
	line.LineStyle.Width = 1
	line.LineStyle.Dashes = nil
	line.LineStyle.DashOffs = 0
	return line
}

func toImage(plots [][]*plot.Plot) (image.Image, error) {
	w := vg.Length(analyzerVidOpts.Width) * vg.Inch / 96
	h := vg.Length(analyzerVidOpts.Height) * vg.Inch / 96
	img := vgimg.New(w, h)
	dc := draw.New(img)

	rows := len(plots)
	cols := len(plots[0])
	pad := h / 20
	tiles := draw.Tiles{
		Rows:      rows,
		Cols:      cols,
		PadTop:    pad,
		PadBottom: pad,
		PadRight:  pad,
		PadLeft:   pad,
		PadX:      pad,
		PadY:      pad,
	}

	canvases := plot.Align(plots, tiles, dc)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			if plots[i][j] != nil {
				plots[i][j].Draw(canvases[i][j])
			}
		}
	}

	tiff := vgimg.TiffCanvas{Canvas: img}
	buf := bytes.NewBuffer(nil)
	if _, err := tiff.WriteTo(buf); err != nil {
		return nil, err
	}
	result, _, err := image.Decode(buf)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func cmplxAbs(data []complex128) []float64 {
	result := make([]float64, len(data))
	for i, v := range data {
		result[i] = cmplx.Abs(v)
	}
	return result
}

var analyzerFrameCounter = 0
var analyzerFrameOnce sync.Once

func analyzerFrame(data *analyzerData) {
	analyzerFrameOnce.Do(func() {
		analyzerVidOpts.FPS = int(float64(data.fs)/(float64(data.fftWidth)/2)/float64(analyzerSlowPlayRate) + .5)
		encoder := simplevid.NewImageEncoder(analyzerVidOpts, analyzerImageCh)
		log.Printf("debug: video option: %#v", analyzerVidOpts)
		go func() {
			if err := encoder.EncodeToFile(analyzerFilename); err != nil {
				panic(err)
			}
			close(analyzerWait)
		}()
	})
	if 0 < analyzerFrameLimit && analyzerFrameLimit <= analyzerFrameCounter {
		return
	}
	log.Printf("debug: FFT process frame %d", analyzerFrameCounter)

	pt, err := plot.New()
	if err != nil {
		panic(err)
	}
	wave1 := fourier.NewFFT((len(data.spec1)-1)*2).Sequence(nil, data.spec1)
	pt.Add(toTimePlotLine(wave1, data.fs, color.RGBA{R: 0, G: 192, B: 255, A: 255}))
	pt.Add(toTimePlotLine(data.wave0, data.fs, color.RGBA{R: 255, G: 112, B: 0, A: 255}))
	pt.Title.Text = "Time Domain"
	pt.X.Label.Text = "Time [s]"
	pt.Y.Label.Text = "Amplitude"
	pt.Y.Min = -1
	pt.Y.Max = 1

	pf, err := plot.New()
	if err != nil {
		panic(err)
	}
	pf.Add(toSpecPlotLine(cmplxAbs(data.spec1), data.fs, color.RGBA{R: 0, G: 192, B: 255, A: 255}))
	pf.Add(toSpecPlotLine(cmplxAbs(data.spec0), data.fs, color.RGBA{R: 255, G: 112, B: 0, A: 255}))
	pf.Add(toSpecPlotLine(data.envelope, data.fs, color.RGBA{R: 255, G: 0, B: 0, A: 255}))
	pf.Title.Text = "Frequency Domain"
	// pf.X.Scale = plot.LogScale{}
	// pf.X.Tick.Marker = plot.LogTicks{}
	pf.X.Label.Text = "Frequency [Hz]"
	pf.Y.Label.Text = "Amplitude [dB]"
	pf.Y.Min = -100
	pf.Y.Max = 0

	img, err := toImage([][]*plot.Plot{
		{pt},
		{pf},
	})
	if err != nil {
		panic(err)
	}
	analyzerImageCh <- img
	analyzerFrameCounter++
}

func analyzerFinish() {
	close(analyzerImageCh)
	// log.Printf("debug: waiting video encoder finishes")
	<-analyzerWait
}
