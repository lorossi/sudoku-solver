package main

import (
	"flag"
	"fmt"
)

func main() {
	directInput := flag.String("stringinput", "", "Load sudoku from console")
	fileInput := flag.String("fileinput", "", "Load sudoku from file")
	imageInput := flag.String("imageinput", "", "Load sudoku from console")
	directOutput := flag.Bool("stringoutput", false, "Print output in console")
	plaintextOutput := flag.Bool("plaintext", false, "Print output in plaintext")
	fileOutput := flag.String("fileoutput", "", "Write solved sudoku in file")
	imageOutput := flag.String("imageoutput", "", "Write solved sudoku over image")
	flag.Parse()

	if flag.NFlag() == 0 {
		flag.PrintDefaults()
		// now exit
		return
	}

	// check output flags
	if !*directOutput && *fileOutput == "" && *imageOutput == "" {
		flag.PrintDefaults()
		return
	}

	// variables declaration
	var e error
	var s Sudoku
	s = NewSudoku()

	// check input flags
	if *directInput != "" {
		if e = s.LoadFromBytes([]byte(*directInput)); e != nil {
			printError(e)
			return
		}
		printSuccess("Sudoku loaded from console")
	} else if *fileInput != "" {
		if e := s.LoadFromFile(*fileInput); e != nil {
			printError(e)
			return
		}
		printSuccess("Sudoku loaded from file", *fileInput)
	} else if *imageInput != "" {
		if e := s.LoadFromImage(*imageInput); e != nil {
			printError(e)
			return
		}
		printSuccess("Sudoku loaded from image", *imageInput)
	} else {
		flag.PrintDefaults()
		return
	}

	// attempt to solve the sudoku
	var iterations int64
	iterations, e = s.Solve()
	if e != nil {
		printError(e)
		return
	}

	printSuccess("Solved in", iterations, "iterations")

	// save the output as wanted by the user

	if *directOutput {
		// print in console
		fmt.Println(s.ShowGrid(*plaintextOutput))
	}

	if *fileOutput != "" {
		// save to file
		s.SaveToFile(*fileOutput)
		if e != nil {
			printError(e)
			return
		}
		printSuccess("File", *fileOutput, "created")
	}

	if *imageOutput != "" {
		// write as image
		e = s.SaveToImage(*imageOutput)
		if e != nil {
			printError(e)
			return
		}
		printSuccess("Image", *imageOutput, "created")
	}
}
