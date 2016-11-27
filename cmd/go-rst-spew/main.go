package main

import (
	"os"

	"github.com/davecgh/go-spew/spew"

	"github.com/apparentlymart/go-rst"
)

func main() {
	fragment := rst.ParseFragment(os.Stdin, "-")

	spewer := &spew.ConfigState{
		Indent:                  "    ",
		SortKeys:                true,
		DisablePointerAddresses: true,
		DisableCapacities:       true,
	}

	spewer.Dump(fragment)
}
