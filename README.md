# WordlistKingdom

```
 ██╗    ██╗ ██╗  ██╗
 ██║    ██║ ██║ ██╔╝
 ██║ █╗ ██║ █████╔╝
 ██║███╗██║ ██╔═██╗
 ╚███╔███╔╝ ██║  ██╗
  ╚══╝╚══╝  ╚═╝  ╚═╝   WordlistKingdom v2.0
```

Target-specific password candidate generator for penetration tests, red team
engagements, password spraying and offline cracking.

WordlistKingdom v2 is a complete rewrite of v1.1. It no longer emits a
Cartesian explosion of every possible mutation. Instead it generates the
password shapes that real people actually choose — `Corp2026`, `corp2026!`,
`corp.local`, `Corpcorp`, `corp_24`, `AcmeCorp!` — in priority order, with
deterministic output and exact deduplication.

For a two-keyword file such as `corp` / `corp.local`, v1.1 produced **161,455**
lines and dropped the most realistic candidates. v2.0 produces **1,033** useful
lines, including the ones v1.1 filtered out. For the sample 11-keyword file,
v1.1 produced **8.1 million** lines / 140 MB; v2.0 focused produces **16,841**
lines / 252 KB.

See [docs/DESIGN.md](docs/DESIGN.md) for the measured comparison and the full
generation model.

---

## Quick start

```bash
# Build
go build -trimpath -ldflags="-s -w" -o wordlistkingdom .

# Focused list (recommended default for spraying)
./wordlistkingdom -L keywords.txt -o corp_wordlist.txt

# Standard list with role affixes and partial leetspeak
./wordlistkingdom -L keywords.txt -o corp_standard.txt -lvl 1

# Active Directory spray list: 8+ chars and AD complexity
./wordlistkingdom -L keywords.txt -o corp_ad.txt -min 8 -ad-policy

# Exclude everything already present in rockyou
./wordlistkingdom -L keywords.txt -o corp_new.txt -exclude rockyou.txt

# Legacy wordlist_generator.exe invocation
./wordlistkingdom -L inputs.txt -o wordlist.txt -v
```

Output is plain text, one candidate per line, ready for `hashcat`, `hydra`,
`kerbrute`, `crackmapexec`/`netexec` or any other tool that accepts a wordlist.

---

## Why the output looks different

The generator is built around four rules:

1. **Human password shapes, not algebra.** Word+digits and word+digits+symbol
   dominate real password dumps. Prefix symbols (`!Corp`), mid-word symbols
   (`corp.loca!l`) and fully mixed case products (`cOrP.CoRp`) are rare and are
   removed from the default output.
2. **Priority tiers.** The wordlist is written in descending likelihood: base
   words first, then years/numbers, then symbols, then combinations. If you cap
   the list or use only its head, you still have the best candidates.
3. **Bounded combinations.** Only the first `-pair-limit` keywords (default 25)
   are cross-combined, so a 500-line keyword file cannot turn into a quadratic
   monster. Input order is importance order.
4. **Deterministic output.** No random map iteration, no goroutine race in the
   writer. The same input and flags produce a byte-identical file.

The default (`-lvl 0`) deliberately omits role affixes, seasons, months,
leetspeak, prefix years and mid-word symbols. Those exist, but behind explicit
levels so you can choose the trade-off between size and coverage.

---

## Installation

Requires Go 1.21 or newer.

```bash
git clone https://github.com/BridgerAlderson/WordlistKingdom.git
cd WordlistKingdom
go build -trimpath -ldflags="-s -w" -o wordlistkingdom .
```

Or use the Makefile:

```bash
make build          # current platform
make test           # unit + integration tests
make race           # race detector
make cross          # Linux/macOS/Windows binaries in dist/
make checksums      # dist/SHA256SUMS
make release        # check + cross + checksums
```

`dist/` is not committed. Tagged releases (`v*`) are built by
[.github/workflows/release.yml](.github/workflows/release.yml), which runs the
test suite and a Windows smoke test, builds all five binaries, generates
`SHA256SUMS`, and publishes the GitHub release.

Verify a downloaded set from inside `dist/`:

```bash
cd dist && sha256sum -c SHA256SUMS
```

---

## Usage

```
wordlistkingdom [flags]
```

### Input and output

| Flag | Default | Description |
|------|---------|-------------|
| `-input`, `-L` | `keywords.txt` | Keyword file, or `-` for stdin. If the default is missing, `inputs.txt` is used automatically. |
| `-output`, `-o` | `custom_wordlist.txt` | Output file, or `-` for stdout. |
| `-userlist` | _(none)_ | Username list. `CORP\jsmith` and `jsmith@corp.local` are normalised to `jsmith`. |
| `-userdata` | _(none)_ | CSV with `username,first,last,birthyear` for full-name and birth-year pivots. Header optional. |
| `-exclude` | _(none)_ | Comma-separated wordlists whose entries must not appear in the output. Gzip is detected by magic bytes. |
| `-v` | `false` | Echo every candidate to stdout. Legacy behaviour; combine with a file output. |
| `-progress` | `false` | Live written/elapsed counter on stderr. |
| `-quiet` | `false` | No banner or summary. |
| `-version` | `false` | Print the version and exit. |

### Generation

| Flag | Default | Description |
|------|---------|-------------|
| `-level`, `-lvl` | `0` | Depth preset: `0` focused, `1` standard, `2` extended, `3` deep. |
| `-perm` | `2` | Maximum terms per candidate: `1` singles only, `2` singles + pairs, `3` also triples. |
| `-pattern` | all | Character-class selector: `q` lower, `@` upper, `%` digits, `!` specials. |
| `-pair-limit` | `25` | Number of keywords used in two-word combinations. `0` disables pairs. |
| `-triple-limit` | `8` | Number of keywords used in three-word combinations. |
| `-user-keywords` | `3` | Keywords crossed with each username. `0` disables username×keyword pivots. |
| `-years` | _(none)_ | Extra years, comma-separated, e.g. `1923,1974`. |
| `-dates` | _(none)_ | Target dates for birthday/founding-date shapes, e.g. `1990-05-17,2000-01-02`. |
| `-numbers` | _(none)_ | Extra numeric suffixes, comma-separated. |
| `-specials` | _(none)_ | Extra special suffixes. `-specials '!@#._'` adds characters; `-specials '!,!123'` adds tokens. |
| `-tr` | `false` | Add Turkish dotted/dotless-I case variants for every keyword. |

### Output policy

| Flag | Default | Description |
|------|---------|-------------|
| `-min` | `1` | Minimum candidate length in characters. |
| `-max` | `32` | Maximum candidate length. `0` means the 128-character hard cap. |
| `-ad-policy` | `false` | Enforce AD complexity: minimum length plus 3 of 4 character classes. |
| `-ad-min` | `8` | Minimum length enforced by `-ad-policy`. |
| `-ascii` | `false` | Drop non-ASCII candidates. ASCII-folded forms remain. |
| `-allow-space` | `false` | Allow spaces in candidates. Multi-word keywords otherwise become joined forms. |
| `-max-candidates` | `5000000` | Hard output cap. `0` means unlimited. |

### Legacy compatibility

The original `wordlist_generator.exe` flags are supported:

| Old flag | New behaviour |
|----------|----------------|
| `-L` | Alias of `-input`. |
| `-o` | Alias of `-output`. |
| `-lvl` | Alias of `-level`. |
| `-perm` | Maximum terms per candidate (default 2). |
| `-pattern` | Character-class selector. |
| `-v` | Echo candidates. |
| `-min-len` | Alias of `-min` (v1.1 compatibility). |

`-workers`, `-no-filter` and `-bloom-size` from v1.1 are accepted but ignored.
Generation is deterministic and single-threaded by design; deduplication is
exact, so there is no Bloom filter to tune.

---

## Levels

Every level is a superset of the previous one. Counts below are from the
11-keyword sample file on the reference machine; your numbers depend on the
keywords and the current year.

| Level | Added on top of the previous level | Sample count | Size |
|-------|------------------------------------|--------------|------|
| `0` focused | Base words, years/numbers, separated numbers, specials, year+special, number+special, word pairs, doubled words | 16,841 | 252 KB |
| `1` standard | 8 role affixes (`AdminCorp`, `corp_backup`), partial leetspeak (`a→@`, `o→0`, `i→1`), PascalCase spellings (`CorpLocal`), more numbers/symbols/years | 28,779 | 416 KB |
| `2` extended | Seasons (English + Turkish), 8 more roles, `.`/`_` affix separators, `-`/`@` pair separators, camelCase spellings, more years and symbols | 53,726 | 782 KB |
| `3` deep | Months, prefix years (`2026Corp`), numeric prefixes (`123corp`), safe mid-word symbols (`Ad!min`), all leet patterns, mixed-case pair matrix | 105,077 | 1.5 MB |

Use `-lvl 0` for spraying. Use `-lvl 1` when you have budget for a few more
attempts. Use `-lvl 2` or `3` for offline cracking where wordlist size is cheap.

Domain- and separator-containing keywords are expanded automatically at every
level:

```text
acme.local        -> acmelocal, acme, AcmeLocal (standard+)
active_directory  -> activedirectory, active, ActiveDirectory (standard+)
```

`-dates` adds the shapes people build from birthdays and founding dates:

```text
1990-05-17 -> Corp17051990, corp170590, corp19900517,
              corp17.05.1990, corp1705, corp0517,
              Corp!17051990, corp1990, corp90
```

Accepted formats are `YYYY-MM-DD`, `DD.MM.YYYY`, `DD/MM/YYYY`, `YYYYMMDD` and
`DDMMYYYY`. The date's year is also added to the year/number variants. Dates
with separators are filtered automatically when the `!` pattern class is
disabled.

To make the focused list smaller, `-pair-limit 0` removes all combinations
(2,805 candidates for the 11-keyword sample):

```bash
./wordlistkingdom -L keywords.txt -o singles.txt -pair-limit 0
```

---

## What gets generated

### Core and numbers

| Shape | Example |
|-------|---------|
| Source, lowercase, Title, UPPER | `corp`, `Corp`, `CORP`, `corp.local`, `Corp.local`, `CORP.LOCAL` |
| Word + year (full and two-digit) | `corp2026`, `corp26`, `Corp2025`, `CORP26` |
| Word + common number | `corp123`, `corp1234`, `corp12345`, `corp1`, `corp99` |
| Word + separator + number | `corp_26`, `corp.2026`, `Corp@99`, `CORP.LOCAL_26` |
| Word + symbol | `corp!`, `Corp@`, `CORP*`, `corp?`, `corp_` |
| Year + symbol, both orders | `Corp2026!`, `Corp!2026`, `corp2026*`, `corp2026?` |
| Number + symbol, both orders | `Corp123!`, `Corp!123`, `corp99!`, `corp123*` |
| Target date shapes | `Corp17051990`, `corp17.05.1990`, `Corp!17051990` |

The year set is dynamic: current year, previous year and, from standard level,
`current-2`. Extra years are supplied with `-years`; historical years are never
inserted for every keyword automatically.

### Word pairs

Only the first `-pair-limit` keywords participate. Ordered pairs are generated
because both directions occur in real password dumps:

```
corp + corp.local  ->  corpcorp.local, Corpcorp.local, corpCorp.local,
                       CORPCORP.LOCAL, corp.local.corp, corp.local_corp, ...
```

At focused level the separator set is empty, `.`, and `_`. Extended level adds
`-` and `@`. Mixed-case pairs such as `CORPcorp` are deep-level only.

A keyword combined with itself is also generated for alphanumeric terms:
`corpcorp`, `Corpcorp`, `corpCorp`, `CORPCORP`. This is a real pattern and was
present in the legacy tool's output.

### Usernames

With `-userlist`, each account gets:

- case forms of the username (`jsmith`, `Jsmith`, `JSMITH`);
- year, number and symbol variants (`jsmith2026`, `jsmith123`, `jsmith!`);
- name-order forms when the input has separators:
  `john.smith` → `johnsmith`, `smithjohn`, `jsmith`, `smithj`, `j.smith`,
  `smith.john`;
- a cross with the first `-user-keywords` keywords:
  `jsmith.corp`, `Jsmith_Corp`, `JSMITH-CORP2026`, ...

With `-userdata`, each record can also carry a first name, last name and birth
year. That adds real name shapes:

```text
jsmith,John,Smith,1990 ->
  john.smith, John.Smith, johnsmith, JohnSmith, JSmith,
  smithj, j.smith, smith.john,
  jsmith1990, jsmith1990!, john.smith1990, JSmith2026
```

The birth year joins the year variants for that user; the cross-keyword pivots
still use the account name, not every name spelling, to keep the list bounded.

The full user list is never crossed with every keyword; that quadratic product
is how v1.1 reached millions of lines.

### Role affixes, seasons and leetspeak

These are level-gated because they are lower-probability shapes:

| Layer | Level | Examples |
|-------|-------|----------|
| Role affixes | 1 | `AdminCorp`, `ADMINCORP`, `CorpAdmin`, `corp_backup` |
| Partial leet | 1 | `c0rp`, `c@rp`, `adm1n`, `P@ssword` |
| Seasons | 2 | `corpSummer`, `Summercorp2026`, `corpYaz` |
| Months | 3 | `corpOcak`, `Ocakcorp2026`, `corpAugust` |
| Prefix years | 3 | `2026corp`, `26corp` |
| Mid-word symbols | 3 | `Ad!min` (only plain alphanumeric words; domain-shaped keywords are skipped) |

Full leetspeak (`p@$$w0rd`) and the mixed-case pair matrix are deep-level only.

---

## Pattern selector

`-pattern` controls which mutation layers are generated, using the legacy
characters:

| Char | Layer |
|------|-------|
| `q` | Lowercase forms |
| `@` | Title/UPPER forms |
| `%` | Numeric suffixes |
| `!` | Special-character suffixes and separators |

Examples:

```bash
# Lowercase + digits only
./wordlistkingdom -L corp.txt -pattern 'q%' -o corp_q.txt

# Uppercase/capitalised + symbols, no digits
./wordlistkingdom -L corp.txt -pattern '@!' -o corp_upper.txt

# All layers (default)
./wordlistkingdom -L corp.txt -pattern 'q@%!' -o corp_all.txt
```

The selector chooses mutation layers, not a strict character whitelist. A
Title-case form counts as the uppercase layer. The source spelling of a keyword
is always preserved where possible.

---

## Input format

`keywords.txt`: one keyword per line. Empty lines and lines starting with `#`
are ignored. Quotes around a keyword are removed. Duplicate spellings are
eliminated case-insensitively. Keywords longer than 64 characters are skipped.

```text
# Company names
Acme
AcmeCorp

# Domain / brand variants
acme.local
acme-global

# Locations
Istanbul
Ankara

# Departments and products
IT
Helpdesk
VPN
Portal

# Important years (optional; also use -years)
1923
```

Tips:

- Put the strongest, most target-specific keywords first. Only the first
  `-pair-limit` keywords are combined.
- Add the company name, its abbreviation, the internal domain without the TLD,
  city, founding year and product/project names.
- Multi-word keywords are joined automatically (`Acme Corp` → `AcmeCorp`).
  Use `-allow-space` if you also want the spaced form.
- Domain and separator keywords are expanded into their useful parts:
  `acme.local` adds `acme` and `acmelocal`; `active_directory` adds `active`
  and `activedirectory`. The last component (TLD) is not added, because
  `local`, `com` and similar TLDs are generic noise.
- Turkish characters are supported. `İstanbul` produces `istanbul`,
  `Istanbul`, `İstanbul`, `ISTANBUL` and `İSTANBUL`; use `-ascii` to drop the
  non-ASCII forms.
- Keep engagement-specific keyword files out of version control. `.gitignore`
  already ignores common input/output names.

`users.txt`: one login per line. `DOMAIN\user`, `DOMAIN/user` and
`user@domain` are normalised to `user`. Duplicate accounts are removed
case-insensitively.

`people.csv` (optional `-userdata`): username plus optional first name, last
name and birth year. A header is optional; common header aliases are accepted.

```csv
username,first,last,birthyear
jsmith,John,Smith,1990
asmith,Alice,Smith,
mlee,Mei,Lee,1987
```

A single-column CSV is equivalent to a plain username list. Birth years are
validated; an invalid or out-of-range value stops the run instead of silently
producing a wrong candidate.

---

## Excluding known wordlists

`-exclude` drops any candidate that already exists in the supplied lists. This
is the fastest way to avoid wasting spray attempts on rockyou material.

```bash
./wordlistkingdom -L corp.txt -o corp_new.txt -exclude rockyou.txt
./wordlistkingdom -L corp.txt -o corp_new.txt -exclude rockyou.txt.gz,previous.txt
```

Gzip is detected by magic bytes, so a renamed `.txt` archive also works.
Exclusion is exact (64-bit hashes; collision probability is negligible), not
probabilistic. Exclude lists use a sorted 64-bit index, roughly 8 bytes per
unique word plus Go/runtime overhead: 5M words ≈ 90 MB RSS and 14M words
≈ 245 MB RSS. Loading is a one-time sort: ~0.6 s for 5M and ~1.7 s for 14M in
the reference environment. Candidate deduplication still uses a small
open-addressed set.

Every candidate that matches is removed, including a keyword's own core form
when that keyword appears in the exclusion list.

---

## Output guarantees

- One candidate per line, UTF-8, `\n` terminated.
- No duplicate lines.
- No leading separators (`_Corp`, `.Corp`).
- No repeated separators (`corp..local`, `corp__corp`).
- No mid-word symbols on domain-shaped keywords.
- Deterministic order and content for identical inputs and flags.
- Candidates are written in priority order; truncating the file keeps the most
  likely candidates.
- `-v` echoes exactly the written candidates to stdout; status and summary go
  to stderr, so `-output -` pipelines are never polluted by the banner.
- `Ctrl+C` flushes the partial wordlist and reports how much was written.

---

## Performance

Measured on an i5-1235U with an NVMe disk, release build, 11 sample keywords,
no exclusion list:

| Level | Candidates | Size | Time | Throughput |
|-------|-----------:|-----:|-----:|-----------:|
| 0 focused | 16,841 | 252 KB | ~5 ms | ~3.5M/s |
| 1 standard | 28,779 | 416 KB | ~8 ms | ~3.4M/s |
| 2 extended | 53,726 | 782 KB | ~15 ms | ~3.6M/s |
| 3 deep | 105,077 | 1.5 MB | ~26 ms | ~4.0M/s |

Two-keyword run (`corp`, `corp.local`): 1,033 candidates / 13.9 KB in ~1 ms.
AD-filtered 11-keyword run (`-min 8 -ad-policy`): 12,797 candidates in ~5 ms.

Generation is intentionally single-threaded to keep output deterministic.
The bottleneck at large sizes is string generation and hashing, not disk.
Exclusion loading is the slow path: a 5M-line synthetic sorted list loads in
~0.6 s and raises RSS to ~90 MB; 14M lines load in ~1.7 s at ~245 MB.

---

## Compatibility with wordlist_generator.exe

| Capability | Old tool | v2.0 |
|------------|----------|------|
| Keyword input | `-L` / default `inputs.txt` | `-L` / `-input`; same fallback |
| Output | `-o` / default `outputs.txt` | `-o` / `-output`; default `custom_wordlist.txt` |
| Depth | `-lvl` | `-lvl` / `-level` (0–3) |
| Combination depth | `-perm` | `-perm` (1–3) |
| Character classes | `-pattern` | `-pattern` (same letters) |
| Verbose words | `-v` | `-v` (echo to stdout) |
| Case variants | yes | yes, controlled |
| Year + special shapes | yes, static years | yes, dynamic years |
| Separated numbers (`corp_24`) | yes | yes |
| Cross-keyword combinations | yes | yes, bounded by `-pair-limit` |
| Target dates | no | yes (`-dates`) |
| Doubled words (`Corpcorp`) | yes | yes |
| Prefix symbols (`!Corp`) | yes | no (intentional: low-probability noise) |
| Broken mid-word shapes (`corp.loca*l`) | v1.1 only | no |
| AD policy filter | no | optional (`-ad-policy`) |
| Exclusion lists | no | yes |
| Username pivots | no | yes |
| Rich user data (name + birth year) | no | yes (`-userdata`) |
| Turkish case/fold | no | yes |
| Deterministic output | no | yes |
| Exact dedup | no | yes |

The old tool's output for `corp` / `corp.local` contained 390 words. v2 focused
contains 1,033, covering the high-value shapes from the old list with dynamic
years instead of the hardcoded 2023/2024, plus additional coverage without the
combinatorial noise. The rare fully mixed pair products (`CORPcorp`,
`corp.localCORP`) are available at deep level.

---

## Project structure

```
WordlistKingdom/
├── main.go                  CLI entry point, signal handling
├── internal/
│   ├── app/                 end-to-end pipeline, statistics, report
│   └── cli/                 flags, aliases, validation, help
├── pkg/
│   ├── config/              configuration, pattern and list parsing
│   ├── source/              keyword/user/exclude readers, gzip, stdin
│   ├── textutil/            case forms, ASCII fold, Turkish casing
│   ├── dedup/               exact 64-bit hash set, exclusion loader
│   ├── policy/              length/whitespace/AD complexity filter
│   ├── generate/            tiered candidate generator
│   └── output/              buffered writer, echo, error propagation
├── docs/DESIGN.md           design notes and measured comparison
├── keywords.txt             sample keyword file
├── Makefile
└── LICENSE
```

---

## Development

```bash
make test          # go test ./...
make race          # go test -race ./...
make vet           # go vet ./...
make bench         # generator benchmark
make fuzz          # short fuzz sessions for parsers and policy
make fmt           # gofmt -w .
make release       # tests + cross-build + dist/SHA256SUMS
```

The test suite covers:

- case, Turkish and ASCII-fold forms;
- AD complexity (3 of 4 classes), length bounds, separators, whitespace;
- exact dedup growth and gzip exclusion;
- reader BOM/CRLF/comments/quotes/usernames;
- deterministic output, level supersets, permutation gating, patterns, caps;
- username pivots, deep-level shapes, pair limits, junk-shape regressions;
- end-to-end runs: exclusion, stdout, caps, cancellation, echo, no-output errors;
- fuzz targets for date/pattern/special parsing, keyword/userdata readers,
  case forms and policy checks;
- a Windows CI job that runs the test suite and a real binary smoke test.

---

## Integration

```bash
# Hashcat, straight wordlist
hashcat -m 1000 ntlm.txt corp_wordlist.txt

# Hydra
hydra -L users.txt -P corp_wordlist.txt ssh://10.0.0.10

# NetExec / CrackMapExec SMB password spray
netexec smb 10.0.0.0/24 -u users.txt -p corp_wordlist.txt --continue-on-success

# Kerbrute password spray
kerbrute passwordspray -d corp.local --dc 10.0.0.5 users.txt corp_wordlist.txt
```

For spraying, keep `-lvl 0`, set `-min 8 -ad-policy` when the domain policy is
known, and consider `-pair-limit 0` if lockout risk is high.

---

## Troubleshooting

**`no usable keywords`** — the input file is empty, comments only, or every
line was skipped (too long, control characters, duplicates). Check the file
encoding and line endings.

**The output is larger than expected** — lower the level, set
`-pair-limit 0`, or cap with `-max-candidates`. Use `-v` to see the phase
breakdown.

**Few candidates survive `-ad-policy`** — AD requires 3 of 4 classes, not all
four. `-min 8 -ad-policy` keeps `Corp2026` (upper+lower+digit) and
`corp2026!` (lower+digit+special); it drops `corp2026` and `CORP2026`.

**Turkish characters look wrong** — use `-tr` and/or `-ascii`. Both the source
spelling and the ASCII-folded spelling are generated; the policy only keeps
what you allow.

**Exclusion is slow or uses too much memory** — that is the cost of exact
exclusion. Use a machine with enough RAM, or split the exclusion list.

**Output goes to stdout together with the banner** — it does not: the banner
and summary always go to stderr. If you see status text in a file, you likely
redirected stderr into it.

---

## License

MIT — see [LICENSE](LICENSE).
