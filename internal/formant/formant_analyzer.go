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
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

const (
	analyzerFilename     = "analyzer.mp4"
	analyzerFrameLimit   = 60
	analyzerSlowPlayRate = 4
)

var (
	analyzerVidOpts = simplevid.EncoderOptions{
		Width:   960,
		Height:  540,
		BitRate: 8 * 1024 * 1024,
		GOPSize: 10,
		FPS:     30,
	}
	analyzerWait    = make(chan struct{})
	analyzerImageCh = make(chan image.Image, 100)
)

func toPlotLine(data []float64, fs int, c color.Color) *plotter.Line {
	data = data[1:]
	xys := make(plotter.XYs, len(data))
	xr := (float64(fs) / 2) / float64(len(data))
	for i, v := range data {
		xys[i].X = float64(i+1) * xr
		xys[i].Y = -96
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

func toImage(p *plot.Plot) (image.Image, error) {
	w := vg.Length(analyzerVidOpts.Width) * vg.Inch / 96
	h := vg.Length(analyzerVidOpts.Height) * vg.Inch / 96
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

	p, err := plot.New()
	if err != nil {
		panic(err)
	}

	p.Add(toPlotLine(cmplxAbs(data.spec1), data.fs, color.RGBA{R: 0, G: 192, B: 255, A: 255}))
	p.Add(toPlotLine(cmplxAbs(data.spec0), data.fs, color.RGBA{R: 255, G: 128, B: 0, A: 255}))
	p.Add(toPlotLine(data.envelope, data.fs, color.RGBA{R: 255, G: 0, B: 0, A: 255}))

	p.Title.Text = "voispire"
	// p.X.Scale = plot.LogScale{}
	// p.X.Tick.Marker = plot.LogTicks{}
	p.X.Label.Text = "Freq."
	p.Y.Label.Text = "Amp."
	p.Y.Min = -90
	p.Y.Max = 0

	img, err := toImage(p)
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
