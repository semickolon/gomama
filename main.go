package main

import (
	gomama "github.com/semickolon/gomama/src"

	"log"
	"os"
	"time"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:      "gomama",
		Version:   "0.1.0",
		Compiled:  time.Now(),
		ArgsUsage: "REGEX SUBST FILE...",
		Action: func(ctx *cli.Context) error {
			return gomama.Run(ctx.Args().Slice())
		},
		// Flags: {cli.Flag{}}, // TODO: Review only flag
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
