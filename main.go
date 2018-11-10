/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package main

import (
	"log"
	"os"

	"github.com/ortuman/jackal/app"
)

func main() {
	prg := app.New(os.Stdout, os.Args)
	if err := prg.Run(); err != nil {
		log.Fatal(err)
	}
}
