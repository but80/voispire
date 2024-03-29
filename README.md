# voispire

[![Go Report Card](https://goreportcard.com/badge/github.com/but80/voispire)](https://goreportcard.com/report/github.com/but80/voispire)
[![Godoc](https://godoc.org/github.com/but80/voispire?status.svg)](https://godoc.org/github.com/but80/voispire)
[![License](https://img.shields.io/badge/License-BSD%203--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause)

**Alpha Version**

コマンドラインで動作するボイスチェンジャーです。

## 特徴

- オープンソースかつ特許に抵触しない技術のみを用いて実装されています。
- 低遅延でのリアルタイムなフォルマントシフトが行えます（現バージョンでは、ピッチシフトはWAVファイルの変換にのみ適用できます）。

## 使用方法

引数なしで実行するとサブコマンド一覧が表示されます。

```
NAME:
   voispire - ボイスチェンジャー

USAGE:
   voispire [global options] command [command options] [arguments...]

DESCRIPTION:
   ピッチシフトに用いる基本周波数の抽出に
   「音声分析変換合成システム WORLD」
   https://github.com/mmorise/World を使用しています。

AUTHOR:
   but80 <mersenne.sister@gmail.com>

COMMANDS:
     version, v  バージョン情報を表示します
     device, d   オーディオデバイス一覧を表示します
     start, s    ストリーミングを開始します
     convert, c  ファイル変換を開始します
     help, h     Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h  show help
```

### `start` サブコマンド

**現バージョンでは、start サブコマンドはフォルマントシフトのみ使用できます。ピッチシフトはできません。**

```
NAME:
   voispire start - ストリーミングを開始します

USAGE:
   voispire start [command options] [ <input-device> [ <output-device> [ <output-file> ] ] ]

OPTIONS:
   --formant value, -f value  フォルマントシフト量 [半音] (default: 0)
   --rate value, -r value     ファイル出力サンプリング周波数（省略時は入力と同じ） (default: 0)
   --verbose, -v              詳細を表示
   --debug                    デバッグ情報を表示
```

- `voispire start -f 3` のようにすると、デフォルトのオーディオデバイスでストリーミングが開始されます。
  マイク等から入力された音声のフォルマントが3半音シフトされ、ヘッドホン等から変換後の音声が出力されます。
- 次項に説明する `device` サブコマンドで確認できるデバイスIDを指定すると、任意のオーディオデバイスを使用できます。
  例えば `voispire start -f 3 10 11` のようにすると、ID=10 の入力デバイス および ID=11 の出力デバイスが使用されます。
- `<output-file>` を指定すると、ストリーミングしながら音声ファイルにも保存できます。

### `device` サブコマンド

`voispire device` でオーディオデバイス一覧が表示されます。
使用するデバイスIDを控えてから `start` サブコマンドに使用してください。

入力デバイス・出力デバイスはそれぞれIDと対応するデバイスが異なることにご注意ください。

### `convert` サブコマンド

```
NAME:
   voispire convert - ファイル変換を開始します

USAGE:
   voispire convert [command options] <input-file> [ <output-file> ]

OPTIONS:
   --formant value, -f value       フォルマントシフト量 [半音] (default: 0)
   --rate value, -r value          ファイル出力サンプリング周波数（省略時は入力と同じ） (default: 0)
   --verbose, -v                   詳細を表示
   --debug                         デバッグ情報を表示
   --transpose value, -t value     ピッチシフト量 [半音] (default: 0)
   --frame-period value, -p value  フレームピリオド [msec] (default: 5)
```

- `voispire convert -t 6 -f 3 input.wav output.wav` のようにすると、音声ファイル `input.wav` を6半音ピッチシフト・3半音フォルマントシフトして `output.wav` に保存します。
- `<output-file>` を省略すると、デフォルトの出力デバイスで直接音声が再生されます。

## ビルド

### 必須環境

- 以下のいずれかのOS
  - Windows 10 + MSYS2 + MinGW-w64
  - Windows 10 + WSL + Ubuntu 18.04 + PulseAudio
  - macOS (動作未検証)
  - Linux (動作未検証)
- 以下のライブラリ
  - [PortAudio](http://www.portaudio.com/)
  - [libsndfile](http://www.mega-nerd.com/libsndfile/)
- [Go 1.12](https://golang.org/)

### Ubuntuのセットアップ手順

```bash
sudo apt install portaudio19-dev libsndfile-dev
```

### Windows開発環境のセットアップ手順

1. [MSYS2](https://www.msys2.org/) をインストール
   - [参考記事](https://qiita.com/Ted-HM/items/4f2feb9fdacb6c72083c)
   - Chocolatey を使用している場合は `choco install msys2` にて `C:\tools\msys64` にインストールできます
2. `C:\tools\msys64\mingw64.exe` より MSYS2+MinGW-w64 環境を起動
3. `pacman -Syu` にて更新後、一旦シェルを閉じて再度起動後に `pacman -Su` （いずれも必要なら管理者権限で）
4. 必要パッケージ・ツール類をインストール
   
   ```bash
   pacman -Sy base-devel
   pacman -Sy msys2-devel
   pacman -Sy git
   pacman -Sy mingw-w64-x86_64-toolchain
   pacman -Sy mingw-w64-x86_64-go
   pacman -Sy mingw-w64-x86_64-portaudio
   
   # Setup Go
   echo 'export GOPATH="$HOME/go"' >> ~/.bashrc
   echo 'export PATH="$GOPATH/bin:$PATH"' >> ~/.bashrc
   source ~/.bashrc
   
   # Setup Mage
   GO111MODULES=off go get -u -d github.com/magefile/mage
   ( cd "$GOPATH/src/github.com/magefile/mage" && go run bootstrap.go )
   ```

### ビルド手順

```bash
go run mage.go build
```

- [Mage](https://magefile.org/) をインストール済みの場合は `mage build` でビルドできますが、 [goenv](https://github.com/syndbg/goenv) を併用時に要求バージョンのGoがインストールされていないと `Error determining list of magefiles: failed to list non-mage gofiles` というエラーが発生します。メッセージからは分かりにくいのでご注意ください。

## 技術情報

### ピッチシフト

だいぶ端折った処理ですが、多分 [Melodyne と同じ方式](https://ja.wikipedia.org/wiki/%E3%82%BF%E3%82%A4%E3%83%A0%E3%82%B9%E3%83%88%E3%83%AC%E3%83%83%E3%83%81/%E3%83%94%E3%83%83%E3%83%81%E3%82%B7%E3%83%95%E3%83%88#%E4%BD%8D%E7%9B%B8%E3%81%A8%E6%99%82%E9%96%93%E3%82%92%E3%81%BB%E3%81%A9%E3%81%8F) です。フォルマントも一緒にずれるので、ピッチシフト量の引数の分だけフォルマントシフト量からマイナスすることで、結果的にキャンセルしています。

### フォルマントシフト

「周波数スペクトルにその包絡線の逆数をかけて一旦キャンセルし、シフトした包絡線をかけ直す」方法でフォルマントシフトを実装しています。

周波数スペクトルの包絡線はケプストラム分析によって抽出していますが、繰り返しこの処理を行うことで、より理想的な包絡線に漸近させる工夫を施しています。

## TODO

- ピッチシフト
  - f0不特定箇所ではフォルマントシフトのみ行う
  - WORLD のGo化とストリーミング
    - 発話開始箇所のプチノイズ軽減（f0の先読み）
- フォルマントシフト
  - 精度・速度向上
    - f0に同期して切り出し
    - 窓がけの影響を除去
  - 連続FFT処理
    - フレーム間の接続方式改善？
- 自動ビルド・リリース
- GUI
- DLL化

## License

[BSD 3-Clause License](./LICENSE)

- ピッチシフトに用いる基本周波数の抽出に [音声分析変換合成システム WORLD](https://github.com/mmorise/World) を使用しています。
