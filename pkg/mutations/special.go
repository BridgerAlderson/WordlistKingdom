package mutations

import (
	"fmt"
	"time"
)

var specialSingle = []string{"!", "@", "#", "$", "%", "*", ".", "_", "?", "-"}

var currentYear = time.Now().Year()

func specialVariants(base string) []string {
	if base == "" {
		return nil
	}
	specialComplex := []string{
		"!.", "!1", "!12", "!123", "!@#",
		"@123", "123!", "1234", "12345",
		fmt.Sprintf("%d!", currentYear),
		fmt.Sprintf("%d!", currentYear+1),
		"#1",
	}

	capacity := len(specialSingle)*2 + len(specialComplex) + len(specialSingle)
	out := make([]string, 0, capacity)

	for _, c := range specialSingle {
		out = append(out, base+c)
	}
	for _, c := range specialSingle {
		out = append(out, c+base)
	}
	for _, c := range specialComplex {
		out = append(out, base+c)
	}

	runes := []rune(base)
	if len(runes) >= 4 {
		mid := len(runes) / 2
		left := string(runes[:mid])
		right := string(runes[mid:])
		for _, c := range specialSingle {
			out = append(out, left+c+right)
		}
	}
	return out
}
