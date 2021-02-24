package main

func getCoords(i, width int8) (x, y int8) {
	x = i % width
	y = i / width
	return
}
