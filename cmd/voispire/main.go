package main

import (
	"os"
	"strconv"
	"strings"

	"github.com/but80/voispire"
	"github.com/comail/colog"
	"github.com/urfave/cli"
	"golang.org/x/xerrors"
)

var version = "unknown"

const description = `
   ピッチシフトに用いる基本周波数の抽出に
   「音声分析変換合成システム WORLD」
   https://github.com/mmorise/World を使用しています。
`

var onExit func()

var commonFlags = []cli.Flag{
	cli.Float64Flag{
		Name:  "formant, f",
		Usage: "フォルマントシフト量 [半音]",
	},
	cli.BoolFlag{
		Name:  "verbose, v",
		Usage: "詳細を表示",
	},
	cli.BoolFlag{
		Name:  "debug",
		Usage: "デバッグ情報を表示",
	},
}

func parseFlags(ctx *cli.Context) (voispire.Options, error) {
	if ctx.Bool("debug") {
		colog.SetMinLevel(colog.LDebug)
	} else if ctx.Bool("verbose") {
		colog.SetMinLevel(colog.LInfo)
	} else {
		colog.SetMinLevel(colog.LWarning)
	}

	var o voispire.Options

	o.Formant = ctx.Float64("formant")
	if o.Formant < -12.0 || 12.0 < o.Formant {
		err := xerrors.New("フォルマントシフト量は -12..12 の数値である必要があります")
		return o, cli.NewExitError(err, 1)
	}

	o.Transpose = ctx.Float64("transpose")
	if o.Transpose < -12.0 || 12.0 < o.Transpose {
		err := xerrors.New("ピッチシフト量は -12..12 の数値である必要があります")
		return o, cli.NewExitError(err, 1)
	}

	o.FramePeriodMsec = ctx.Float64("frame-period")
	if o.FramePeriodMsec != 0 && (o.FramePeriodMsec < 1.0 || 200.0 < o.FramePeriodMsec) {
		err := xerrors.New("フレームピリオドは 1..200 の数値である必要があります")
		return o, cli.NewExitError(err, 1)
	}

	o.Rate = ctx.Int("rate")
	if o.Rate != 0 && (o.Rate < 8000 || 96000 < o.Rate) {
		err := xerrors.New("サンプリング周波数は 8000..96000 の数値である必要があります")
		return o, cli.NewExitError(err, 1)
	}

	return o, nil
}

var versionCmd = cli.Command{
	Name:    "version",
	Aliases: []string{"v"},
	Usage:   "バージョン情報を表示します",
	Action: func(ctx *cli.Context) error {
		cli.ShowVersion(ctx)
		return nil
	},
}

var deviceCmd = cli.Command{
	Name:    "device",
	Aliases: []string{"d"},
	Usage:   "オーディオデバイス一覧を表示します",
	Action: func(ctx *cli.Context) error {
		if err := voispire.ListDevices(); err != nil {
			return cli.NewExitError(err, 1)
		}
		return nil
	},
}

var startCmd = cli.Command{
	Name:      "start",
	Aliases:   []string{"s"},
	Usage:     "ストリーミングを開始します",
	ArgsUsage: "[ <input-device> [ <output-device> ] ]",
	Flags:     commonFlags,
	Action: func(ctx *cli.Context) error {
		o, err := parseFlags(ctx)
		if err != nil {
			return err
		}

		if 1 <= ctx.NArg() {
			o.InDevID, _ = strconv.Atoi(ctx.Args()[0])
		}

		if 2 <= ctx.NArg() {
			o.OutDevID, _ = strconv.Atoi(ctx.Args()[1])
		}

		if err := voispire.Start(o); err != nil {
			return cli.NewExitError(err, 1)
		}
		return nil
	},
}

var convertCmd = cli.Command{
	Name:      "convert",
	Aliases:   []string{"c"},
	Usage:     "ファイル変換を開始します",
	ArgsUsage: "<input-file> [ <output-file> ]",
	Flags: append(
		commonFlags,
		cli.Float64Flag{
			Name:  "transpose, t",
			Usage: "ピッチシフト量 [半音]",
		},
		cli.Float64Flag{
			Name:  "frame-period, p",
			Usage: "フレームピリオド [msec]",
			Value: 5.0,
		},
		cli.IntFlag{
			Name:  "rate, r",
			Usage: "出力サンプリング周波数（省略時は入力と同じ）",
		},
	),
	Action: func(ctx *cli.Context) error {
		o, err := parseFlags(ctx)
		if err != nil {
			return err
		}

		if ctx.NArg() < 1 {
			cli.ShowCommandHelpAndExit(ctx, "convert", 1)
		}

		if 1 <= ctx.NArg() {
			o.InFile = ctx.Args()[0]
		}

		if 2 <= ctx.NArg() {
			o.OutFile = ctx.Args()[1]
		}

		if err := voispire.Start(o); err != nil {
			return cli.NewExitError(err, 1)
		}
		return nil
	},
}

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
	app.Flags = []cli.Flag{}
	app.HideVersion = true

	app.Commands = []cli.Command{
		versionCmd,
		deviceCmd,
		startCmd,
		convertCmd,
	}

	app.Action = func(ctx *cli.Context) error {
		cli.ShowAppHelpAndExit(ctx, 1)
		return nil
	}

	colog.Register()
	app.Run(os.Args)
}
