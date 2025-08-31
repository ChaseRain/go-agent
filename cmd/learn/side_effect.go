package main

import "fmt"

func Multiply(a, b int, reply *int) {
	*reply = a * b
}

func sideEffect() {
	n := 0
	reply := &n
	Multiply(10, 5, reply)
	fmt.Printf("Multiply: %d\n", *reply)

	fmt.Printf("n = %d\n", n)
}

func sideEffect2() {
	sideEffect()
}
