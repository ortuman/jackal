/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package main

import "github.com/ortuman/jackal/config"
import "fmt"

func main() {
	if err := config.Load("jackal.yaml"); err != nil {
		fmt.Printf("%v", err)
	}
	d := config.DefaultConfig
	println(d.PIDFile)
}
