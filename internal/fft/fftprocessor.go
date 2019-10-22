package fft

import (
	"log"

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
	output    chan float64
	src       []float64
	width     int
	processor func([]complex128, []float64) []complex128
	onFinish  func()
}

// NewProcessor は、新しい Processor を作成します。
func NewProcessor(src []float64, width int, processor func([]complex128, []float64) []complex128) Processor {
	if width < 4 {
		width = 4
	}
	width = (width >> 1) << 1
	return &fftProcessor{
		fft:       fourier.NewFFT(width),
		src:       src,
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
		win := series.Hann(s.width)              // フレームを取り出す窓関数
		wave0 := make([]float64, s.width)        // 1フレーム分のソース時間波形
		spec0 := make([]complex128, s.width/2+1) // 1フレーム分のソース周波数スペクトル
		wave1 := make([]float64, s.width)        // wave0 を加工した結果
		wave1Prev := make([]float64, s.width)    // 直前のフレームの wave1

		// ソースのスライス末尾を切りの良いところまで 0 で埋める
		s.src = series.ExtendFloatSliceCeil(s.src, step)
		s.src = series.ExtendFloatSlice(s.src, step)

		for i := 0; i+step < len(s.src); i += step {
			// 窓がけして周波数スペクトルを作成
			win.Apply(wave0, s.src[i:])
			s.fft.Coefficients(spec0, wave0)
			series.CmplxDivFloatConst(spec0, spec0, float64(s.fft.Len())) // 振幅を調整

			// 周波数スペクトルを加工処理
			spec1 := s.processor(spec0, wave0)

			// 時間領域に戻す
			s.fft.Sequence(wave1, spec1)

			// 直前のフレームと合成しながら出力
			prev := wave1Prev[step:]
			for i := 0; i < step; i++ {
				s.output <- prev[i] + wave1[i]
			}
			wave1, wave1Prev = wave1Prev, wave1
		}
		if s.onFinish != nil {
			s.onFinish()
		}
		close(s.output)
	}()
}
