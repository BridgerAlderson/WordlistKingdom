# WordlistKingdom

```
 ██╗    ██╗ ██╗  ██╗
 ██║    ██║ ██║ ██╔╝
 ██║ █╗ ██║ █████╔╝
 ██║███╗██║ ██╔═██╗
 ╚███╔███╔╝ ██║  ██╗
  ╚══╝╚══╝  ╚═╝  ╚═╝   WordlistKingdom v1.1
```

**Target-Specific Custom Wordlist Generator** — A high-performance, concurrent CLI tool for generating organisation-aware password candidates during authorised penetration tests and red team operations.

> **Legal Notice:** This tool is intended exclusively for authorised security testing. Using it against systems you do not have explicit written permission to test is illegal. The author assumes no liability for misuse.

---

## How It Works

WordlistKingdom reads a plain-text keyword file (`keywords.txt`) containing organisation-specific terms — company name, city, department names, project names, etc. — and applies seven layered mutation rules to produce a deduplicated, policy-filtered wordlist ready to feed into tools like `hydra`, `crackmapexec`, `kerbrute`, or `hashcat`.

### Pipeline Architecture

```
keywords.txt
     │
     ▼
 [ Reader ]  ─── loads & deduplicates keywords
     │
     ▼
 [ Generator Goroutines ]  ─── applies all 7 mutation rules concurrently
     │  (bounded channel with 100k buffer — natural backpressure, zero OOM risk)
     ▼
 [ Filter Workers × NumCPU ]
     ├── AD Policy Filter  (min length · uppercase · digit · special char)
     └── Bloom Filter      (probabilistic dedup, ~12 MB for 5M entries at 1% FP)
     │
     ▼
 [ Buffered Writer ]  ─── 64 MiB write buffer, single syscall flush
     │
     ▼
 custom_wordlist.txt
```

The pipeline never holds the full candidate set in memory. Producers block when the channel fills; consumers drain it. Memory usage is bounded and constant regardless of keyword count.

---

## Mutation Rules

### 1. Case Variations
Every keyword is emitted in four forms:

| Form | Input | Output |
|------|-------|--------|
| lowercase | `Admin` | `admin` |
| UPPERCASE | `Admin` | `ADMIN` |
| Capitalized | `Admin` | `Admin` |
| camelCase | `active_directory` | `activeDirectory` |

### 2. Leetspeak Substitution
Full character substitution applied to every case variant:

| Char | Substitute |
|------|-----------|
| `a`  | `@` |
| `e`  | `3` |
| `i`  | `1` |
| `o`  | `0` |
| `s`  | `$` |
| `l`  | `1` |

Example: `Password` → `p@$$w0rd`

### 3. Year Append / Prepend
Combined with every base mutation as both suffix and prefix, in full and 2-digit short form:

- **Dynamic:** current year ± 3 (auto-updated each run)
- **Historical:** `1071`, `1453`, `1923`, `1999`, `2000`, `2001`, `2019`

Example: `Admin` → `Admin2024`, `2024Admin`, `Admin24`, `24Admin`

### 4. Special Character Injection
Three injection modes per candidate:

| Mode | Example |
|------|---------|
| Suffix (single) | `Admin!` `Admin@` `Admin#` … |
| Prefix (single) | `!Admin` `@Admin` … |
| Complex suffix | `Admin!123` `Admin!@#` `Admin123!` … |
| Midpoint infix | `Ad!min` (words ≥ 4 chars) |

Single chars: `! @ # $ % * . _ ? -`  
Complex tokens: `!.` `!1` `!12` `!123` `!@#` `@123` `123!` `1234` `12345` `#1`

### 5. Keyword Combinations
All ordered pairs from the keyword list are cross-combined with five separators, and all case variants of each pair are emitted:

```
Separators: .  _  -  @  (none)
```

Example with keywords `Acme` + `Admin`:  
`Acme.Admin`, `acme_admin`, `ACME-ADMIN`, `AcmeAdmin`, `acme@Admin`, …

### 6. Common Prefix / Suffix Affixes
Industry-standard role and department terms are prepended and appended (with `.` and `_` separator variants):

| Prefixes | Suffixes |
|----------|----------|
| `Admin` `Test` `User` `Dev` `IT` | `Admin` `Test` `User` `Dev` `IT` |
| `Sistem` `Helpdesk` `Root` `Super` | `123` `1234` `12345` |
| `Backup` `Net` `Sec` | `0` `01` `99` `00` |

Example: `Istanbul` → `AdminIstanbul`, `Admin_Istanbul`, `Istanbul123`, `Istanbul_IT`

### 7. Season / Month Cross-Combinations
Universal season and month terms (English + Turkish) are cross-combined with every keyword, then year and special character variants are applied to each combination:

```
Seasons: Summer Winter Spring Autumn Yaz Kis Ilkbahar Sonbahar
Months:  January February … December / Ocak Subat … Aralik
```

Example with keyword `Acme`:  
`AcmeSummer2024`, `AcmeSummer!`, `WinterAcme2025`, `AcmeOcak!@#`, …

### 8. Username Pivot Mutations
When a username list is provided via `-userlist`, every username is cross-combined with every keyword across five separators, with full year and special character variants:

```
Separators: .  _  -  @  (none)
```

Example with username `jsmith` and keyword `Acme`:  
`jsmith.Acme2026`, `jsmith_acme!`, `JSMITH@Acme2025`, `JsmithAcme!@#`, …

### 9. Active Directory Policy Filter
Before any candidate is written to disk, it is validated against the configured AD complexity policy:

- Minimum length (default: **8 characters**)
- At least **1 uppercase** letter
- At least **1 digit**
- At least **1 special character**

Candidates that fail the filter are discarded. Disable with `-no-filter` to output all raw mutations.

---

## Installation

### Option A — Build from Source (recommended)

Requires [Go 1.21+](https://go.dev/dl/).

```bash
git clone https://github.com/BridgerAlderson/WordlistKingdom.git
cd WordlistKingdom
go build -ldflags="-s -w" -o wordlistkingdom .
```

### Option B — Pre-built Binaries

Cross-compiled binaries are available in the `dist/` directory:

| Platform | Binary |
|----------|--------|
| Linux x86-64 | `dist/wordlistkingdom-linux-amd64` |
| Linux ARM64 (Raspberry Pi / cloud) | `dist/wordlistkingdom-linux-arm64` |
| macOS Intel | `dist/wordlistkingdom-darwin-amd64` |
| macOS Apple Silicon | `dist/wordlistkingdom-darwin-arm64` |
| Windows x86-64 | `dist/wordlistkingdom-windows-amd64.exe` |

Make the binary executable on Unix systems:

```bash
chmod +x dist/wordlistkingdom-linux-amd64
```

### Build All Platforms at Once

```bash
GOOS=linux   GOARCH=amd64 go build -ldflags="-s -w" -o dist/wordlistkingdom-linux-amd64 .
GOOS=linux   GOARCH=arm64 go build -ldflags="-s -w" -o dist/wordlistkingdom-linux-arm64 .
GOOS=darwin  GOARCH=amd64 go build -ldflags="-s -w" -o dist/wordlistkingdom-darwin-amd64 .
GOOS=darwin  GOARCH=arm64 go build -ldflags="-s -w" -o dist/wordlistkingdom-darwin-arm64 .
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/wordlistkingdom-windows-amd64.exe .
```

---

## Usage

```
./wordlistkingdom [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-input` | `keywords.txt` | Path to the keywords file |
| `-output` | `custom_wordlist.txt` | Path to the output wordlist (use `-` for stdout) |
| `-userlist` | _(none)_ | Path to username list for pivot mutations (SAMAccountName format) |
| `-years` | _(none)_ | Extra years to include, comma-separated (e.g. `1938,2010`) |
| `-workers` | NumCPU | Number of parallel goroutines |
| `-min-len` | `8` | Minimum password length (AD policy) |
| `-no-filter` | `false` | Disable AD policy filter — output all mutations |
| `-bloom-size` | `5000000` | Expected unique candidates; tunes bloom filter accuracy |
| `-v` | `false` | Print live progress counter (entries written + elapsed time) |

### Examples

**Standard run — AD policy ON, verbose output:**
```bash
./wordlistkingdom -input keywords.txt -output passwords.txt -v
```

**Disable AD filter — all mutations regardless of complexity:**
```bash
./wordlistkingdom -no-filter -output raw_mutations.txt
```

**Custom input/output paths with explicit worker count:**
```bash
./wordlistkingdom -input /tmp/acme_keywords.txt -output /tmp/acme_wordlist.txt -workers 8
```

**Large keyword set — tune bloom filter to avoid false positives:**
```bash
./wordlistkingdom -input big_keywords.txt -bloom-size 20000000 -v
```

**Custom minimum password length:**
```bash
./wordlistkingdom -min-len 12 -v
```

**Username pivot — cross-combine a user list with keywords:**
```bash
./wordlistkingdom -input keywords.txt -userlist users.txt -output passwords.txt -v
```

**Include extra target-specific years (e.g. founding year, known incident year):**
```bash
./wordlistkingdom -input keywords.txt -years 1938,2010 -output passwords.txt -v
```

**Pipe output directly into crackmapexec (via process substitution):**
```bash
./wordlistkingdom -no-filter -output - 2>/dev/null | crackmapexec smb 192.168.1.0/24 -u users.txt -p -
```

---

## keywords.txt Format

One keyword per line. Lines starting with `#` are treated as comments and ignored. Duplicate entries are silently dropped.

```text
# Company name and variations
Acme
AcmeCorp

# Location
Istanbul
Ankara

# Departments and roles
IT
Helpdesk
Network

# Project / product names
Portal
VPN
ERP
```

**Tips for effective keywords:**
- Include the company name, abbreviation, and local-language version
- Add city, country, and founding year
- Include internal project names, product names, and department names
- Add names of key personnel if OSINT suggests password reuse
- Include the organisation's domain name without TLD

---

## Performance

Benchmarked on a 12-core machine with 11 keywords:

| Mode | Unique Passwords | Time | Speed | Output Size |
|------|-----------------|------|-------|-------------|
| AD filter ON | 36,479 | 144 ms | 252,000 /sec | 438 KB |
| AD filter OFF | 107,566 | 178 ms | 603,000 /sec | 1.2 MB |

Performance scales linearly with keyword count. The bottleneck is disk I/O at large scales; the 64 MiB write buffer keeps syscall overhead minimal.

---

## Project Structure

```
WordlistKingdom/
├── main.go                      Entry point, CLI flags, progress reporter
├── go.mod
├── keywords.txt                 Sample keyword file
└── pkg/
    ├── reader/
    │   └── reader.go            Keyword file loader (comment skip, dedup)
    ├── bloom/
    │   └── bloom.go             Thread-safe probabilistic bloom filter
    ├── filter/
    │   └── policy.go            Active Directory complexity policy filter
    ├── writer/
    │   └── writer.go            64 MiB buffered concurrent file writer
    └── mutations/
        ├── engine.go            Goroutine pipeline orchestrator
        ├── case.go              Case variation mutations
        ├── leet.go              Leetspeak substitution
        ├── year.go              Year append/prepend variants
        ├── special.go           Special character injection
        ├── combine.go           Cross-keyword combination generator
        └── affixes.go           Common prefix/suffix affixes
```

---

## Integration with Common Tools

**Hydra (SSH brute-force):**
```bash
hydra -L users.txt -P custom_wordlist.txt ssh://192.168.1.10
```

**CrackMapExec (SMB password spray):**
```bash
crackmapexec smb 192.168.1.0/24 -u users.txt -P custom_wordlist.txt
```

**Kerbrute (Kerberos pre-auth spray):**
```bash
kerbrute passwordspray -d corp.local --dc 192.168.1.5 users.txt custom_wordlist.txt
```

**Hashcat (offline cracking — convert to wordlist attack):**
```bash
hashcat -m 1000 hashes.txt custom_wordlist.txt
```

---

## Dependencies

Zero external dependencies. The entire project is built on the Go standard library:

- `hash/fnv` — bloom filter hashing
- `sync` / `sync/atomic` — goroutine coordination
- `bufio` — buffered I/O
- `math` — bloom filter sizing calculations

---

## License

MIT License — see [LICENSE](LICENSE) for details.

---

> This tool is for **authorised penetration testing only**. Always obtain explicit written permission before testing any system.
