package fft

import (
	"log"

	"github.com/but80/voispire/internal/buffer"
	"github.com/but80/voispire/internal/series"
	"gonum.org/v1/gonum/fourier"
)

// Processor は、FFT・逆FFTを用いて波形を加工する処理器です。
type Processor interface {
	Output() <-chan float64
	OnFinish(func())
	Start()
}

type fftProcessor struct {
	fft       *fourier.FFT
	input     *buffer.WaveSource
	output    chan float64
	width     int
	processor func([]complex128, []float64) []complex128
	onFinish  func()
}

// NewProcessor は、新しい Processor を作成します。
func NewProcessor(input *buffer.WaveSource, width int, processor func([]complex128, []float64) []complex128) Processor {
	return &fftProcessor{
		fft:       fourier.NewFFT(width),
		input:     input,
		width:     width,
		processor: processor,
		output:    make(chan float64, 4096),
	}
}

func (s *fftProcessor) Output() <-chan float64 {
	return s.output
}

func (s *fftProcessor) OnFinish(callback func()) {
	s.onFinish = callback
}

func (s *fftProcessor) Start() {
	go func() {
		log.Print("debug: fftProcessor goroutine is started")

		step := s.width / 2                      // フレームをずらす幅（フレーム自体の幅の半分）
		wave0 := make([]float64, s.width)        // 1フレーム分のソース時間波形
		spec0 := make([]complex128, s.width/2+1) // 1フレーム分のソース周波数スペクトル
		wave1 := make([]float64, s.width)        // wave0 を加工した結果
		wave1Prev := make([]float64, s.width)    // 直前のフレームの wave1

		// 窓関数の選び方について
		// - https://www.jstage.jst.go.jp/article/jasj/72/12/72_764/_pdf
		//   「3.2 完全再構成条件」
		// - https://jp.mathworks.com/help/signal/ref/iscola.html
		//   「ルート-ハン ウィンドウの COLA 準拠の確認」
		wa := series.SqrtHann(s.width) // 分析窓関数
		ws := series.SqrtHann(s.width) // 合成窓関数

		i := 0
		for {
			src, cont := s.input.Read(i, i+s.width)
			if !cont {
				src = series.ExtendFloatSlice(src, s.width-len(src))
			}

			// 窓がけして周波数スペクトルを作成
			wa.Apply(wave0, src)
			s.fft.Coefficients(spec0, wave0)
			series.CmplxDivFloatConst(spec0, spec0, float64(s.fft.Len())) // 振幅を調整

			// 周波数スペクトルを加工処理
			spec1 := s.processor(spec0, wave0)

			// 時間領域に戻して窓がけ
			s.fft.Sequence(wave1, spec1)
			ws.Apply(wave1, wave1)

			// 直前のフレームと合成しながら出力
			prev := wave1Prev[step:]
			for i := 0; i < step; i++ {
				s.output <- prev[i] + wave1[i]
			}

			s.input.DiscardUntil(i)
			if !cont {
				break
			}
			wave1, wave1Prev = wave1Prev, wave1
			i += step
		}
		if s.onFinish != nil {
			s.onFinish()
		}
		close(s.output)
	}()
}
