package formant

import (
	"math"
	"math/cmplx"

	"github.com/but80/voispire/internal/buffer"
	"github.com/but80/voispire/internal/fft"
	"gonum.org/v1/gonum/fourier"
)

type cepstralShifter struct {
	fft.Processor
	cfft       *fourier.FFT
	width      int
	fs         int
	shiftInv   float64
	maxPeakNum int
	envelope   []float64
	envelopeDb []complex128
	specDb     []complex128
	ceps       []float64
	spec1      []complex128
}

// NewCepstralShifter は、ケプストラム分析を用いたフォルマントシフタを作成します。
func NewCepstralShifter(input *buffer.WaveSource, fs, width int, shift float64) FormantShifter {
	s := &cepstralShifter{
		cfft:       fourier.NewFFT(width),
		width:      width,
		fs:         fs,
		envelope:   make([]float64, width/2+1),
		envelopeDb: make([]complex128, width/2+1),
		specDb:     make([]complex128, width/2+1),
		ceps:       make([]float64, width),
		spec1:      make([]complex128, width/2+1),
	}
	s.Processor = fft.NewProcessor(input, width, func(spec0 []complex128, wave0 []float64) []complex128 {
		if len(spec0) <= 4 {
			return spec0
		}
		if len(spec0) != len(s.specDb) {
			panic("wrong length")
		}

		// 対数スペクトル
		for i, v := range spec0 {
			s.specDb[i] = complex(math.Log(cmplx.Abs(v)), 0)
		}

		// 包絡線（微細構造の中央を縫う）により隙間を埋めていく
		const kn = 16   // 繰り返し回数
		const cn0 = 192 // ケプストラム中の包絡線成分とみなす次数
		const cn1 = 96
		r := 1.0 / float64(s.width)
		specSrc := s.specDb
		for k := 1; k <= kn; k++ {
			// ケプストラム
			s.cfft.Sequence(s.ceps, specSrc)

			// 包絡線化
			cn := int(lerp(cn0, cn1, float64(k-1)/float64(kn-1)) + .5)
			for i := cn; i < len(s.ceps)-cn; i++ {
				s.ceps[i] = 0
			}

			// 包絡線を周波数軸に戻し、元の周波数スペクトルの隙間を埋める
			s.cfft.Coefficients(s.envelopeDb, s.ceps)
			if k < kn {
				for i, v := range s.envelopeDb {
					v1 := real(v) * r
					v0 := real(s.specDb[i])
					if v1 < v0 {
						v1 = v0
					}
					s.envelopeDb[i] = complex(v1, 0)
				}
			}
			specSrc = s.envelopeDb
		}

		// 対数スペクトルから通常のスペクトルに戻す
		for i, v := range s.envelopeDb {
			v1 := real(v) * r
			s.envelope[i] = math.Pow(math.E, v1)
		}

		// flattenLowerCoefs(s.envelope, s.fs)
		applyEnvelopeShift(s.spec1, spec0, s.envelope, shift)
		analyzerFrame(&analyzerData{
			fs:       fs,
			fftWidth: width,
			wave0:    wave0,
			envelope: s.envelope,
			spec0:    spec0,
			spec1:    s.spec1,
		})
		return s.spec1
	})
	s.Processor.OnFinish(analyzerFinish)
	return s
}
