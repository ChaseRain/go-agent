package main

import "fmt"

var num int = 10
var numx2, numx3 int

func multipleReturn() {
	numx2, numx3 = getX2AndX3(num)
	PrintValue()
	numx2, numx3 = getX2AndX32(num)
	PrintValue()
}

func PrintValue() {
	fmt.Printf("num = %d,2X num = %d,3X num = %d", num, numx2, numx3)
}

func getX2AndX3(input int) (int, int) {
	return 2 * input, 3 * input
}

func getX2AndX32(input int) (x2 int, x3 int) {
	x2 = 2 * input
	x3 = 3 * input
	return
}
