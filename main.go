package main

import (
	keycut "github.com/wyrmroot/keycut/src"
)

func main() {
	args := keycut.ParseOptions()
	for _, file := range args.Filepaths {
		keycut.Run(file, args)
	}
}
