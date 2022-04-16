package fasttld

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/idna"
)

const defaultPSLFileName string = "public_suffix_list.dat"

// Hashmap with keys as strings
type dict map[string]interface{}

type FastTLD struct {
	TldTrie       dict
	cacheFilePath string
}

type ExtractResult struct {
	SubDomain, Domain, Suffix, RegisteredDomain string
}

type SuffixListParams struct {
	CacheFilePath        string
	IncludePrivateSuffix bool
}

type UrlParams struct {
	Url                  string
	IgnoreSubDomains     bool
	ConvertURLToPunyCode bool
}

// credits: https://stackoverflow.com/questions/13687924 and https://github.com/jophy/fasttld
func nestedDict(dic dict, keys []string) {
	if len(keys) == 0 {

	} else if len(keys) == 1 {
		dic[keys[0]] = true

	} else {
		keys_ := keys[0 : len(keys)-1]

		var dic_ interface{}

		dic_ = dic

		for _, key := range keys_ {
			if dic_bk, isDict := dic_.(dict); isDict {
				if _, ok := dic_bk[key]; !ok { // dic_[key] does not exist
					dic_bk[key] = dict{} // set to dict{}
					dic_ = dic_bk[key]   // point dic_ to it
				} else { // dic_[key] exists
					if _, isDict := dic_bk[key].(dict); isDict {
						dic_ = dic_bk[key] // point dic_ to it
					} else {
						// it's a boolean
						dic_ = dic_bk
						dic_bk[keys[len(keys)-2]] = dict{"_END": true, keys[len(keys)-1]: true}
					}
				}
			}
		}
		if val, isDict := dic_.(dict); isDict {
			val[keys[len(keys)-1]] = true
		}
	}
}

// Reverse a slice of strings in-place
func reverse(input []string) {
	for i, j := 0, len(input)-1; i < j; i, j = i+1, j-1 {
		input[i], input[j] = input[j], input[i]
	}
}

// Format string as punycode
func formatAsPunycode(s string) string {
	asPunyCode, err := idna.ToASCII(strings.ToLower(strings.TrimSpace(s)))
	if err != nil {
		log.Println(strings.SplitAfterN(err.Error(), "idna: invalid label", 2)[0])
		return ""
	}

	return asPunyCode
}

// Construct a compressed trie to store Public Suffix List TLDs split at "." in reverse-order
//
// For example: "us.gov.pl" will be stored in the order {"pl", "gov", "us"}
func trieConstruct(includePrivateSuffix bool, cacheFilePath string) dict {
	tldTrie := dict{}
	suffixLists := getPublicSuffixList(cacheFilePath)

	SuffixList := []string{}
	if !includePrivateSuffix {
		// public suffixes only
		SuffixList = suffixLists[0]
	} else {
		// public suffixes AND private suffixes
		SuffixList = suffixLists[2]
	}

	for _, suffix := range SuffixList {
		if strings.Contains(suffix, ".") {
			sp := strings.Split(suffix, ".")
			reverse(sp)
			nestedDict(tldTrie, sp)
		} else {
			tldTrie[suffix] = dict{"_END": true}
		}
	}

	for key := range tldTrie {
		if val, ok := tldTrie[key].(dict); ok {
			if len(val) == 1 {
				if _, ok := tldTrie["_END"]; ok {
					tldTrie[key] = true
				}
			}
		}
	}
	return tldTrie
}

// Extract subdomain, domain, suffix and registered domain from a given `url`
//
//  Example: "https://maps.google.com.ua/a/long/path?query=42"
//  subdomain: maps
//  domain: google
//  suffix: com.ua
//  registered domain: maps.google.com.ua
func (f *FastTLD) Extract(e UrlParams) *ExtractResult {
	urlParts := ExtractResult{}

	if e.ConvertURLToPunyCode {
		e.Url = formatAsPunycode(e.Url)
	}

	// Remove URL scheme and everything after the URL host subcomponent
	// Credits: https://github.com/mjd2021usa/tldextract/blob/main/tldextract.go
	netloc := e.Url
	netloc = schemeRegex.ReplaceAllString(netloc, "")

	atIdx := strings.Index(netloc, "@")
	if atIdx != -1 {
		netloc = netloc[atIdx+1:]
	}

	index := strings.IndexFunc(netloc, func(r rune) bool {
		switch r {
		case ':', '/', '?', '&', '#':
			return true
		}
		return false
	})
	if index != -1 {
		netloc = netloc[0:index]
	}

	// Determine if url is an IPv4 address
	if looksLikeIPv4Address(netloc) {
		urlParts.Domain = netloc
		urlParts.RegisteredDomain = netloc
		return &urlParts
	}

	labels := strings.Split(netloc, ".")

	var node dict
	// define the root node
	node = f.TldTrie

	lenSuffix := 0
	lenLabels := len(labels)
	for idx := range labels {
		label := labels[lenLabels-idx-1]
		/*
			if node_, isBool := node.(bool); isBool && node_ == true {
				// this node is an end node.
				urlParts.Domain = label
				break
			}
		*/

		// this node has sub-nodes and maybe an end-node.
		// eg. cn -> (cn, gov.cn)
		if _, ok := node["_END"]; ok {
			// check if there is a sub node
			// eg. gov.cn
			if val, ok := node[label]; ok {
				lenSuffix += 1
				if val, ok := val.(dict); ok {
					node = val
					continue
				} else {
					urlParts.Domain = labels[idx-1]
					break
				}
			}
		}

		if _, ok := node["*"]; ok {
			// check if there is a sub node
			// e.g. www.ck
			var sb strings.Builder
			sb.Grow(1 + len(label))
			sb.WriteString("!")
			sb.WriteString(label)
			if _, ok := node[sb.String()]; ok {
				urlParts.Domain = label
			} else {
				lenSuffix += 1
			}
			break
		}

		// check if TLD in Public Suffix List
		if val, ok := node[label]; ok {
			lenSuffix += 1
			if val_, ok := val.(dict); ok {
				node = val_
			} else {
				urlParts.Domain = labels[idx-1]
				break
			}
		} else {
			break
		}

	}

	urlParts.Suffix = strings.Join(labels[lenLabels-lenSuffix:], ".")

	len_url_suffix := len(urlParts.Suffix)
	len_url_domain := 0

	if 0 < lenSuffix && lenSuffix < lenLabels {
		urlParts.Domain = labels[lenLabels-lenSuffix-1]
		len_url_domain = len(urlParts.Domain)
		if !e.IgnoreSubDomains && (lenLabels-lenSuffix) >= 2 {
			urlParts.SubDomain = netloc[:len(netloc)-len_url_domain-len_url_suffix-2]
			//urlParts.SubDomain = strings.Join(labels[0:lenLabels-lenSuffix-1], ".")
		}
	}

	if len_url_domain > 0 && len_url_suffix > 0 {
		urlParts.RegisteredDomain = urlParts.Domain + "." + urlParts.Suffix
	}

	return &urlParts
}

// New creates a new *FastTLD
func New(n SuffixListParams) (*FastTLD, error) {
	cachePath := filepath.Dir(n.CacheFilePath)
	if stat, err := os.Stat(cachePath); cachePath == "." || err != nil || !stat.IsDir() {
		// if cachePath is unreachable, use default Public Suffix List
		cachePath = getCurrentFilePath()
		n.CacheFilePath = cachePath + string(os.PathSeparator) + defaultPSLFileName

		// Download new Public Suffix List if local cache does not exist
		// or if local cache is older than 3 days
		autoUpdate(n.CacheFilePath)
	}

	// Construct tldTrie using list located at n.CacheFilePath
	tldTrie := trieConstruct(n.IncludePrivateSuffix, n.CacheFilePath)

	return &FastTLD{tldTrie, n.CacheFilePath}, nil
}
