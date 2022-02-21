// package main

// import (
// 	"os"

// 	"xupdate/merge"
// )

// func main() {
// 	if len(os.Args) != 4 {
// 		printUsage(1)
// 	}
// 	err := merge.File(os.Args[1], os.Args[2], os.Args[3])
// 	if err != nil {
// 		println(err.Error())
// 		printUsage(1)
// 	}
// }

// func printUsage(exitcode int) {
// 	println("usage: " + os.Args[0] + " oldfile newfile patchfile")
// 	os.Exit(exitcode)
// }
