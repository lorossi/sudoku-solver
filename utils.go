package main

import (
	"fmt"
	"os"
)

func getCoords(i, width int8) (x, y int8) {
	x = i % width
	y = i / width
	return
}

func printError(e error) {
	// bold white bright text on black background
	style := "\u001b[37;1m\u001b[41;1m\u001b[1m"
	// reset ansi code
	reset := "\u001b[0m"
	message := style + e.Error() + reset
	fmt.Fprint(os.Stdout, message)
	fmt.Fprint(os.Stdout, "\n")
}
