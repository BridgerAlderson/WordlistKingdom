package mutations

import "fmt"

var historicalYears = []int{
	1071, 1453, 1923,
	1999, 2000, 2001,
	2019,
}

var extraYears []int

func SetExtraYears(years []int) {
	extraYears = years
}

func yearVariants(base string) []string {
	seen := make(map[int]bool)
	years := make([]int, 0, 20)
	add := func(y int) {
		if !seen[y] {
			seen[y] = true
			years = append(years, y)
		}
	}

	for y := currentYear - 3; y <= currentYear+1; y++ {
		add(y)
	}
	for _, y := range historicalYears {
		add(y)
	}
	for _, y := range extraYears {
		add(y)
	}

	out := make([]string, 0, len(years)*4)
	for _, y := range years {
		full := fmt.Sprintf("%d", y)
		short := fmt.Sprintf("%02d", y%100)
		out = append(out,
			base+full,
			full+base,
			base+short,
			short+base,
		)
	}
	return out
}
