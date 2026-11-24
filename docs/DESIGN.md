# Design notes

This document records why WordlistKingdom v2 looks the way it does, what was
measured in v1.1, and how the new generator trades brute-force coverage for
realistic password shapes.

## 1. The problem with v1.1

v1.1 was not merely "too large"; it was producing the wrong candidates. The
generator applied every rule to every base with no probability model, and the
AD policy filter then removed the most plausible passwords while keeping the
most exotic ones.

### Measured baseline

Commands (v1.1 binary, 2026-11-24):

```bash
printf 'corp\ncorp.local\n' > inputs.txt
./wordlistkingdom -input inputs.txt -output v1_default.txt        # AD filter ON
./wordlistkingdom -input inputs.txt -output v1_raw.txt -no-filter
./wordlistkingdom -input keywords.txt -output v1_11.txt
```

| Run | Lines | Size |
|-----|------:|-----:|
| 2 keywords, AD filter | 161,455 | 3.1 MB |
| 2 keywords, no filter | 200,798 | 3.7 MB |
| 11 keywords, AD filter | 8,087,538 | 140.0 MB |

The README claimed 36,479 / 107,566 candidates for 11 keywords. The real
numbers were two orders of magnitude larger. The pipeline held only a bounded
channel, but `combinations()` materialised the full pair list before sending
anything, so the "zero OOM risk" claim was false for larger keyword files.

### What the filter removed

The following realistic candidates were absent from the AD-filtered output:

```
corp        Corp        CORP        corp.local
corp2026    Corp2026    corp!       corp2026!
Corp@2026   (present, but without its simpler neighbours)
```

The filter required uppercase, a digit **and** a special character. Active
Directory complexity requires **three of four** classes (uppercase, lowercase,
digit, special), so the implementation was stricter than the policy it claimed
to model. More importantly, filtering by an assumed policy at generation time
is backwards: a wordlist is an input to a spray/cracking tool, and the target
policy is not always known. v2 makes the policy opt-in and models AD correctly.

### What the filter kept

For the two-keyword input, the AD-filtered output contained:

| Shape | Lines |
|-------|------:|
| Leading-special candidates (`!Corp…`, `%01Corp…`) | 38,200 |
| Mid-word specials that break domains (`corp.loca*l…`) | 1,200 |
| Season/month combinations (`corp.localAugust…`) | 1,680 |

These are not impossible passwords, but they are vanishingly unlikely compared
with `Corp2026!`, and they crowded out the useful candidates at the top of the
file.

### Nondeterminism

Two runs with identical inputs and flags produced different files:

```
82a98ba8acc32ce6a880cabd8b667469  v1_raw.txt
74b8737e1540364d1c7ab2a6b932b163  v1_raw2.txt
```

The sets were identical but the line order was not, because multiple producer
and consumer goroutines wrote through a mutex. Order matters: it is the only
priority signal a plain wordlist has.

### Concrete defects

1. `combinations()` built `O(n² × 5 × 25)` strings in memory, then each pair
   was expanded by `yearVariants` (12 years × 4 forms) and `specialVariants`
   (~30 shapes), including “special on every year” (≈1,440 extra strings per
   pair).
2. Historical years `1071`, `1453`, `1923`, `1999`, `2000`, `2001`, `2019`
   were appended to every base. They belong to specific targets, not to every
   engagement.
3. Specials were prepended as well as appended, and inserted at the midpoint,
   with no guard for domain-shaped keywords.
4. Pair case variants were the full 5×5 product, including implausible forms
   such as `aDmIn.IsTaNbUl`.
5. Seasons and months were generated for every keyword at the default level.
6. Role affixes (`Admin`, `Test`, `Backup`, …) were generated for every keyword
   at the default level.
7. Leetspeak substituted every eligible character, producing `@dm1n` instead of
   the partial forms people actually use.
8. The AD filter was on by default, required uppercase + digit + special
   (stricter than AD's 3-of-4 rule), and measured length in bytes.
9. Deduplication was a Bloom filter with a user-tuned capacity; false positives
   silently dropped unique candidates and were only reported at the end.
10. Output order was nondeterministic and the writer swallowed write errors.
11. The pipeline's `Engine.Run` always returned `nil` and ignored writer errors.

## 2. Design principles for v2

### 2.1 Model passwords, not permutations

Breach analyses consistently show a small number of dominant structures:
word+digits, word+digits+symbol, capitalised word, all-lower, and simple
two-word combinations. v2 enumerates those structures and adds lower-probability
layers only when asked.

The grammar for a single term is:

```
case(base)
case(base) + number
case(base) + separator + number
case(base) + special
case(base) + year + special
case(base) + special + year
case(base) + number + special
case(base) + special + number
case(base) + date
```

Not every combination of every rule; each row is a separate, deliberate shape.

### 2.2 Priority order is the file order

The generator emits phases in descending likelihood:

```
core → numbers → dates → separated numbers → specials → year+special →
number+special → word pairs → usernames → role affixes → leet →
seasons/months → deep variants
```

Within a phase, input keyword order is preserved. A capped or truncated list
therefore keeps the most plausible candidates. This also makes the output
useful for incremental spraying, where you stop after the first N attempts.

### 2.3 Bound every combinatoric

- Only the first `-pair-limit` keywords (default 25) are paired.
- Three-word combinations use only the first `-triple-limit` keywords (default 8).
- Usernames cross only the first `-user-keywords` keywords (default 3).
- `-max-candidates` is a hard stop (default 5,000,000).
- `-perm` caps terms per candidate; `-level` caps enrichment layers.

A 100-keyword file no longer produces 100×99×5×25 pair bases. It produces at
most 25×24 ordered pairs, and the rest of the keywords still get single-term
coverage.

### 2.4 Deterministic, exact dedup

- Generation is single-threaded. The cost is irrelevant at these sizes (16k
  candidates in ~4 ms); the benefit is reproducible ordering.
- Deduplication is an open-addressed set of 64-bit FNV-1a hashes. A collision
  is possible in theory (≈2.7×10⁻⁶ chance of any collision among 10M entries)
  but is not a practical concern. Unlike a Bloom filter, it does not silently
  drop unique candidates at a 1% rate.
- The write path is buffered and error-checked; `Close` reports the first
  write/flush/close failure.

### 2.5 Curated sets

| Set | Default |
|-----|---------|
| Years | current, current−1 (full + two-digit); more at higher levels; extras via `-years` |
| Numbers | `123`, `1234`, `12345`, `1`, `12`, `99`; more at higher levels |
| Specials | `! @ . _ * ?`; `# $ - %` at standard; more later |
| Separators | `. _ @` for separated numbers; empty / `.` / `_` for pairs |
| Roles | `Admin Test Dev IT Support Backup Helpdesk Root` at standard |
| Seasons | 8 English/Turkish terms at extended |
| Months | 12 English + 12 Turkish at deep |
| Dates | Explicit via `-dates`: DDMMYYYY, DDMMYY, YYYYMMDD, YYMMDD, dotted/dashed/slashed, DDMM and MMDD |

Dates are opt-in because a wrong birthday is worse than no birthday. The date's
year also joins the normal year variants (`Corp1990`, `Corp90`, `Corp1990!`).
Date tokens containing `.`, `-` or `/` disappear when the `!` pattern class is
disabled; the digit-only tokens remain.

There is no hardcoded year list in v2. A Turkish engagement that needs `1923`
or a company founded in `1974` uses `-years 1923,1974`; the tool does not
assume every target cares about those dates.

### 2.6 No junk shapes

The policy layer rejects:

- leading separators (`_corp`, `.corp`);
- repeated separators (`corp..local`, `corp__corp`);
- whitespace unless `-allow-space`;
- control characters and invalid UTF-8 input lines;
- non-ASCII when `-ascii` is set;
- candidates outside `-min`/`-max`;
- AD-invalid candidates when `-ad-policy` is set.

The generator itself avoids:

- prefix symbols (the legacy tool emitted them; `!Corp` is rare, so v2 does not);
- mid-word symbols on keywords containing `.`, `_` or `-` (deep level only,
  plain alphanumeric words);
- appending digits to a base that already ends in a digit;
- appending symbols to a base that already ends in a symbol;
- mixed-case pair products before deep level;
- full leetspeak before deep level.

## 3. Levels in detail

Counts from the 11-keyword sample file:

| Phase | L0 | L1 | L2 | L3 |
|-------|----:|----:|----:|----:|
| Core | 33 | 33 | 33 | 33 |
| Numbers | 330 | 660 | 924 | 1,056 |
| Separated numbers | 594 | 1,320 | 1,584 | 1,584 |
| Specials | 198 | 330 | 429 | 594 |
| Year+special | 1,056 | 2,376 | 3,696 | 3,696 |
| Number+special | 792 | 1,980 | 3,168 | 3,168 |
| Pairs | 14,036 | 21,736 | 37,136 | 84,546 |
| Usernames | — | — | — | — |
| Role affixes | — | 1,360 | 8,304 | 8,304 |
| Leet | — | 24 | 112 | 280 |
| Seasons/months | — | — | 704 | 4,224 |
| Deep variants | — | — | — | 190 |
| **Written** | **16,841** | **28,779** | **53,726** | **105,077** |

The counts are attempts before deduplication and policy filtering; the written
totals are slightly lower (for example 17,039 attempts → 16,841 lines at L0).
The sample run has no `-dates`, so the date phase is zero; supplying dates adds
a separate phase without changing the other numbers.
The pair phase dominates because it is quadratic in the number of participating
keywords. That is intentional for 11 keywords, and bounded by `-pair-limit` for
larger files. `-pair-limit 0` reduces the 11-keyword focused run to 2,805
single-term candidates.

## 4. Case handling

Case forms are generated from spellings, not by blindly mixing both sides of a
pair:

- Spelling variants: source, ASCII fold, and (at extended level) camelCase for
  separator-delimited keywords.
- For each spelling: lowercase, Title, UPPER, source (if distinct), plus Turkish
  dotted/dotless-I forms when the spelling contains Turkish letters.
- Pair case patterns at focused level: lower+lower, TitleWhole, TitleEach,
  UPPER+UPPER, lower+TitleWhole. Focused uses whole-string title for
  no-separator pairs so it does not create `CorpCorp.Local`.
- Mixed 3×3 pair casing is deep-level only, where the cost is expected.
- Separator-delimited keywords get a PascalCase spelling at standard level
  (`active_directory` → `ActiveDirectory`) and camelCase at extended level
  (`activeDirectory`).

Turkish example: `İstanbul` produces:

```
istanbul
İstanbul
İSTANBUL
Istanbul
ISTANBUL
```

`-ascii` keeps only the ASCII forms; `-tr` forces Turkish variants for every
keyword instead of only those containing Turkish-specific letters.

### 4.1 Domain and compound derivation

A keyword is not only the exact string the operator typed. For separator
keywords v2 adds:

```
acme.local        -> acme, acmelocal
active_directory  -> active, activedirectory
Acme Corp         -> Acme (the joined AcmeCorp is already the primary)
```

The first component is added only when it contains a letter and is at least
three characters. The last component/TLD is not added; `local`, `com` and `net`
are generic noise. IP addresses and `www.` hosts are skipped. This closes a
real gap: an operator who enters only `acme.local` still gets `acme2026`,
`acmelocal2026` and `AcmeLocal2026` at standard level, not just domain-shaped
strings.

### 4.2 Rich user data

`-userlist` already supports name-order variants when the username contains
separators (`john.smith` -> `jsmith`, `smithj`). Many enterprise usernames do
not (`jdoe`), so `-userdata` accepts an optional CSV:

```csv
username,first,last,birthyear
jsmith,John,Smith,1990
```

The username forms are unchanged. The extra fields add:

```
john, smith, john.smith, John.Smith, johnsmith, JohnSmith,
JSmith, smithj, j.smith, smith.john, jsmith1990, jsmith1990!
```

The birth year joins the per-user number ladder (full and two-digit). It does
not leak into every keyword; only that account's name candidates. The
keyword cross still uses the account username and the first
`-user-keywords` keywords, not every name spelling, which keeps the output
bounded. A plain `-userlist` can be supplied together with `-userdata`; rich
records win and missing bare accounts are appended.

## 5. AD policy

`-ad-policy` applies:

- `len(candidate) >= -ad-min` (default 8, characters not bytes);
- at least three of: uppercase, lowercase, digit, non-alphanumeric.

This matches the common AD complexity rule. It is off by default because:

- not every target is AD;
- the wordlist may be used for offline cracking where the original policy is
  unknown;
- filtering at generation time is destructive and hides candidates.

`-min 8 -ad-policy` on the 11-keyword sample keeps 12,797 of 17,039 candidates.

## 6. Deduplication and exclusion

The candidate set and each exclusion list use the same exact hash set.

- Per candidate: `Hash` → exclude check → dedup add → policy → write.
- Exclude lines preserve leading/trailing spaces (they can be part of a
  password); only CRLF and a BOM are stripped.
- Gzip is detected by the first two bytes, not by suffix.
- Exclude lists use a sorted `[]uint64` index: roughly 8 bytes per unique
  word plus Go/runtime overhead. Measured: 5M words ≈ 90 MB RSS / 0.6 s,
  14M words ≈ 245 MB RSS / 1.7 s. Lookups are binary searches; candidates are
  a much smaller stream, so the trade is a clear win.
- Candidate deduplication still uses the small open-addressed set because it
  inserts while streaming and its working set is bounded by `-max-candidates`.

A Bloom filter would use even less memory but would drop candidates at a
nonzero false-positive rate. For a wordlist tool, silently losing a valid
candidate is worse than using more RAM.

## 7. Compatibility with the original tool

`wordlist_generator.exe` produced 390 lines for `corp` / `corp.local`. Its
shapes were:

- three case forms per keyword: lower, Title, UPPER;
- full and two-digit years;
- symbols `* ? ! @ _ .` appended, prefixed and sandwiched with numbers;
- cross-keyword concatenations, including doubled words;
- no policy filter.

v2 focused generates 1,033 lines for the same input and covers the high-value
subset of that grammar. Differences:

| Old behaviour | v2 behaviour |
|---------------|--------------|
| Static years in the binary (2023/2024) | Current year and previous year, dynamically |
| Full 3×3 pair case product | Five consistent forms at focused; nine at deep |
| Prefix symbols | Not generated |
| Mid-symbol inside domains | Not generated |
| AD filter absent | Optional, correctly modelled |
| Nondeterministic order | Deterministic priority order |
| Duplicates possible via Bloom | Exact dedup |

The old `-L`, `-o`, `-lvl`, `-perm`, `-pattern` and `-v` flags are accepted.
`-lvl` now selects a documented depth preset rather than an opaque internal
count; `-perm` selects the maximum number of terms. `-min-len` from v1.1 maps
to the output minimum length.

## 8. Testing strategy

- Unit tests for every pure component: case/fold, parsing, policy, dedup,
  source reader, writer.
- Generator tests assert both presence (real shapes) and absence (junk shapes
  at focused level), plus determinism, level supersets, caps and pattern
  gating.
- Integration tests run the whole pipeline against temp files and verify
  counts, uniqueness, exclusion, stdout purity, cancellation and errors.
- The race detector is run over the full suite.
- Short fuzz sessions cover date/pattern/special parsing, keyword and userdata
  readers, case forms and policy checks.
- A Windows CI job runs the test suite and a real binary smoke test (version,
  focused generation, AD policy, userdata, dates).

## 9. Known limits and future work

- Generation is single-threaded. A deterministic parallel merge is possible
  but unnecessary at current sizes.
- Exclusion tables are exact and sorted in memory. A partitioned on-disk set
  could reduce the 14M-line RSS further, at the cost of lookup latency.
- No hashcat rule/mask export yet. A future mode could emit a `.rule` file for
  a small base list, which is more efficient than enumerating every suffix.
- Locale-specific dictionaries (Turkish keyboard-adjacent substitutions,
  city plates) are intentionally left to keywords/`-numbers` rather than
  hardcoded.
- There is no interactive mode; the tool is designed to be scriptable.
