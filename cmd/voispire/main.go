package main

import (
	"os"
	"strings"

	"github.com/but80/voispire"
	"github.com/comail/colog"
	"github.com/urfave/cli"
	"golang.org/x/xerrors"
)

var version = "unknown"

const description = `
   - 基本周波数の抽出に「音声分析変換合成システム WORLD」
     https://github.com/mmorise/World を使用しています。
`

var onExit func()

func main() {
	defer func() {
		if onExit != nil {
			onExit()
		}
	}()
	app := cli.NewApp()
	app.Name = "voispire"
	app.Version = version
	app.Usage = "ボイスチェンジャー"
	app.Description = strings.TrimSpace(description)
	app.Authors = []cli.Author{
		{
			Name:  "but80",
			Email: "mersenne.sister@gmail.com",
		},
	}
	app.HelpName = "voispire"
	app.UsageText = "voispire [オプション...] -t <ピッチシフト量[半音]> <入力音声ファイル> [<出力音声ファイル保存先>]"
	app.Flags = []cli.Flag{
		cli.Float64Flag{
			Name:  "transpose, t",
			Usage: "ピッチシフト量 [半音]",
		},
		cli.Float64Flag{
			Name:  "formant, f",
			Usage: "フォルマントシフト量 [半音]",
		},
		cli.Float64Flag{
			Name:  "frame-period, p",
			Usage: "フレームピリオド [msec]",
			Value: 5.0,
		},
		cli.Int64Flag{
			Name:  "rate, r",
			Usage: "出力サンプリング周波数（ファイル保存時のみ有効・省略時は入力と同じ）",
		},
		cli.BoolFlag{
			Name:  "verbose, v",
			Usage: "詳細を表示",
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "デバッグ情報を表示",
		},
		cli.BoolFlag{
			Name:  "version",
			Usage: "バージョン番号を表示",
		},
	}
	app.HideVersion = true

	app.Action = func(ctx *cli.Context) error {
		if ctx.Bool("version") {
			cli.ShowVersion(ctx)
			return nil
		}
		if ctx.NArg() < 1 {
			cli.ShowAppHelpAndExit(ctx, 1)
		}

		if ctx.Bool("debug") {
			colog.SetMinLevel(colog.LDebug)
		} else if ctx.Bool("verbose") {
			colog.SetMinLevel(colog.LInfo)
		} else {
			colog.SetMinLevel(colog.LWarning)
		}

		transpose := ctx.Float64("transpose")
		if transpose < -24.0 || 24.0 < transpose {
			err := xerrors.New("ピッチシフト量は -24..24 の数値である必要があります")
			return cli.NewExitError(err, 1)
		}

		formant := ctx.Float64("formant")
		if formant < -24.0 || 24.0 < formant {
			err := xerrors.New("フォルマントシフト量は -24..24 の数値である必要があります")
			return cli.NewExitError(err, 1)
		}

		framePeriodMsec := ctx.Float64("frame-period")
		if framePeriodMsec < 1.0 || 200.0 < framePeriodMsec {
			err := xerrors.New("フレームピリオドは 1..200 の数値である必要があります")
			return cli.NewExitError(err, 1)
		}

		rate := ctx.Int64("rate")
		if rate != 0 && (rate < 8000 || 96000 < rate) {
			err := xerrors.New("サンプリング周波数は 8000..96000 の数値である必要があります")
			return cli.NewExitError(err, 1)
		}

		infile := ctx.Args()[0]
		outfile := ""
		if 2 <= ctx.NArg() {
			outfile = ctx.Args()[1]
		}
		if err := voispire.Demo(transpose, formant, framePeriodMsec*.001, int(rate), infile, outfile); err != nil {
			return cli.NewExitError(err, 1)
		}
		return nil
	}

	colog.Register()
	app.Run(os.Args)
}
