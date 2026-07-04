package mutations

import (
	"fmt"
	"time"
)

var historicalYears = []int{
	1071, 1453, 1923,
	1999, 2000, 2001,
	2019,
}

func yearVariants(base string) []string {
	cur := time.Now().Year()
	years := make([]int, 0, 12)
	for y := cur - 3; y <= cur+1; y++ {
		years = append(years, y)
	}
	years = append(years, historicalYears...)

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
