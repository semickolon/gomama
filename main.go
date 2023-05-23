package main

import (
	"log"
	"os"

	"github.com/dlclark/regexp2"
	gomama "github.com/semickolon/gomama/src"
	"github.com/semickolon/gomama/src/replacer"
	"github.com/urfave/cli/v2"
)

func headlessReplace(regex *regexp2.Regexp, subst string, filenames []string) error {
	for _, fn := range filenames {
		data, err := os.ReadFile(fn)
		if err != nil {
			return err
		}

		s := string(data)
		s, err = replacer.Replace(s, regex, subst)
		if err != nil {
			return err
		}

		err = os.WriteFile(fn, []byte(s), 0666)
		if err != nil {
			return err
		}
	}

	return nil
}

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
			&cli.BoolFlag{
				Name:  "skip-review",
				Usage: "Skip review mode",
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "Apply all replacements immediately",
			},
		},
		Action: func(ctx *cli.Context) error {
			regexStr := ctx.String("regex")
			regex, err := regexp2.Compile(regexStr, 0)
			if err != nil {
				return err
			}

			filenames := ctx.StringSlice("file")
			var subst *string

			if ctx.IsSet("subst") {
				s := ctx.String("subst")
				subst = &s
			}

			if ctx.Bool("force") {
				if subst == nil {
					log.Fatal("subst must be given if --force")
				} else {
					return headlessReplace(regex, *subst, filenames)
				}
			}

			infoTitle := ctx.String("info-title")
			infoMessage := ctx.String("info-message")
			skipReview := ctx.Bool("skip-review")

			return gomama.Run(regex, subst, filenames, infoTitle, infoMessage, skipReview)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
