package mutations

import (
	"sync"
	"sync/atomic"

	"wordlistkingdom/pkg/bloom"
	"wordlistkingdom/pkg/filter"
	"wordlistkingdom/pkg/writer"
)

type Engine struct {
	workers int
	bloom   *bloom.Filter
	filter  *filter.PolicyFilter
	writer  *writer.Writer
}

func NewEngine(workers int, bf *bloom.Filter, pf *filter.PolicyFilter, w *writer.Writer) *Engine {
	if workers < 1 {
		workers = 1
	}
	return &Engine{workers: workers, bloom: bf, filter: pf, writer: w}
}

func (e *Engine) Run(keywords []string, usernames []string) (int64, error) {
	candidates := make(chan string, 100_000)

	var written atomic.Int64

	var wg sync.WaitGroup
	for i := 0; i < e.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for s := range candidates {
				if !e.filter.Passes(s) {
					continue
				}
				if !e.bloom.TestAndAdd(s) {
					e.writer.Write(s)
					written.Add(1)
				}
			}
		}()
	}

	e.generate(keywords, usernames, candidates)
	close(candidates)

	wg.Wait()
	return written.Load(), nil
}

func (e *Engine) generate(keywords []string, usernames []string, out chan<- string) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, e.workers)

	allTerms := make([]string, 0, len(keywords)+len(usernames))
	allTerms = append(allTerms, keywords...)
	allTerms = append(allTerms, usernames...)

	for _, kw := range allTerms {
		wg.Add(1)
		sem <- struct{}{}
		go func(k string) {
			defer wg.Done()
			defer func() { <-sem }()
			genKeywordMutations(k, out)
		}(kw)
	}
	wg.Wait()

	sendCombinations(keywords, out)

	for i, a := range keywords {
		for j, b := range keywords {
			if i == j {
				continue
			}
			combo := a + b
			for _, y := range yearVariants(combo) {
				out <- y
			}
			for _, s := range specialVariants(combo) {
				out <- s
			}
		}
	}

	for _, u := range usernames {
		uCases := caseVariants(u)
		for _, kw := range keywords {
			for _, sep := range separators {
				for _, cu := range uCases {
					combo := cu + sep + kw
					out <- combo
					for _, y := range yearVariants(combo) {
						out <- y
					}
					for _, s := range specialVariants(combo) {
						out <- s
					}
				}
			}
		}
	}

	for _, kw := range keywords {
		for _, season := range universalSeasons {
			for _, combo := range []string{kw + season, season + kw} {
				for _, y := range yearVariants(combo) {
					out <- y
				}
				for _, s := range specialVariants(combo) {
					out <- s
				}
			}
		}
		for _, month := range universalMonths {
			for _, combo := range []string{kw + month, month + kw} {
				for _, y := range yearVariants(combo) {
					out <- y
				}
				for _, s := range specialVariants(combo) {
					out <- s
				}
			}
		}
	}
}

func genKeywordMutations(kw string, out chan<- string) {
	out <- kw

	cases := caseVariants(kw)
	baseMuts := make([]string, 0, len(cases)*2)

	for _, cv := range cases {
		out <- cv
		baseMuts = append(baseMuts, cv)
		if leet := leetspeak(cv); leet != cv {
			out <- leet
			baseMuts = append(baseMuts, leet)
		}
	}

	for _, m := range baseMuts {
		years := yearVariants(m)
		for _, y := range years {
			out <- y
		}
		for _, s := range specialVariants(m) {
			out <- s
		}
		for _, y := range years {
			for _, s := range specialVariants(y) {
				out <- s
			}
		}
		sendAffixVariants(m, out)
	}
}
