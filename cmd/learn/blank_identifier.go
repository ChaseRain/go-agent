package main

import "fmt"

func blankIdentifier() {
	var i1 int
	var f1 float32
	i1, _, f1 = ThreeValues()
	fmt.Printf("i1 = %d, f1 = %f", i1, f1)

}

func ThreeValues() (int, int, float32) {
	return 5, 6, 7.5
}
