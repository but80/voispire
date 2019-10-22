package simplevid

/*
#cgo LDFLAGS: -lavcodec -lavutil

// copyright (c) 2001 Fabrice Bellard
//
// This file is part of Libav.
//
// Libav is free software; you can redistribute it and/or
// modify it under the terms of the GNU Lesser General Public
// License as published by the Free Software Foundation; either
// version 2.1 of the License, or (at your option) any later version.
//
// Libav is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
// Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public
// License along with Libav; if not, write to the Free Software
// Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "libavcodec/avcodec.h"
#include "libavutil/frame.h"
#include "libavutil/imgutils.h"
static const char* encode(AVCodecContext *enc_ctx, AVFrame *frame, AVPacket *pkt, FILE *outfile) {
	int ret;
	// send the frame to the cEncoder
	ret = avcodec_send_frame(enc_ctx, frame);
	if (ret < 0) return "error sending a frame for encoding";
	while (ret >= 0) {
		ret = avcodec_receive_packet(enc_ctx, pkt);
		if (ret == AVERROR(EAGAIN) || ret == AVERROR_EOF) break;
		if (ret < 0) return "error during encoding";
		// printf("encoded frame %3"PRId64" (size=%5d)\n", pkt->pts, pkt->size);
		fwrite(pkt->data, 1, pkt->size, outfile);
		av_packet_unref(pkt);
	}
	return NULL;
}

struct CEncoder {
	int width;
	int height;
	int bit_rate;
	int gop_size;
	int fps;

	const AVCodec *codec;
	AVCodecContext *c;
	FILE *f;
	AVFrame *picture;
	AVPacket *pkt;
	int frame;
};

static void set_data(struct CEncoder *e, int ch, int x, int y, uint8_t v) {
	e->picture->data[ch][y*e->picture->linesize[ch]+x] = v;
}

static void on_log(void *avcl, int level, const char *fmt, va_list vl) {
	if (level <= AV_LOG_WARNING) {
		av_log_default_callback(avcl, level, fmt, vl);
	}
}

static const char* initialize(struct CEncoder *e, const char *filename) {
	av_log_set_level(AV_LOG_DEBUG);
	av_log_set_callback(on_log);

	avcodec_register_all();
	// find the mpeg1video cEncoder
	e->codec = avcodec_find_encoder(AV_CODEC_ID_H264);
	if (!e->codec) {
		return "codec not found";
	}
	e->c = avcodec_alloc_context3(e->codec);
	e->picture = av_frame_alloc();
	e->pkt = av_packet_alloc();
	if (!e->pkt) return "could not allocate packet";
	// put sample parameters
	e->c->bit_rate = e->bit_rate;
	// resolution must be a multiple of two
	e->c->width = e->width;
	e->c->height = e->height;
	// frames per second
	e->c->time_base = (AVRational){1, e->fps};
	e->c->framerate = (AVRational){e->fps, 1};
	e->c->gop_size = e->gop_size; // emit one intra frame every these frames
	e->c->max_b_frames=1;
	e->c->pix_fmt = AV_PIX_FMT_YUV444P;
	// open it
	if (avcodec_open2(e->c, e->codec, NULL) < 0) {
		return "could not open codec";
	}
	e->f = fopen(filename, "wb");
	if (!e->f) {
		return "could not open file";
	}
	e->picture->format = e->c->pix_fmt;
	e->picture->width  = e->c->width;
	e->picture->height = e->c->height;
	if (av_frame_get_buffer(e->picture, 32) < 0) {
		return "could not alloc the frame data";
	}
	e->frame = 0;
	return NULL;
}

static const char* begin_frame(struct CEncoder *e) {
	fflush(stdout);
	// make sure the frame data is writable
	if (av_frame_make_writable(e->picture) < 0) return "frame data is not writable";
	return NULL;
}

static const char* end_frame(struct CEncoder *e) {
	e->picture->pts = e->frame;
	e->frame++;
	// encode the image
	return encode(e->c, e->picture, e->pkt, e->f);
}

static uint8_t endcode[] = { 0, 0, 1, 0xb7 };

static const char* finalize(struct CEncoder *e) {
	// flush the cEncoder
	const char* result = encode(e->c, NULL, e->pkt, e->f);
	// add sequence end code to have a real MPEG file
	fwrite(endcode, 1, sizeof(endcode), e->f);
	return result;
}

static void free_resources(struct CEncoder *e) {
	fclose(e->f);
	avcodec_free_context(&e->c);
	av_frame_free(&e->picture);
	av_packet_free(&e->pkt);
}

*/
import "C"

import (
	"errors"
	"image/color"
	"unsafe"
)

type cEncoder = C.struct_CEncoder

type callbackEncoder struct {
	*cEncoder
	options EncoderOptions
	onDraw  func(CallbackEncoder) bool
}

// NewCallbackEncoder は、新しい CallbackEncoder を返します。
func NewCallbackEncoder(opts EncoderOptions, onDraw func(CallbackEncoder) bool) CallbackEncoder {
	return &callbackEncoder{
		cEncoder: &cEncoder{
			width:    C.int(opts.Width),
			height:   C.int(opts.Height),
			bit_rate: C.int(opts.BitRate),
			gop_size: C.int(opts.GOPSize),
			fps:      C.int(opts.FPS),
		},
		options: opts,
		onDraw:  onDraw,
	}
}

// Options は、このエンコーダに指定されたオプションを返します。
func (e *callbackEncoder) Options() EncoderOptions {
	return e.options
}

// Frame は、現在エンコード中のフレーム番号を返します。
func (e *callbackEncoder) Frame() int {
	return int(e.cEncoder.frame)
}

// SetYUV は、位置 (x, y) にYUVカラー (cy, cu, cv) の画素を描画します。
func (e *callbackEncoder) SetYUV(x, y, cy, cu, cv int) {
	C.set_data(e.cEncoder, 0, C.int(x), C.int(y), C.uint8_t(cy))
	C.set_data(e.cEncoder, 1, C.int(x), C.int(y), C.uint8_t(cu))
	C.set_data(e.cEncoder, 2, C.int(x), C.int(y), C.uint8_t(cv))
}

// SetRGB は、位置 (x, y) にRGBカラー (cr, cg, cb) の画素を描画します。
func (e *callbackEncoder) SetRGB(x, y, cr, cg, cb int) {
	cy, cu, cv := color.RGBToYCbCr(uint8(cr), uint8(cg), uint8(cb))
	C.set_data(e.cEncoder, 0, C.int(x), C.int(y), C.uint8_t(cy))
	C.set_data(e.cEncoder, 1, C.int(x), C.int(y), C.uint8_t(cu))
	C.set_data(e.cEncoder, 2, C.int(x), C.int(y), C.uint8_t(cv))
}

// EncodeToFile は、ビデオをエンコードしてファイルに保存します。
func (e *callbackEncoder) EncodeToFile(filename string) error {
	cFilename := C.CString(filename)
	defer C.free(unsafe.Pointer(cFilename))
	if msg := C.initialize(e.cEncoder, cFilename); msg != nil {
		return errors.New(C.GoString(msg))
	}
	defer C.free_resources(e.cEncoder)
	for {
		if msg := C.begin_frame(e.cEncoder); msg != nil {
			return errors.New(C.GoString(msg))
		}
		if !e.onDraw(e) {
			break
		}
		if msg := C.end_frame(e.cEncoder); msg != nil {
			return errors.New(C.GoString(msg))
		}
	}
	if msg := C.finalize(e.cEncoder); msg != nil {
		return errors.New(C.GoString(msg))
	}
	return nil
}
