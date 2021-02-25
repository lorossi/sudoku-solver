package main

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io/ioutil"
	"math"
	"os"
	"strconv"

	"github.com/otiai10/gosseract"
	"gocv.io/x/gocv"
)

// Position -> struct containing the position, used in backtracking
type Position struct {
	x, y int8
}

// Sudoku -> object containing the sudoku grid
type Sudoku struct {
	grid                               [9][9]int8
	visited                            []Position
	debug                              bool
	remaining                          int8
	tempFolder                         string
	inputPath, outputPath              string
	minArea, maxArea, maxAspectRatio   float64
	border, blurRadius                 float64
	digitContainers                    []image.Rectangle
	containerRect                      image.Rectangle
	sourceImg, processedImg, outputImg gocv.Mat
	fontColor                          color.RGBA
}

// NewSudoku returns a new initialized sudoku
func NewSudoku() (s Sudoku) {
	s = Sudoku{}

	for y := int8(0); y < 9; y++ {
		for x := int8(0); x < 9; x++ {
			s.grid[y][x] = 0
		}
	}
	s.remaining = 81
	s.tempFolder = "temp/"
	s.minArea, s.maxArea, s.maxAspectRatio, s.border = 150, 800, 3, 5
	s.fontColor = color.RGBA{0, 0, 0, 255}
	s.blurRadius = 5
	s.debug = false
	return
}

// Create the temporary folder
func (s *Sudoku) createTemporaryFolder() (e error) {
	// check if folder already exists
	if _, e := os.Stat(s.tempFolder); !os.IsNotExist(e) {
		// if so, remove it
		os.RemoveAll(s.tempFolder)
	}

	return os.Mkdir(s.tempFolder, 0755)
}

// Delete the temporary folder
func (s *Sudoku) deleteTemporaryFolder() (e error) {
	return os.RemoveAll(s.tempFolder)
}

// Check if the row is valid
func (s *Sudoku) checkRow(y, val int8) (valid bool) {
	// if the same digit as val has been found, of course it's not valid
	for x := int8(0); x < 9; x++ {
		if s.grid[y][x] == val {
			return false
		}
	}

	return true
}

// Check if the column is valid
func (s *Sudoku) checkCol(x, val int8) (valid bool) {
	// if the same digit as val has been found, of course it's not valid
	for y := int8(0); y < 9; y++ {
		if s.grid[y][x] == val {
			return false
		}
	}

	return true
}

// Check if the cell (3x3) is valid
func (s *Sudoku) checkCell(pos Position, val int8) (valid bool) {
	// if the same digit as val has been found, of course it's not valid
	// unpack x and y
	x := pos.x
	y := pos.y
	// roud x and y to find the top left corner of cell
	x = x - (x % 3)
	y = y - (y % 3)

	// now iterate throught every position in the cell
	for dy := int8(0); dy < 3; dy++ {
		for dx := int8(0); dx < 3; dx++ {
			if s.grid[y+dy][x+dx] == val {
				return false
			}
		}
	}

	return true
}

// Check if a number can be added to visited
func (s *Sudoku) checkPos(pos Position, val int8) (valid bool) {
	// check row, col and cell (3x3)
	return s.checkRow(pos.y, val) && s.checkCol(pos.x, val) && s.checkCell(pos, val)
}

// Find first free cell
func (s *Sudoku) findFirstFree() (free Position) {
	// iterate throught all cells until you find an empty one
	// empty cells are marked as zero
	for y := int8(0); y < 9; y++ {
		for x := int8(0); x < 9; x++ {
			if s.grid[y][x] == 0 {
				return Position{x: x, y: y}
			}
		}
	}
	// no cell found, return -1 -1
	return Position{x: -1, y: -1}
}

// Process image
func (s *Sudoku) processImage() (e error) {
	// load source image
	s.sourceImg = gocv.IMRead(s.inputPath, gocv.IMReadColor)
	defer s.sourceImg.Close()
	if s.sourceImg.Empty() {
		return errors.New("Cannot load image as it's empty")
	}
	// convert it to gray
	gray := gocv.NewMat()
	gocv.CvtColor(s.sourceImg, &gray, gocv.ColorRGBToGray)
	// blur it
	blur := gocv.NewMat()
	gocv.GaussianBlur(gray, &blur, image.Point{X: 5, Y: 5}, 3, 3, gocv.BorderDefault)
	// threshold it using Otsu, no need to specify a threshold
	threshold := gocv.NewMatWithSize(s.sourceImg.Size()[0], s.sourceImg.Size()[1], gocv.MatTypeCV16S)
	gocv.Threshold(blur, &threshold, 0, 255.0, gocv.ThresholdBinary+gocv.ThresholdOtsu)

	if s.debug {
		gocv.IMWrite("process.png", threshold)
	}

	// copy the image into the permanent variable
	s.processedImg = threshold.Clone()
	return
}

// Calculate rect aspect ratio
func (s *Sudoku) rectAspectRatio(rect image.Rectangle) (aspectRatio float64) {
	// get width and height of rectangle
	w, h := float64(rect.Dx()), float64(rect.Dy())
	// calculate aspect ratio
	aspectRatio = w / h
	// if it's less than one, invert it
	if aspectRatio < 1 {
		aspectRatio = 1 / aspectRatio
	}
	return
}

// Detects the bounding boxes for the letters
func (s *Sudoku) imageDetectContainers() (containersFound int, containerArea float64) {
	s.sourceImg = gocv.IMRead(s.inputPath, gocv.IMReadColor)
	defer s.sourceImg.Close()
	debugImg := s.sourceImg.Clone()

	// var containing area of the candidate cotnainer rect
	containerArea = 0
	// var containing the outer rectangle
	s.containerRect = image.Rectangle{}
	// slice containing all the candidate digits
	// not always contain a number
	s.digitContainers = make([]image.Rectangle, 0)
	// no do some opencv magic and find all contours
	contours := gocv.FindContours(s.processedImg, gocv.RetrievalList, gocv.ChainApproxSimple)
	// iterate throught each one of them and find the one containing numbers
	for _, c := range contours {
		area := gocv.ContourArea(c)
		// the area must be between certain bounds
		if area >= s.minArea && area <= s.maxArea {
			rect := gocv.BoundingRect(c)
			aspectRatio := s.rectAspectRatio(rect)
			// the aspect ratio must be under a certain value (it must be more or less squar-ish)
			if aspectRatio <= s.maxAspectRatio {
				s.digitContainers = append(s.digitContainers, rect)
				if s.debug {
					gocv.Rectangle(&debugImg, rect, color.RGBA{255, 0, 0, 0}, 2)
				}
			}
		} else if area > containerArea {
			// if the found area is the biggest so far
			rect := gocv.BoundingRect(c)
			aspectRatio := s.rectAspectRatio(rect)
			// it must not touch image borders and its ratio must be approximately one
			if rect.Min.X != 0 && rect.Min.Y != 0 && math.Abs(aspectRatio-1) < 0.05 {
				// we found a new candidate container
				containerArea = area
				s.containerRect = rect
			}
		}
	}

	if s.debug {
		gocv.IMWrite("debug.png", debugImg)
	}

	containersFound = len(s.digitContainers)
	return
}

// Fill grid from single digit rects
func (s *Sudoku) fillGridFromContainers() (e error) {
	// reset the grid
	s.remaining = 81
	for y := int8(0); y < 9; y++ {
		for x := int8(0); x < 9; x++ {
			s.grid[y][x] = 0
		}
	}

	// load the new tesseract client
	client := gosseract.NewClient()
	defer client.Close()
	// set whitelist so that only number are accepted as valid
	client.SetWhitelist("0123456789")
	// calculate width and height of the container
	containerWidth, containerHeight := float64(s.containerRect.Max.X-s.containerRect.Min.X), float64(s.containerRect.Max.Y-s.containerRect.Min.Y)
	// calculate offset of the container (relative to the border)
	// the border is set because the container square is bigger than it should
	containerOffsetX, containerOffsetY := float64(s.containerRect.Min.X)+s.border/2, float64(s.containerRect.Min.Y)+s.border/2

	// now iterate throught the slice of rects that contain text
	for i, r := range s.digitContainers {
		// create a new img
		newImg := s.processedImg.Region(r)
		// save the new img in to the temp folder
		outputPath := s.tempFolder + strconv.Itoa(i) + ".png"
		if ok := gocv.IMWrite(outputPath, newImg); !ok {
			e = errors.New("Failed to write image: " + outputPath)
			return
		}
		// vars containing the string and the value corrisponding to the content of the
		// current container
		var text string
		var digit int
		client.SetImage(outputPath)
		text, _ = client.Text()
		// try to convert it to integer
		// if it fails, continue as it's not a number and was mis-selected
		digit, e = strconv.Atoi(text)
		if e != nil {
			fmt.Println("Error while converting", text, "to string. Error", e)
		}
		// get digit pos
		digitX := int((float64(r.Min.X) - containerOffsetX) / containerWidth * 9)
		digitY := int((float64(r.Min.Y) - containerOffsetY) / containerHeight * 9)
		// check boundaries
		if digitX < 0 || digitX >= 9 {
			continue
		} else if digitY < 0 || digitY >= 9 {
			continue
		}
		// fill the grid and decrease the number of digits to place
		s.grid[digitY][digitX] = int8(digit)
		s.remaining--
	}

	return nil
}

// LoadFromImage -> Load sudoku from image
func (s *Sudoku) LoadFromImage(inputPath string) (e error) {
	s.inputPath = inputPath
	// create temporary folder in which ocr-able images are stored
	if e := s.createTemporaryFolder(); e != nil {
		return e
	}
	// do some effects to image so that it's easier for the ocr to convert
	if e = s.processImage(); e != nil {
		return e
	}
	// detect all containers
	containers, area := s.imageDetectContainers()
	if containers == 0 || area == 0 {
		return errors.New("Cannot find digit in the image")
	}
	// if filling grid fails, return
	if e = s.fillGridFromContainers(); e != nil {
		return e
	}
	// delete temporary folder
	if e = s.deleteTemporaryFolder(); e != nil {
		return e
	}

	return nil
}

// SaveToImage -> Save the solved sudoku into source image
func (s *Sudoku) SaveToImage(outputPath string) (e error) {
	s.outputPath = outputPath
	// load output image
	s.outputImg = gocv.IMRead(s.inputPath, gocv.IMReadColor)
	defer s.outputImg.Close()

	if s.outputImg.Empty() {
		// if it does not exists, probably because the input has been passed throught string and or console,
		// create a new image
		s.outputImg = gocv.NewMatWithSizes([]int{520, 520}, gocv.MatTypeCV8UC3)
		containerWidth, containerHeight := 500, 500
		containerOffsetX, containerOffsetY := 10, 10
		fontSize := float64(2)
		textSize := gocv.GetTextSize("8", gocv.FontHersheyPlain, fontSize, 1)
		textDx, textDy := (containerWidth/9-textSize.X)/2, (containerWidth/9+textSize.Y)/2
		// create a rect containing the background so that we can set a color
		backgroundRect := image.Rectangle{Min: image.Point{X: 0, Y: 0}, Max: image.Point{X: containerWidth + 2*containerOffsetX, Y: containerHeight + 2*containerOffsetY}}
		gocv.Rectangle(&s.outputImg, backgroundRect, color.RGBA{255, 255, 255, 255}, int(gocv.Filled))
		// loop throught each grid position and create a square with the number inside (cell)
		for x := 0; x < 9; x++ {
			for y := 0; y < 9; y++ {
				// digit position on image calculation
				digitX := x*containerWidth/9 + containerOffsetX + textDx
				digitY := y*containerHeight/9 + containerOffsetY + textDy
				// convert digit to string
				digit := fmt.Sprint(s.grid[y][x])
				// write number
				gocv.PutText(&s.outputImg, digit, image.Point{X: int(digitX), Y: int(digitY)}, gocv.FontHersheyPlain, fontSize, s.fontColor, 1)
				// draw cell
				digitRect := image.Rectangle{Min: image.Point{X: containerOffsetX + containerWidth/9*x, Y: containerOffsetY + containerHeight/9*y}, Max: image.Point{X: containerOffsetX + containerWidth/9*(x+1), Y: containerOffsetY + containerHeight/9*(y+1)}}
				gocv.Rectangle(&s.outputImg, digitRect, color.RGBA{0, 0, 0, 255}, 1)
			}
		}

	} else {
		// calculate the position and offset of the container
		containerWidth, containerHeight := float64(s.containerRect.Dx())-s.border/2, float64(s.containerRect.Dy())-s.border/2
		containerOffsetX, containerOffsetY := float64(s.containerRect.Min.X)+s.border/2, float64(s.containerRect.Min.Y)+s.border/2
		// calculate the font size according to the size of the container
		fontSize := float64(0.0045 * containerHeight)
		textSize := gocv.GetTextSize("8", gocv.FontHersheyPlain, fontSize, 1)
		// calculate text offset isnide container
		textDx, textDy := (containerWidth/9-float64(textSize.X))/2, (containerWidth/9+float64(textSize.Y))/2
		// fill only the numbers inside the visited slice, thus skipping numbers that weren't placed by the algorithm
		for _, v := range s.visited {
			// digit position on image calculation
			digitX := float64(v.x)*containerWidth/9 + containerOffsetX + textDx
			digitY := float64(v.y)*containerHeight/9 + containerOffsetY + textDy
			// convert digit to string
			digit := fmt.Sprint(s.grid[v.y][v.x])
			// write number
			gocv.PutText(&s.outputImg, digit, image.Point{X: int(digitX), Y: int(digitY)}, gocv.FontHersheyPlain, fontSize, s.fontColor, 1)
		}
	}

	// save the output image
	gocv.IMWrite(s.outputPath, s.outputImg)

	return
}

// LoadFromBytes -> Load sudoku from a string
func (s *Sudoku) LoadFromBytes(sudokuString []byte) (e error) {
	if len(sudokuString) != 81 {
		return errors.New("invalid sudoku length")
	}
	// reset number of remaining (empty) cells
	s.remaining = 81

	for i, c := range sudokuString {
		x, y := getCoords(int8(i), int8(9))
		if c == 45 || c == 32 { //ascii(45) = "-", ascii(32) == " "
			// if the char is a dash, set 0 (zero)
			s.grid[y][x] = int8(0)
		} else {
			// otherwise, try to convert it as int8 and set the corresponding cell
			if e == nil {
				// by subracting 48 you get the digit
				s.grid[y][x] = int8(c - 48)
				s.remaining--
			} else {
				return errors.New("invalid sudoku string")
			}

		}
	}

	return
}

// LoadFromFile -> Load sudoku from file
func (s *Sudoku) LoadFromFile(inputPath string) (e error) {
	// var containing the content of the file
	var content []byte
	s.inputPath = inputPath
	// attempt to read from file
	content, e = ioutil.ReadFile(s.inputPath)
	if e != nil {
		return
	}
	// remove newlines
	content = bytes.ReplaceAll(content, []byte("\r"), []byte(""))
	content = bytes.ReplaceAll(content, []byte("\n"), []byte(""))
	// attempt to populate the grid with the parsed content
	if e = s.LoadFromBytes(content); e != nil {
		return
	}

	return nil
}

// SaveToFile -> saved solved sudoku to file
func (s *Sudoku) SaveToFile(outputPath string) (e error) {
	s.outputPath = outputPath
	// attempt to open the file and write on it
	e = ioutil.WriteFile(s.outputPath, []byte(s.ShowGrid(true)), 0744)
	if e != nil {
		return
	}

	return
}

// ShowGrid -> returns all the grid in a formatted manner
func (s *Sudoku) ShowGrid(plaintext bool) (grid string) {
	// var containing the newline seprator
	var newl string
	newl = ""
	if !plaintext {
		for i := int8(0); i < 13; i++ {
			newl += "-"
		}
		newl += "\n"
	}

	grid = ""
	grid += newl
	for y := int8(0); y < 9; y++ {
		if y%3 == 0 && y != 0 {
			grid += newl
		}

		for x := int8(0); x < 9; x++ {
			if x == 0 && !plaintext {
				grid += "|"
			} else if x%3 == 0 && x != 0 && !plaintext {
				grid += "|"
			}

			if val := s.grid[y][x]; val != 0 {
				grid += strconv.Itoa(int(val))
			} else if !plaintext {
				grid += " "
			}
		}

		if !plaintext {
			grid += "|\n"
		}
	}

	grid += newl
	return
}

// Solve -> Solve the sudoku
func (s *Sudoku) Solve() (iterations int64, e error) {
	var candidate int8
	var currentPosition Position
	// starting coords
	currentPosition = s.findFirstFree()
	// try to place this number
	candidate = 1
	// total number of iterations
	iterations = 0

	for s.remaining > 0 {
		iterations++

		if candidate > 9 {
			// all the number have been tried, time to backtrack
			if len(s.visited) == 0 {
				// we cannot backtrack -> the sudoku is not solvable
				e = errors.New("Sudoku is not solvable")
				return
			}

			// reset current cell
			s.grid[currentPosition.y][currentPosition.x] = 0
			s.remaining++
			// get last pos and resize slice (pop)
			lastPos := s.visited[len(s.visited)-1]
			s.visited = s.visited[:len(s.visited)-1]
			// restart from last value + 1
			// if it's 9, it will reset the next iteration
			currentPosition = lastPos
			candidate = s.grid[currentPosition.y][currentPosition.x] + 1
		} else if s.checkPos(currentPosition, candidate) {
			// place the number in this position
			s.grid[currentPosition.y][currentPosition.x] = candidate
			s.remaining--
			// add this coordinates to the visited list
			s.visited = append(s.visited, currentPosition)
			// reset candidate
			candidate = 1
			// find the next free coordinates
			currentPosition = s.findFirstFree()
		} else {
			candidate++
		}
	}

	return
}
