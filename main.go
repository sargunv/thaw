package main

import (
	"context"
	"os"

	"charm.land/fang/v2"
	"github.com/sargunv/thaw/cmd"
)

func main() {
	if err := fang.Execute(context.Background(), cmd.New()); err != nil {
		os.Exit(1)
	}
}
