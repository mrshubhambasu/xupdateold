package main

import (
	"os"

	"github.com/mrshubhambasu/xupdate/xupdate/diff"
)

func main() {
	if len(os.Args) != 4 {
		printUsage(1)
	}

	oldfile, err := os.ReadFile(os.Args[1]) //[]byte("I love coding")
	if err != nil {
		panic(err)
	}

	newfile, err := os.ReadFile(os.Args[2]) //[]byte("I love coding in Go")
	if err != nil {
		panic(err)
	}

	// generate a BSDIFF4 patch
	patch, err := diff.Bytes(oldfile, newfile)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(os.Args[3], patch, 0777)
	if err != nil {
		panic(err)
	}
}

func printUsage(exitcode int) {
	println("usage: " + os.Args[0] + " oldfile newfile patchfile")
	os.Exit(exitcode)
}
