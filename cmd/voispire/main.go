package main

import (
	"os"
	"strings"

	"github.com/but80/voispire"
	"github.com/comail/colog"
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
	app.UsageText = "voispire [オプション...] <入力音声ファイル> <出力音声ファイル保存先>"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "verbose, v",
			Usage: "詳細を表示します",
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "デバッグ情報を表示します",
		},
		cli.BoolFlag{
			Name:  "version",
			Usage: "バージョン番号を表示します",
		},
	}
	app.HideVersion = true

	app.Action = func(ctx *cli.Context) error {
		if ctx.Bool("version") {
			cli.ShowVersion(ctx)
			return nil
		}
		if ctx.NArg() < 2 {
			cli.ShowAppHelpAndExit(ctx, 1)
		}

		if ctx.Bool("debug") {
			colog.SetMinLevel(colog.LDebug)
		} else if ctx.Bool("verbose") {
			colog.SetMinLevel(colog.LInfo)
		} else {
			colog.SetMinLevel(colog.LWarning)
		}

		if err := voispire.Demo(ctx.Args()[0], ctx.Args()[1]); err != nil {
			panic(err)
		}
		return nil
	}

	colog.Register()
	app.Run(os.Args)
}
