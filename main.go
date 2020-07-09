package main

import (
	"log"
	"os"

	"github.com/beyang/ghdump/addrepo"
	"github.com/beyang/ghdump/ghdump"
)

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "add" {
		if err := addrepo.Main(); err != nil {
			log.Fatal(err)
		}
		return
	}

	ghdump.Main()
}
