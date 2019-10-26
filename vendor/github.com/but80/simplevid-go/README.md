# simplevid-go

simplevid-go は、とても簡単に利用できるGo言語向けビデオエンコーダです。

- エンコーダに渡したコールバックが1フレームごとに呼び出されるので、その中で画素を描画するだけでビデオを作成することができます。
- 現時点では、フォーマットは H264 YUV420 に固定されています。
- 使用例は [imageencoder_test.go](imageencoder_test.go), [callbackencoder_test.go](callbackencoder_test.go) をお読みください。

## 必須環境

- 以下のいずれかのOS
  - Windows + WSL
  - Linux
  - macOS（動作はしますが、作成された動画はQuicktimeで再生できない場合があります）
- 依存ライブラリ
  - libavcodec
  - libavutil
  - libavformat

```bash
# Ubuntu
sudo apt install libavcodec-dev libavutil-dev libavformat-dev
```

## ライセンス

[libav公式サンプル](https://libav.org/documentation/doxygen/master/encode_video_8c-example.html) をベースに実装しているため、これと同様に [LGPL 2.1 or later](LICENSE) とします。
