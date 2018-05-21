package main

import (
	"fmt"
	"strings"
)

func avg(ar []int) int {
	var sum int
	for _, v := range ar {
		sum += v
	}
	return sum / len(ar)
}

func extractBtih(magnet string) (string, error) {
	btihPrefix := "btih:"
	pos := strings.Index(magnet, btihPrefix)
	if pos == -1 {
		return "", fmt.Errorf("btih prefix not found")
	}

	btih := magnet[pos+len(btihPrefix):]
	del := strings.Index(btih, "&amp")
	if del == -1 {
		return btih, nil
	}

	return btih[:del], nil
}
