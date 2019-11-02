package simplevid

// EncoderOptions は、ビデオエンコーダのオプションです。
type EncoderOptions struct {
	// Width は、ビデオ画面の横幅 [px] です。
	Width int
	// Height は、ビデオ画面の縦幅 [px] です。
	Height int
	// BitRate は、ビットレート [byte/sec] です。
	BitRate int
	// GOPSize は、GOP (Group Of Picture) フレーム数です。
	GOPSize int
	// FPS は、1秒あたりのフレーム数です。
	FPS int
}

// Encoder は、ビデオエンコーダです。
type Encoder interface {
	// EncodeToFile は、ビデオをエンコードしてファイルに保存します。
	EncodeToFile(filename string) error
}

// CallbackEncoder は、コールバック中で画素を描画するビデオエンコーダです。
type CallbackEncoder interface {
	Encoder
	// Options は、このエンコーダに指定されたオプションを返します。
	Options() EncoderOptions
	// Frame は、現在エンコード中のフレーム番号を返します。
	Frame() int
	// SetY は、Yチャンネルの位置 (x, y) における画素値を設定します。
	SetY(x, y int, value uint8)
	// SetU は、Uチャンネルの位置 (x*2, y*2) - (x*2+1, y*2+1) における画素値を設定します。
	SetU(xHalf, yHalf int, value uint8)
	// SetV は、Vチャンネルの位置 (x*2, y*2) - (x*2+1, y*2+1) における画素値を設定します。
	SetV(xHalf, yHalf int, value uint8)
}
