package main

import (
	"fmt"
	"strconv"
)

func stringConversion2() {
	//定义字符串
	var orig string = "123"
	//定义整数
	// var an int
	var newS string
	// var err error

	fmt.Printf("The size of ints is: %d\n", strconv.IntSize)
	//将字符串转换为整数
	// anInt.err = strconv.Atoi(orig)
	an, err := strconv.Atoi(orig)
	//判断是否报错
	if err != nil {
		fmt.Printf("orig %s is not an integer - exiting with error\n", orig)
		return
	}
	//打印结果
	fmt.Printf("The integer is: %d\n", an)
	an = an + 5
	//将整数转换为字符串
	newS = strconv.Itoa(an)
	fmt.Printf("The new string is: %s\n", newS)

	// switch 语句
	switch i := 0; i {
	case 0:
		fallthrough
	case 1:
		fmt.Printf("i is 1")
	}

}
