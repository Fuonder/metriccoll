package main

import "os"

func main() {
	os.Exit(1) // want "direct call to os.Exit is forbidden in main.main"
}
