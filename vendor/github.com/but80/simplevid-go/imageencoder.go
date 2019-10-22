package simplevid

import (
	"image"
)

type imageEncoder struct {
	CallbackEncoder
	ch <-chan image.Image
}

// NewImageEncoder は、フレーム画像をチャネル経由で Image として渡す方式の新しい Encoder を返します。
func NewImageEncoder(opts EncoderOptions, ch <-chan image.Image) Encoder {
	e := &imageEncoder{ch: ch}
	e.CallbackEncoder = NewCallbackEncoder(opts, e.onDraw)
	return e
}

func (e *imageEncoder) onDraw(CallbackEncoder) bool {
	img, ok := <-e.ch
	if !ok {
		return false
	}
	opts := e.Options()
	for y := 0; y < opts.Height; y++ {
		for x := 0; x < opts.Width; x++ {
			c := img.At(x, y)
			r, g, b, _ := c.RGBA()
			e.SetRGB(x, y, int(r>>8), int(g>>8), int(b>>8))
		}
	}
	return true
}
