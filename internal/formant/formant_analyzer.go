// +build analyzer

package formant

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math/cmplx"

	"github.com/but80/simplevid-go"
	"gonum.org/v1/gonum/fourier"
	"gonum.org/v1/plot"
)

const (
	analyzerFilename  = "analyzer.mp4"
	analyzerTimeLimit = 1.0 // ビデオ出力を打ち切る分析時間
	analyzerPlayRate  = 1.0 // ビデオの再生速度（1.0未満でスローモーション）
)

var (
	analyzerVidOpts = simplevid.EncoderOptions{
		Width:   1920,
		Height:  1200,
		BitRate: 25 * 1024 * 1024,
		GOPSize: 30,
		FPS:     30,
	}
	analyzerWait    = make(chan struct{})
	analyzerImageCh = make(chan image.Image, 100)
)

func cmplxAbs(data []complex128) []float64 {
	result := make([]float64, len(data))
	for i, v := range data {
		result[i] = cmplx.Abs(v)
	}
	return result
}

var (
	analyzerInputFrameTime  = .0
	analyzerInputSpentTime  = .0
	analyzerOutputFrameTime = .0
	analyzerOutputSpentTime = .0
)

func analyzerStart(fs, fftStep int) {
	targetFPS := float64(fs) / float64(fftStep)
	analyzerInputFrameTime = 1.0 / targetFPS
	analyzerOutputFrameTime = 1.0 / float64(analyzerVidOpts.FPS)
	encoder := simplevid.NewImageEncoder(analyzerVidOpts, analyzerImageCh)
	log.Printf("debug: video option: %#v", analyzerVidOpts)
	go func() {
		if err := encoder.EncodeToFile(analyzerFilename); err != nil {
			panic(err)
		}
		close(analyzerWait)
	}()
}

func analyzerFrame(data *analyzerData) {
	if 0 < analyzerTimeLimit && analyzerTimeLimit <= analyzerInputSpentTime {
		// 分析済みの時間が analyzerTimeLimit に達したら打ち切り
		return
	}
	defer func() {
		analyzerInputSpentTime += analyzerInputFrameTime
	}()

	if analyzerInputSpentTime/analyzerPlayRate < analyzerOutputSpentTime {
		// 出力ビデオのFPSよりも分析速度が速いため、このフレームはスキップ
		log.Printf("debug: analyzer time %4.3f => %4.3f (skipped)", analyzerInputSpentTime, analyzerOutputSpentTime*analyzerPlayRate)
		return
	}

	// グラフを作成
	var (
		colorOrange = color.RGBA{R: 255, G: 112, B: 0, A: 255}
		colorBlue   = color.RGBA{R: 0, G: 192, B: 255, A: 255}
		colorRed    = color.RGBA{R: 255, G: 0, B: 0, A: 255}
	)

	pt := newPlot()
	wave1 := fourier.NewFFT((len(data.spec1)-1)*2).Sequence(nil, data.spec1)
	plotAddTimeSeries(pt, []plotSeries{
		{legend: "Input", data: data.wave0, fs: data.fs, color: colorOrange},
		{legend: "Output", data: wave1, fs: data.fs, color: colorBlue},
	})
	pt.Title.Text = "Time Domain"
	pt.X.Label.Text = fmt.Sprintf("Time [s] (at %4.3f)", analyzerInputSpentTime)
	pt.X.Min = 0
	pt.X.Max = float64(len(wave1)) / float64(data.fs)
	pt.Y.Label.Text = "Amplitude"
	pt.Y.Min = -1
	pt.Y.Max = 1

	pf := newPlot()
	plotAddSpecSeries(pf, []plotSeries{
		{legend: "Input", data: cmplxAbs(data.spec0), fs: data.fs, color: colorOrange},
		{legend: "Envelope", data: data.envelope, fs: data.fs, color: colorRed},
		{legend: "Output", data: cmplxAbs(data.spec1), fs: data.fs, color: colorBlue},
	})
	pf.Title.Text = "Frequency Domain"
	// pf.X.Scale = plot.LogScale{}
	// pf.X.Tick.Marker = plot.LogTicks{}
	pf.X.Label.Text = "Frequency [Hz]"
	pf.X.Min = 0
	pf.X.Max = float64(data.fs) / 2
	pf.Y.Label.Text = "Amplitude [dB]"
	pf.Y.Min = -100
	pf.Y.Max = 0

	img, err := plotToImage(
		analyzerVidOpts.Width,
		analyzerVidOpts.Height,
		[][]*plot.Plot{
			{pt},
			{pf},
		},
	)
	if err != nil {
		panic(err)
	}

	// ビデオ出力
	repeated := ""
	for analyzerOutputSpentTime <= analyzerInputSpentTime/analyzerPlayRate {
		log.Printf("debug: analyzer time %4.3f => %4.3f%s", analyzerInputSpentTime, analyzerOutputSpentTime*analyzerPlayRate, repeated)
		analyzerImageCh <- img
		analyzerOutputSpentTime += analyzerOutputFrameTime
		repeated = " (repeated)"
	}
}

func analyzerFinish() {
	close(analyzerImageCh)
	// log.Printf("debug: waiting video encoder finishes")
	<-analyzerWait
}
