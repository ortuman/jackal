/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package main

import (
	"fmt"
	"os"

	"github.com/ortuman/jackal/app"
)

func main() {
	exitCode, err := app.New(os.Stdout, os.Args).Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
	}
	os.Exit(exitCode)
}
