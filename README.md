# voispire

[![Go Report Card](https://goreportcard.com/badge/github.com/but80/voispire)](https://goreportcard.com/report/github.com/but80/voispire)
[![Godoc](https://godoc.org/github.com/but80/voispire?status.svg)](https://godoc.org/github.com/but80/voispire)
[![License](https://img.shields.io/badge/License-BSD%203--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause)

**WORK IN PROGRESS**

ボイスチェンジャーです。

## 使用方法

```
USAGE:
   voispire [オプション...] <入力音声ファイル> [<出力音声ファイル保存先>]

GLOBAL OPTIONS:
   --transpose value, -t value  ピッチシフト量 [半音] (default: 0)
   --formant value, -f value    フォルマントシフト量 [半音] (default: 0)
   --verbose, -v                詳細を表示
   --debug                      デバッグ情報を表示
   --version                    バージョン番号を表示
   --help, -h                   show help
```

今のところデモとしてWAVファイルのピッチシフト・フォルマントシフトを試せます（`input.wav` はモノラル16bit限定）。

```bash
# 引数は -t <ピッチシフト量[半音]> -f <フォルマントシフト量[半音]> <入力音声ファイル> [<出力音声ファイル保存先>]
go run cmd/voispire/main.go -t 6 -f 3 input.wav output.wav

# 出力先を省略するとPortAudioで直接再生
go run cmd/voispire/main.go -t 6 -f 3 input.wav
```

## ビルド手順

### macOS / Linux

```bash
cd cmodules/world
make
cd ../..
go run mage.go build
./voispire -h
```

### Windows + MinGW

```batch
cd cmodules\world
make
cd ..\..
go run mage.go build
voispire.exe -h
```

## 技術情報

### ピッチシフト

だいぶ端折った処理ですが、多分 [Melodyne と同じ方式](https://ja.wikipedia.org/wiki/%E3%82%BF%E3%82%A4%E3%83%A0%E3%82%B9%E3%83%88%E3%83%AC%E3%83%83%E3%83%81/%E3%83%94%E3%83%83%E3%83%81%E3%82%B7%E3%83%95%E3%83%88#%E4%BD%8D%E7%9B%B8%E3%81%A8%E6%99%82%E9%96%93%E3%82%92%E3%81%BB%E3%81%A9%E3%81%8F) です。フォルマントも一緒にずれるので、ピッチシフト量の引数の分だけフォルマントシフト量からマイナスすることで、結果的にキャンセルしています。

### フォルマントシフト

「周波数スペクトルにその包絡線の逆数をかけて一旦キャンセルし、シフトした包絡線をかけ直す」方法でフォルマントシフトを実装しています。

周波数スペクトルの包絡線はケプストラム分析によって抽出していますが、繰り返しこの処理を行うことで、より理想的な包絡線に漸近させる工夫を施しています。

## TODO

- 基本周波数を先読みして発話開始箇所のプチノイズ軽減
- 入力のPortAudioストリーム化

## License

BSD 3-Clause License

This software includes the following packages under each license:

- [WORLD - a high-quality speech analysis, manipulation and synthesis system](https://github.com/mmorise/World) : BSD 3-Clause License
- [go-audio/audio](https://github.com/go-audio/audio) : Apache-2.0
- [go-audio/wav](https://github.com/go-audio/wav) : Apache-2.0
- and the various MIT/ISC licensed great softwares
