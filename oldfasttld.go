package fasttld

import (
	"strconv"
	"strings"
)

// Hashmap with keys as strings
type dict map[string]interface{}

// credits: https://stackoverflow.com/questions/13687924 and https://github.com/jophy/fasttld
func OldNestedDict(dic dict, keys []string) {
	if len(keys) == 0 {
	} else if len(keys) == 1 {
		dic[keys[0]] = true
	} else {
		keys_ := keys[0 : len(keys)-1]
		len_keys := len(keys)

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
						dic_bk[keys[len_keys-2]] = dict{"_END": true, keys[len_keys-1]: true}
					}
				}
			}
		}
		if val, isDict := dic_.(dict); isDict {
			val[keys[len_keys-1]] = true
		}
	}
}

// Construct a compressed trie to store Public Suffix List TLDs split at "." in reverse-order
//
// For example: "us.gov.pl" will be stored in the order {"pl", "gov", "us"}
func oldTrieConstruct(includePrivateSuffix bool, cacheFilePath string) dict {
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
			OldNestedDict(tldTrie, sp)
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

// OldExtract subdomain, domain, suffix and registered domain from a given `url`
//
//  Example: "https://maps.google.com.ua/a/long/path?query=42"
//  subdomain: maps
//  domain: google
//  suffix: com.ua
//  registered domain: maps.google.com.ua
func (f *fastTLD) OldExtract(e UrlParams) *ExtractResult {
	urlParts := ExtractResult{}

	if e.ConvertURLToPunyCode {
		e.Url = formatAsPunycode(e.Url)
	}

	// Remove URL scheme
	// Credits: https://github.com/mjd2021usa/tldextract/blob/main/tldextract.go
	netloc := e.Url
	netloc = schemeRegex.ReplaceAllString(netloc, "")
	netloc = strings.Trim(netloc, ". \n\t\r")
	var afterHost string

	// Remove URL userinfo
	atIdx := strings.Index(netloc, "@")
	if atIdx != -1 {
		netloc = netloc[atIdx+1:]
	}

	// Separate URL host from subcomponents thereafter
	hostEndIndex := strings.IndexFunc(netloc, func(r rune) bool {
		switch r {
		case ':', '/', '?', '&', '#':
			return true
		}
		return false
	})
	if hostEndIndex != -1 {
		afterHost = netloc[hostEndIndex:]
		netloc = netloc[0:hostEndIndex]
	}

	// extract port and path if any
	if lenAfterHost := len(afterHost); lenAfterHost != 0 {
		var maybePort string
		hasPort := afterHost[0] == ':'
		pathStartIndex := strings.Index(afterHost, "/")
		var invalidPort bool
		if hasPort {
			if pathStartIndex == -1 {
				maybePort = afterHost[1:]
			} else {
				maybePort = afterHost[1:pathStartIndex]
			}
			if port, err := strconv.Atoi(maybePort); !(err == nil && 0 <= port && port <= 65535) {
				maybePort = ""
				invalidPort = true
			}
			urlParts.Port = maybePort
		}
		if !invalidPort && pathStartIndex != -1 && pathStartIndex != lenAfterHost {
			// fmt.Println(afterHost[pathStartIndex+1:])
		}
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
	node = f.OldTldTrie

	lenSuffix := 0
	suffixCharCount := 0
	lenLabels := len(labels)
	for idx := range labels {
		label := labels[lenLabels-idx-1]
		labelLength := len(label)

		// this node has sub-nodes and maybe an end-node.
		// eg. cn -> (cn, gov.cn)
		if _, ok := node["_END"]; ok {
			// check if there is a sub node
			// eg. gov.cn
			if val, ok := node[label]; ok {
				lenSuffix += 1
				suffixCharCount += labelLength
				if val, ok := val.(dict); !ok {
					urlParts.Domain = labels[idx-1]
					break
				} else {
					node = val
					continue
				}
			}
		}

		if _, ok := node["*"]; ok {
			// check if there is a sub node
			// e.g. www.ck
			if _, ok := node["!"+label]; !ok {
				lenSuffix += 1
				suffixCharCount += labelLength
			} else {
				urlParts.Domain = label
			}
			break
		}

		// check if TLD in Public Suffix List
		if val, ok := node[label]; ok {
			lenSuffix += 1
			suffixCharCount += labelLength
			if val_, ok := val.(dict); ok {
				node = val_
			} else {
				break
			}
		} else {
			break
		}

	}

	netlocLen := len(netloc)
	if lenSuffix != 0 {
		urlParts.Suffix = netloc[netlocLen-suffixCharCount-lenSuffix+1:]
	}

	len_url_suffix := len(urlParts.Suffix)
	len_url_domain := 0

	if 0 < lenSuffix && lenSuffix < lenLabels {
		urlParts.Domain = labels[lenLabels-lenSuffix-1]
		len_url_domain = len(urlParts.Domain)
		if !e.IgnoreSubDomains && (lenLabels-lenSuffix) >= 2 {
			urlParts.SubDomain = netloc[:netlocLen-len_url_domain-len_url_suffix-2]
			// urlParts.SubDomain = strings.Join(labels[0:lenLabels-lenSuffix-1], ".")
		}
	}

	if len_url_domain > 0 && len_url_suffix > 0 {
		urlParts.RegisteredDomain = urlParts.Domain + "." + urlParts.Suffix
	}

	return &urlParts
}
