package mutations

var separators = []string{".", "_", "-", "@", ""}

func sendCombinations(keywords []string, out chan<- string) {
	cases := make([][]string, len(keywords))
	for i, kw := range keywords {
		cases[i] = caseVariants(kw)
	}

	for i, a := range keywords {
		for j, b := range keywords {
			if i == j {
				continue
			}
			for _, sep := range separators {
				out <- a + sep + b
				for _, ca := range cases[i] {
					for _, cb := range cases[j] {
						if combo := ca + sep + cb; combo != a+sep+b {
							out <- combo
						}
					}
				}
			}
		}
	}
}
