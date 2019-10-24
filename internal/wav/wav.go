package wav

import (
	"log"

	"github.com/but80/voispire/internal/buffer"
	"github.com/mkb218/gosndfile/sndfile"
	"golang.org/x/xerrors"
)

// NewWavFileSource は、wavファイルをソースとする波形供給用バッファを作成します。
func NewWavFileSource(filename string) (*buffer.WaveSource, int, error) {
	var inInfo sndfile.Info
	fin, err := sndfile.Open(filename, sndfile.Read, &inInfo)
	if err != nil {
		return nil, 0, xerrors.Errorf("入力音声ファイルのオープンに失敗しました: %s: %w", filename, err)
	}

	const step = 4096
	ch := int(inInfo.Channels)
	s := buffer.NewWaveSource()

	go func() {
		defer fin.Close()
		defer s.Close()
		buf := make([]float64, step*ch)
		mix := make([]float64, step)
		log.Printf("debug: NewWavFileSource goroutine is started")
		for {
			n64, err := fin.ReadFrames(buf)
			if err != nil {
				log.Printf("error: 入力音声ファイルの読み込みに失敗しました: %s: %w", filename, err)
				return
			}
			n := int(n64)
			if n == 0 {
				return
			}
			if n < step {
				buf = buf[:n*ch]
			}
			if 1 < ch {
				for i := 0; i < n; i++ {
					for j := 0; j < ch; j++ {
						mix[i] += buf[i*ch+j]
					}
					mix[i] /= float64(ch)
				}
				s.Append(mix)
			} else {
				s.Append(buf)
			}
		}
	}()

	return s, int(inInfo.Samplerate), nil
}

// Load は、wavファイルを読み込み、モノラルの []float64 として返します。
func Load(filename string) ([]float64, int, error) {
	var inInfo sndfile.Info
	fin, err := sndfile.Open(filename, sndfile.Read, &inInfo)
	if err != nil {
		return nil, 0, xerrors.Errorf("入力音声ファイルのオープンに失敗しました: %s: %w", filename, err)
	}
	defer fin.Close()

	n0 := int(inInfo.Frames)
	ch := int(inInfo.Channels)
	result := make([]float64, n0*ch)
	n, err := fin.ReadFrames(result)
	if err != nil {
		return nil, 0, xerrors.Errorf("入力音声ファイルの読み込みに失敗しました: %s: %w", filename, err)
	}
	if int(n) != n0 {
		return nil, 0, xerrors.Errorf("入力音声ファイルの読み込みに失敗しました (%d != %d): %s", n, n0, filename)
	}

	if 1 < ch {
		tmp := result
		result = make([]float64, n0)
		for i := 0; i < n0; i++ {
			for j := 0; j < ch; j++ {
				result[i] += tmp[i*ch+j]
			}
			result[i] /= float64(ch)
		}
	}

	return result, int(inInfo.Samplerate), nil
}

// Save は、モノラルの []float64 をwavファイルとして保存します。
func Save(filename string, fs int, data []float64) error {
	iLastClip := -fs
	for i, v := range data {
		if -1.0 <= v && v <= 1.0 {
			continue
		}
		if v < -1.0 {
			data[i] = -1.0
		} else if 1.0 < v {
			data[i] = 1.0
		}
		if iLastClip+fs <= i {
			log.Printf("warn: クリッピングが発生しました: %.3f sec", float64(i)/float64(fs))
			iLastClip = i
		}
	}

	outInfo := sndfile.Info{
		Frames:     int64(len(data)),
		Samplerate: int32(fs),
		Channels:   1,
		Format:     sndfile.SF_FORMAT_WAV | sndfile.SF_FORMAT_PCM_16,
	}
	fout, err := sndfile.Open(filename, sndfile.Write, &outInfo)
	if err != nil {
		return xerrors.Errorf("出力音声ファイルのオープンに失敗しました: %s: %w", filename, err)
	}
	defer fout.Close()

	m, err := fout.WriteFrames(data)
	if err != nil {
		return xerrors.Errorf("出力音声ファイルの書き込みに失敗しました: %s: %w", filename, err)
	}
	if int(m) != len(data) {
		return xerrors.Errorf("出力音声ファイルの書き込みに失敗しました (%d != %d): %s: %w", filename, err)
	}
	fout.WriteSync()
	return nil
}
