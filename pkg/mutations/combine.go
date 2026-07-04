package mutations

var separators = []string{".", "_", "-", "@", ""}

func combinations(keywords []string) []string {
	cases := make([][]string, len(keywords))
	for i, kw := range keywords {
		cases[i] = caseVariants(kw)
	}

	var out []string
	for i, a := range keywords {
		for j, b := range keywords {
			if i == j {
				continue
			}
			for _, sep := range separators {
				out = append(out, a+sep+b)
				for _, ca := range cases[i] {
					for _, cb := range cases[j] {
						out = append(out, ca+sep+cb)
					}
				}
			}
		}
	}
	return out
}
