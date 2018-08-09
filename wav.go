package voispire

import (
	"fmt"
	"os"
	"reflect"

	"github.com/go-audio/wav"
	dspwav "github.com/mjibson/go-dsp/wav"
	"github.com/pkg/errors"
)

func loadWav(filename string) ([]float64, int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()
	w, err := dspwav.New(file)
	if err != nil {
		return nil, 0, err
	}
	samples, err := w.ReadSamples(w.Samples)
	if err != nil {
		return nil, 0, err
	}
	result := make([]float64, w.Samples)
	switch s := samples.(type) {
	case []int16:
		for i, v := range s {
			result[i] = float64(v) / 32767.0
		}
	default:
		return nil, 0, fmt.Errorf("Unsupported sample size: %s", reflect.TypeOf(samples))
	}
	return result, int(w.SampleRate), nil
}

func saveWav(filename string, fs int, data []float64) error {
	out, err := os.Create(filename)
	if err != nil {
		return errors.Wrap(err, "出力音声ファイルのオープンに失敗しました")
	}
	enc := wav.NewEncoder(out, fs, 16, 1, 1)
	for _, v := range data {
		if err := enc.WriteFrame(uint16(v * 32767)); err != nil {
			return errors.Wrap(err, "出力音声ファイルの書き込みに失敗しました")
		}
	}
	if err := enc.Close(); err != nil {
		return errors.Wrap(err, "出力音声ファイルのクローズに失敗しました")
	}
	return out.Close()
}
