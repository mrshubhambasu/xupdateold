package main

import (
	"fmt"
	"os"

	"./xupdate/diff"
)

func main() {
	oldfile := []byte("I love coding!")
	newfile := []byte("I love coding in golang!")
	fmt.Println("oldfile byte→", oldfile)

	// generate a BSDIFF4 patch
	_, err := diff.Bytes(oldfile, newfile)
	if err != nil {
		panic(err)
	}
	//	fmt.Println("patch→ ", patch)

	// Apply a BSDIFF4 patch
	// newfile2, err := diff.Bytes(oldfile, patch)
	// if err != nil {
	// 	panic(err)
	// }
	// if !bytes.Equal(newfile, newfile2) {
	// 	panic("missing")
	// }
	// if len(os.Args) != 4 {
	// 	printusage(1)
	// }
	// err := diff.File(os.Args[1], os.Args[2], os.Args[3])
	// if err != nil {
	// 	println(err.Error())
	// 	printusage(1)
	// }

}

func printusage(exitcode int) {
	println("usage: " + os.Args[0] + " oldfile newfile patchfile")
	os.Exit(exitcode)
}
