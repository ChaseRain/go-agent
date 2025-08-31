package main

import (
	"fmt"
	"io"
	"log"
)

func func1(s string) (n int, err error) {
	defer func() {
		log.Printf("func1(%q) = %d, %v", s, n, err)
	}()

	fmt.Println("s = ", s)
	return 7, io.EOF
}

func defer_logvalues() {
	func1("Go")
}
