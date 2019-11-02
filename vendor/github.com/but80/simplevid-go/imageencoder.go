package simplevid

import (
	"image"
	"image/color"
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
	for y := 0; y < opts.Height; y += 2 {
		for x := 0; x < opts.Width; x += 2 {
			uSum := 0
			vSum := 0
			for i := 0; i < 2; i++ {
				for j := 0; j < 2; j++ {
					r, g, b, _ := img.At(x+j, y+i).RGBA()
					yy, uu, vv := color.RGBToYCbCr(uint8(r>>8), uint8(g>>8), uint8(b>>8))
					e.SetY(x+j, y+i, yy)
					uSum += int(uu)
					vSum += int(vv)
				}
			}
			e.SetU(x/2, y/2, uint8((uSum+2)/4))
			e.SetV(x/2, y/2, uint8((vSum+2)/4))
		}
	}
	return true
}
