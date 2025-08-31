package main

import "fmt"

// Season 函数接受一个代表月份的数字，返回所代表月份所在季节的名称
func Season(month int) string {
	switch month {
	case 12, 1, 2:
		return "冬季"
	case 3, 4, 5:
		return "春季"
	case 6, 7, 8:
		return "夏季"
	case 9, 10, 11:
		return "秋季"
	default:
		return "无效月份"
	}
}

func season() {
	// 测试所有月份
	fmt.Println("=== 季节测试 ===")

	months := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

	for _, month := range months {
		season := Season(month)
		fmt.Printf("%d月: %s\n", month, season)
	}

	fmt.Println("\n=== 边界测试 ===")
	// 测试无效月份
	invalidMonths := []int{0, 13, -1, 25}

	for _, month := range invalidMonths {
		season := Season(month)
		fmt.Printf("%d月: %s\n", month, season)
	}

	fmt.Println("\n=== 交互式测试 ===")
	// 交互式测试
	var userMonth int
	fmt.Print("请输入月份 (1-12): ")
	fmt.Scanf("%d", &userMonth)

	result := Season(userMonth)
	fmt.Printf("您输入的 %d 月属于: %s\n", userMonth, result)
}
