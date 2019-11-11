// +build analyzer

package formant

import (
	"bytes"
	"image"
	"image/color"
	"math"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

const (
	plotDrawResize = 2
)

type plotSeries struct {
	data   []float64
	fs     int
	color  color.Color
	legend string
}

func plotAddTimeSeries(p *plot.Plot, series []plotSeries) {
	lines := []plot.Plotter{}
	for _, s := range series {
		xys := make(plotter.XYs, len(s.data))
		xr := 1 / float64(s.fs)
		for i, v := range s.data {
			xys[i].X = float64(i) * xr
			xys[i].Y = v
		}
		line, _ := plotter.NewLine(xys)
		line.LineStyle.Color = s.color
		line.LineStyle.Width = plotDrawResize
		line.LineStyle.Dashes = nil
		line.LineStyle.DashOffs = 0
		p.Legend.Add(s.legend, line)
		lines = append([]plot.Plotter{line}, lines...)
	}
	for _, line := range lines {
		p.Add(line)
	}
}

func plotAddSpecSeries(p *plot.Plot, series []plotSeries) {
	lines := []plot.Plotter{}
	for _, s := range series {
		data := s.data[1:]
		xys := make(plotter.XYs, len(data))
		xr := (float64(s.fs) / 2) / float64(len(data))
		for i, v := range data {
			xys[i].X = float64(i+1) * xr
			xys[i].Y = -1e100
			if 0 < v {
				xys[i].Y = 20 * math.Log10(v)
			}
		}
		line, _ := plotter.NewLine(xys)
		line.LineStyle.Color = s.color
		line.LineStyle.Width = plotDrawResize
		line.LineStyle.Dashes = nil
		line.LineStyle.DashOffs = 0
		p.Legend.Add(s.legend, line)
		lines = append([]plot.Plotter{line}, lines...)
	}
	for _, line := range lines {
		p.Add(line)
	}
}

func plotToImage(width, height int, plots [][]*plot.Plot) (image.Image, error) {
	w := vg.Length(width) * vg.Inch / 96
	h := vg.Length(height) * vg.Inch / 96
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

func newPlot() *plot.Plot {
	p, err := plot.New()
	if err != nil {
		panic(err)
	}
	p.Legend.Top = true
	p.Legend.Left = false
	for _, m := range []*vg.Length{
		&p.Title.Padding,
		&p.Title.TextStyle.Font.Size,
		&p.Legend.TextStyle.Font.Size,
		&p.Legend.Padding,
	} {
		*m *= plotDrawResize
	}
	for _, a := range []*plot.Axis{&p.X, &p.Y} {
		for _, m := range []*vg.Length{
			&a.Label.TextStyle.Font.Size,
			&a.LineStyle.Width,
			&a.Padding,
			&a.Tick.Label.Font.Size,
			&a.Tick.LineStyle.Width,
			&a.Tick.Length,
		} {
			*m *= plotDrawResize
		}
	}
	return p
}
