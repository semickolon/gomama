package main

import (
	"log"
	"os"

	gomama "github.com/semickolon/gomama/src"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:    "gomama",
		Version: "0.1.0",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "regex",
				Aliases:  []string{"r"},
				Usage:    "Regex pattern",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "subst",
				Aliases: []string{"s"},
				Usage:   "Substitution patern",
			},
			&cli.StringSliceFlag{
				Name:     "file",
				Aliases:  []string{"F"},
				Usage:    "File(s) to match against",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "info-title",
				Usage: "Info title",
			},
			&cli.StringFlag{
				Name:  "info-message",
				Usage: "Info message",
			},
		},
		Action: func(ctx *cli.Context) error {
			regexStr := ctx.String("regex")
			var subst *string
			filenames := ctx.StringSlice("file")
			infoTitle := ctx.String("info-title")
			infoMessage := ctx.String("info-message")

			if ctx.IsSet("subst") {
				s := ctx.String("subst")
				subst = &s
			}

			return gomama.Run(regexStr, subst, filenames, infoTitle, infoMessage)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
