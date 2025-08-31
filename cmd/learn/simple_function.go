package main

import "fmt"

func simpleFunction() {
	fmt.Printf("Multiply 3 numbers: %d\n", MulitPly3Nums(1, 2, 3))
}

func MulitPly3Nums(a int, b int, c int) int {
	// var product int = a * b * c
	// return product
	return a * b * c
}
