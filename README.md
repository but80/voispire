# voispire

[![Go Report Card](https://goreportcard.com/badge/github.com/but80/voispire)](https://goreportcard.com/report/github.com/but80/voispire)
[![MIT License](https://img.shields.io/badge/license-MIT-brightgreen.svg)](LICENSE)

**WORK IN PROGRESS**

ボイスチェンジャーです。

だいぶ端折った処理ですが、多分 Melodyne と同じ方式です → [技術情報](https://ja.wikipedia.org/wiki/%E3%82%BF%E3%82%A4%E3%83%A0%E3%82%B9%E3%83%88%E3%83%AC%E3%83%83%E3%83%81/%E3%83%94%E3%83%83%E3%83%81%E3%82%B7%E3%83%95%E3%83%88#%E4%BD%8D%E7%9B%B8%E3%81%A8%E6%99%82%E9%96%93%E3%82%92%E3%81%BB%E3%81%A9%E3%81%8F)

今のところデモとしてWAVファイルのピッチシフトを試せます（`input.wav` はモノラル16bit限定）。

```bash
# 引数は <ピッチシフト量[半音]> <入力音声ファイル> <出力音声ファイル保存先>
go run cmd/voispire/main.go 12 input.wav output.wav

# 出力を -- とするとPortAudioで直接再生
go run cmd/voispire/main.go 12 input.wav --
```

## TODO

- フォルマントシフト
- 基本周波数を先読みして発話開始箇所のプチノイズ軽減
- 実際の周波数とのズレを軽減
  - BPF通した波形の0クロスを見る？
- 入力のPortAudioストリーム化

## License

BSD 3-Clause License

This software includes the following packages under each license:

- [WORLD - a high-quality speech analysis, manipulation and synthesis system](https://github.com/mmorise/World) : BSD 3-Clause License
- [go-audio/audio](https://github.com/go-audio/audio) : Apache-2.0
- [go-audio/wav](https://github.com/go-audio/wav) : Apache-2.0
- and the various MIT/ISC licensed great softwares
