package main

import (
	"fmt"
	"os"

	"./xupdate/diff"
)

func main() {
	oldfile := []byte("AB")
	newfile := []byte("")
	fmt.Println("oldfile byte→", oldfile)
	//oldfile := []byte{0xfa, 0xdd, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff}
	//newfile := []byte{0xfa, 0xdd, 0x00, 0x00, 0x00, 0xee, 0xee, 0x00, 0x00, 0xff, 0xfe, 0xfe}

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
