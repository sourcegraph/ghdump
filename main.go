package main

import (
	"log"
	"os"

	"github.com/sourcegraph/ghdump/addrepo"
	"github.com/sourcegraph/ghdump/ghdump"
)

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "add" {
		var filterText = ""
		if len(os.Args) >= 3 {
			filterText = os.Args[2]
		}
		if err := addrepo.Main(filterText, true); err != nil {
			log.Fatal(err)
		}
		return
	}

	ghdump.Main()
}
