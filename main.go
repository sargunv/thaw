package main

import (
	"context"
	"errors"
	"io"
	"os"

	"charm.land/fang/v2"
	"github.com/sargunv/thaw/cmd"
)

func main() {
	errHandler := func(w io.Writer, styles fang.Styles, err error) {
		var exitErr *cmd.ExitError
		if errors.As(err, &exitErr) {
			return
		}
		fang.DefaultErrorHandler(w, styles, err)
	}

	err := fang.Execute(context.Background(), cmd.New(), fang.WithErrorHandler(errHandler))
	if err != nil {
		var exitErr *cmd.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		os.Exit(1)
	}
}
