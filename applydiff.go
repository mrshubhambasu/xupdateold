package main

import (
	"os"

	"./xupdate/merge"
)

func main() {
	if len(os.Args) != 4 {
		printusage(1)
	}
	err := merge.File(os.Args[1], os.Args[2], os.Args[3])
	if err != nil {
		println(err.Error())
		printusage(1)
	}
}

func printusage(exitcode int) {
	println("usage: " + os.Args[0] + " oldfile newfile patchfile")
	os.Exit(exitcode)
}
