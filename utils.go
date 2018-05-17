package main

func avg(ar []int) int {
	var sum int
	for _, v := range ar {
		sum += v
	}
	return sum / len(ar)
}
