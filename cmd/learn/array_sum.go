package main

import "fmt"

func main() {
	array := [3]float64{1.0, 2.0, 3.0}
	sum := Sum(&array)
	fmt.Println(sum)
}

func Sum(a *[3]float64) (sum float64) {
	for _, v := range *a {
		sum += v
	}
	return
}
