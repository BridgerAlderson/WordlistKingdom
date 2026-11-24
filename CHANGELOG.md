# Changelog

All notable changes are recorded here. Versions follow semantic versioning for
the command-line interface.

## v2.2.0 — 2026-11-24

### Changed

- Complete output now lives at `-lvl 3`; there is no separate output-parity
  parameter or alias.
- Level 3 is named **complete** and includes prefix symbols, leading
  separators, the full mixed-case pair matrix, domain self-doubling, prefix
  year/number + suffix shapes and the historical years 1971, 1999, 2000, 2001,
  2019, 2023, 2024.
- Complete pair-prefix expansion is bounded to the base patterns seen in the
  original output, keeping the level usable on larger keyword files.
- Focused, standard and extended output are unchanged.

### Added

- Complete-output parity regression test covering representative original
  output shapes.
- App-level complete-output test, including leading-separator policy.

### Removed

- The separate output-parity flag and its alias.

### Notes

- Default focused profile: 1,033 candidates for `corp`/`corp.local` and
  16,841 for the 11-keyword sample.
- Complete level: 25,579 candidates for the two-keyword sample and 688,922 for
  the 11-keyword sample.
- Use `-max-candidates` or `-pair-limit` to bound complete output.

## v2.0.0 — 2026-11-24

Complete rewrite of the generator and output pipeline.

### Added

- Tiered, priority-ordered candidate generation (core, numbers, separated
  numbers, symbols, year+symbol, number+symbol, pairs, usernames, affixes,
  leetspeak, seasons/months, deep variants).
- Four depth presets: focused (`0`), standard (`1`), extended (`2`), complete (`3`).
- `-perm 1|2|3` for single terms, pairs and triples.
- `-userlist` pivots: username case forms, year/number/symbol variants, name
  order variants (`john.smith` → `jsmith`, `smithj`, `johnsmith`) and controlled
  username×keyword crosses.
- `-exclude` lists with exact matching, gzip autodetection and line/word counts.
- Sorted exact exclude index: 5M words ≈ 90 MB / 0.6 s, 14M ≈ 245 MB / 1.7 s
  (down from hundreds of MB for the open-addressed table).
- Turkish case and ASCII-fold forms; `-tr`, `-ascii`.
- `-min`, `-max`, `-ad-policy`, `-ad-min`, `-max-candidates`.
- `-years`, `-numbers`, `-specials` for engagement-specific suffixes.
- `-dates` for birthday/founding-date shapes (DDMMYYYY, YYYYMMDD, dotted,
  dashed, slashed, DDMM/MMDD); the date year also joins the year variants.
- Domain/compound keyword derivation (`acme.local` → `acme`, `acmelocal`;
  `active_directory` → `active`, `activedirectory`).
- `-userdata` CSV (`username,first,last,birthyear`) with full-name shapes
  (`john.smith`, `JohnSmith`, `JSmith`, `smithj`), birth-year variants and
  merged plain `-userlist` entries.
- PascalCase spellings at standard level (`ActiveDirectory`), camelCase at
  extended level.
- `-pair-limit`, `-triple-limit`, `-user-keywords` bounding knobs.
- `-progress`, `-quiet`, `-version`, `-v` echo.
- Signal handling: Ctrl+C flushes the partial wordlist and exits 130.
- Per-phase and per-reason statistics; `-v` prints the breakdown.
- Unit, integration, regression and race tests; fuzz targets; Makefile;
  GitHub Actions CI for Linux and Windows; tagged release workflow with
  checksums; design notes.

### Changed

- Default generation is **focused** and realistic: 1,033 candidates for
  `corp` / `corp.local`, 16,841 for the 11-keyword sample, instead of 161,455
  and 8,087,538 in v1.1.
- Output is deterministic and exactly deduplicated; the Bloom filter is gone.
- The AD filter is opt-in and implements the real 3-of-4 class rule with
  character-count lengths.
- Years are dynamic (current and previous by default) instead of a hardcoded
  historical list.
- Multi-word keywords are joined (`Acme Corp` → `AcmeCorp`); spaced forms are
  available with `-allow-space`.
- Candidate order is priority order, and the writer reports errors.
- Original-tool aliases `-L`, `-o`, `-lvl`, `-min-len`, `-perm`, `-pattern`
  and `-v` are accepted. `-min-len` maps to `-min`.

### Removed

- Full Cartesian pair case products, prefix symbols, mid-word symbols on
  domain-shaped keywords, all-character leetspeak at default level, and
  default seasons/months/affixes.
- The probabilistic Bloom deduplicator and the unbounded `combinations()`
  materialisation.
- The default-on AD filter that removed `corp`, `corp2026`, `corp.local` and
  other realistic candidates.

### Compatibility notes

- Default output file is `custom_wordlist.txt` (v1.1 also used this); the
  original executable defaulted to `outputs.txt`.
- Default input is `keywords.txt`, with automatic fallback to `inputs.txt`.
- `-workers`, `-no-filter` and `-bloom-size` are accepted but ignored.
- v1.1 scripts that relied on an implicit `-min-len 8` AD filter must pass
  `-min 8 -ad-policy` explicitly.

## v1.1 — 2026-07-04

- Added username pivots, custom years, dynamic special years, season/month
  combinations.
- Fixed stdout output support; added a Bloom false-positive warning.
- README updated with new flags.

## v1.0 — 2026-07-04

- Initial release: concurrent pipeline, case/leet/year/special/affix/pair
  mutations, AD policy filter, Bloom deduplication.
