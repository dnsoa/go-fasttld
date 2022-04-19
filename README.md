# go-fasttld

[![Go Reference](https://img.shields.io/badge/go-reference-blue?logo=go&logoColor=white&style=for-the-badge)](https://pkg.go.dev/github.com/elliotwutingfeng/go-fasttld)
[![Go Report Card](https://goreportcard.com/badge/github.com/elliotwutingfeng/go-fasttld?style=for-the-badge)](https://goreportcard.com/report/github.com/elliotwutingfeng/go-fasttld)
[![Codecov Coverage](https://img.shields.io/codecov/c/github/elliotwutingfeng/go-fasttld?color=bright-green&logo=codecov&style=for-the-badge&token=GB00MYK51E)](https://codecov.io/gh/elliotwutingfeng/go-fasttld)

[![GitHub license](https://img.shields.io/badge/LICENSE-BSD--3--CLAUSE-GREEN?style=for-the-badge)](LICENSE)

**go-fasttld** is a high performance [top level domains (TLD)](https://en.wikipedia.org/wiki/Top-level_domain) extraction module implemented with [compressed tries](https://en.wikipedia.org/wiki/Trie).

This module is a port of the Python [fasttld](https://github.com/jophy/fasttld) module, with additional modifications to support extraction of subcomponents from full URLs and IPv4 addresses.

![Trie](Trie_example.svg)

## Background

**go-fasttld** extracts subcomponents like [top level domains (TLDs)](https://en.wikipedia.org/wiki/Top-level_domain), subdomains and hostnames from [URLs](https://en.wikipedia.org/wiki/URL) efficiently by using the regularly-updated [Mozilla Public Suffix List](http://www.publicsuffix.org) and the [compressed trie](https://en.wikipedia.org/wiki/Trie) data structure.

For example, it extracts the `com` TLD, `maps` subdomain, and `google` domain from `https://maps.google.com:8080/a/long/path/?query=42`.

**go-fasttld** also supports extraction of private domains listed in the [Mozilla Public Suffix List](http://www.publicsuffix.org) like 'blogspot.co.uk' and 'sinaapp.com', and extraction of IPv4 addresses (e.g. `https://127.0.0.1`).

### Why not split on "." and take the last element instead?

Splitting on "." and taking the last element only works for simple TLDs like `.com`, but not more complex ones like `oseto.nagasaki.jp`.

## Installation

```sh
go get github.com/elliotwutingfeng/go-fasttld
```

## Quick Start

Full demo available in the _examples_ folder

```go
// Initialise fasttld extractor
extractor, _ := fasttld.New(fasttld.SuffixListParams{})

//Extract URL subcomponents
url := "https://some-user@a.long.subdomain.ox.ac.uk:5000/a/b/c/d/e/f/g/h/i?id=42"
res := extractor.Extract(fasttld.UrlParams{Url: url})

// Display results
fmt.Println(res.Scheme)           // https://
fmt.Println(res.UserInfo)         // some-user
fmt.Println(res.SubDomain)        // a.long.subdomain
fmt.Println(res.Domain)           // ox
fmt.Println(res.Suffix)           // ac.uk
fmt.Println(res.RegisteredDomain) // ox.ac.uk
fmt.Println(res.Port) // 5000
fmt.Println(res.Path) // a/b/c/d/e/f/g/h/i?id=42
```

## Public Suffix List options

### Specify custom public suffix list file

You can use a custom public suffix list file by setting `CacheFilePath` in `fasttld.SuffixListParams{}` to its absolute path.

```go
cacheFilePath := "/absolute/path/to/file.dat"
extractor, _ := fasttld.New(fasttld.SuffixListParams{CacheFilePath: cacheFilePath})
```

### Updating the default Public Suffix List cache

Whenever `fasttld.New` is called without specifying `CacheFilePath` in `fasttld.SuffixListParams{}`, the local cache of the default Public Suffix List is updated automatically if it is more than 3 days old. You can also manually update the cache by using `Update()`.

```go
// Automatic update performed if `CacheFilePath` is not specified
// and local cache is more than 3 days old
extractor, _ := fasttld.New(fasttld.SuffixListParams{})

// Manually update local cache
if err := extractor.Update(); err != nil {
    log.Println(err)
}
```

### Private domains

According to the [Mozilla.org wiki](https://wiki.mozilla.org/Public_Suffix_List/Uses), the Mozilla Public Suffix List contains private domains like `blogspot.com` and `sinaapp.com`.

By default, **go-fasttld** _excludes_ these private domains (i.e. `IncludePrivateSuffix = false`)

```go
extractor, _ := fasttld.New(fasttld.SuffixListParams{})

url := "https://google.blogspot.com"
res := extractor.Extract(fasttld.UrlParams{Url: url})

// res.Scheme = https://
// res.UserInfo = <no output>
// res.SubDomain = google
// res.Domain = blogspot
// res.Suffix = com
// res.RegisteredDomain = blogspot.com
// res.Port = <no output>
// res.Path = <no output>
```

You can _include_ private domains by setting `IncludePrivateSuffix = true`

```go
extractor, _ := fasttld.New(fasttld.SuffixListParams{IncludePrivateSuffix: true})

url := "https://google.blogspot.com"
res := extractor.Extract(fasttld.UrlParams{Url: url})

// res.Scheme = https://
// res.UserInfo = <no output>
// res.SubDomain = <no output>
// res.Domain = google
// res.Suffix = blogspot.com
// res.RegisteredDomain = google.blogspot.com
// res.Port = <no output>
// res.Path = <no output>
```

## Extraction options

### Ignore Subdomains

You can ignore subdomains by setting `IgnoreSubDomains = true`. By default, subdomains are extracted.

```go
extractor, _ := fasttld.New(fasttld.SuffixListParams{})

url := "https://maps.google.com"
res := extractor.Extract(fasttld.UrlParams{Url: url, IgnoreSubDomains: true})

// res.Scheme = https://
// res.UserInfo = <no output>
// res.SubDomain = <no output>
// res.Domain = google
// res.Suffix = com
// res.RegisteredDomain = google.com
// res.Port = <no output>
// res.Path = <no output>
```

### Punycode

Convert internationalised URLs to [punycode](https://en.wikipedia.org/wiki/Punycode) before extraction by setting `ConvertURLToPunyCode = true`. By default, URLs are not converted to punycode.

```go
extractor, _ := fasttld.New(fasttld.SuffixListParams{})

url := "https://hello.世界.com"
res := extractor.Extract(fasttld.UrlParams{Url: url, ConvertURLToPunyCode: true})

// res.Scheme = https://
// res.UserInfo = <no output>
// res.SubDomain = hello
// res.Domain = xn--rhqv96g
// res.Suffix = com
// res.RegisteredDomain = xn--rhqv96g.com
// res.Port = <no output>
// res.Path = <no output>

res = extractor.Extract(fasttld.UrlParams{Url: url, ConvertURLToPunyCode: false})

// res.Scheme = https://
// res.UserInfo = <no output>
// res.SubDomain = hello
// res.Domain = 世界
// res.Suffix = com
// res.RegisteredDomain = 世界.com
// res.Port = <no output>
// res.Path = <no output>
```

## Testing

```sh
go test -v -coverprofile=test_coverage.out && go tool cover -html=test_coverage.out -o test_coverage.html
```

## Benchmarks

```sh
go test -bench=. -benchmem -cpu 1
```

### Modules used

| Benchmark Name                | Source                           |
|:------------------------------|:---------------------------------|
| BenchmarkGoFastTld              | go-fasttld (this module)         |
| BenchmarkJPilloraGoTld                | github.com/jpillora/go-tld       |
| BenchmarkJoeGuoTldExtract     | github.com/joeguo/tldextract     |
| BenchmarkMjd2021USATldExtract | github.com/mjd2021usa/tldextract |
| BenchmarkM507Tlde                 | github.com/M507/tlde             |

### Results

Benchmarks performed on AMD Ryzen 7 5800X, Manjaro Linux.

**go-fasttld** performs especially well on longer URLs.

---

#### #1

<code>https://news.google.com</code>

| Benchmark Name                | Iterations | ns/op       | B/op     | allocs/op   | Fastest            |
|:------------------------------|------------|-------------|----------|-------------|--------------------|
| BenchmarkGoFastTld            | 2284368    | 523.3 ns/op | 240 B/op | 6 allocs/op |                    |
| BenchmarkJPilloraGoTld        | 2661440    | 444.6 ns/op | 224 B/op | 2 allocs/op | :heavy_check_mark: |
| BenchmarkJoeGuoTldExtract     | 2400536    | 494.7 ns/op | 160 B/op | 5 allocs/op |                    |
| BenchmarkMjd2021USATldExtract | 1439263    | 841.2 ns/op | 208 B/op | 7 allocs/op |                    |
| BenchmarkM507Tlde             | 2388453    | 506.8 ns/op | 160 B/op | 5 allocs/op |                    |

---

#### #2

<code>https://iupac.org/iupac-announces-the-2021-top-ten-emerging-technologies-in-chemistry/</code>

| Benchmark Name                | Iterations | ns/op       | B/op     | allocs/op   | Fastest            |
|:------------------------------|------------|-------------|----------|-------------|--------------------|
| BenchmarkGoFastTld            | 2129161    | 565.0 ns/op | 352 B/op | 6 allocs/op | :heavy_check_mark: |
| BenchmarkJPilloraGoTld        | 1880540    | 643.9 ns/op | 224 B/op | 2 allocs/op |                    |
| BenchmarkJoeGuoTldExtract     | 1586659    | 761.4 ns/op | 288 B/op | 6 allocs/op |                    |
| BenchmarkMjd2021USATldExtract | 1488391    | 810.3 ns/op | 288 B/op | 6 allocs/op |                    |
| BenchmarkM507Tlde             | 2076404    | 572.9 ns/op | 272 B/op | 5 allocs/op |                    |

---

#### #3

<code>https://www.google.com/maps/dir/Parliament+Place,+Parliament+House+Of+Singapore,+Singapore/Parliament+St,+London,+UK/@25.2440033,33.6721455,4z/data=!3m1!4b1!4m14!4m13!1m5!1m1!1s0x31da19a0abd4d71d:0xeda26636dc4ea1dc!2m2!1d103.8504863!2d1.2891543!1m5!1m1!1s0x487604c5aaa7da5b:0xf13a2197d7e7dd26!2m2!1d-0.1260826!2d51.5017061!3e4</code>

| Benchmark Name                | Iterations | ns/op       | B/op      | allocs/op   | Fastest            |
|:------------------------------|------------|-------------|-----------|-------------|--------------------|
| BenchmarkGoFastTld            | 1678980    | 752.2 ns/op | 832 B/op  | 5 allocs/op | :heavy_check_mark: |
| BenchmarkJPilloraGoTld        | 464935     | 2571 ns/op  | 928 B/op  | 4 allocs/op |                    |
| BenchmarkJoeGuoTldExtract     | 823464     | 1341 ns/op  | 1120 B/op | 6 allocs/op |                    |
| BenchmarkMjd2021USATldExtract | 851773     | 1307 ns/op  | 1120 B/op | 6 allocs/op |                    |
| BenchmarkM507Tlde             | 856286     | 1362 ns/op  | 1120 B/op | 6 allocs/op |                    |

---

## Acknowledgements

- [fasttld (Python)](https://github.com/jophy/fasttld)
- [tldextract (Python)](https://github.com/john-kurkowski/tldextract)
- [tldextract (Go)](https://github.com/mjd2021usa/tldextract)
