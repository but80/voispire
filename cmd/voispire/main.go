package main

import (
	"os"
	"strings"

	"github.com/but80/voispire"
	"github.com/comail/colog"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	// Go >= 1.10 required
	_ "github.com/theckman/goconstraint/go1.10/gte"
)

var version = "unknown"

const description = `
   - 基本周波数の抽出に「音声分析変換合成システム WORLD」
     https://github.com/mmorise/World を使用しています。
`

func main() {
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
			err := errors.New("ピッチシフト量は -24..24 の数値である必要があります")
			return cli.NewExitError(err, 1)
		}
		infile := ctx.Args()[0]
		outfile := ""
		if 2 <= ctx.NArg() {
			outfile = ctx.Args()[1]
		}
		if err := voispire.Demo(transpose, infile, outfile); err != nil {
			return cli.NewExitError(err, 1)
		}
		return nil
	}

	colog.Register()
	app.Run(os.Args)
}
