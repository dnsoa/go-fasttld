// go-fasttld is a high performance top level domains (TLD)
// extraction module implemented with compressed tries.
//
// This module is a port of the Python fasttld module,
// with additional modifications to support extraction
// of subcomponents from full URLs and IPv4 addresses.
package fasttld

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/afero"
	"golang.org/x/net/idna"
)

var publicSuffixListSources = []string{
	"https://publicsuffix.org/list/public_suffix_list.dat",
	"https://raw.githubusercontent.com/publicsuffix/list/master/public_suffix_list.dat",
}

// Return true if `maybeIPv4Address` is an IPv4 address
func looksLikeIPv4Address(maybeIPv4Address string) bool {
	return net.ParseIP(maybeIPv4Address) != nil
}

// Extract Public Suffixes and Private Suffixes from Public Suffix list located at `cacheFilePath`
//
// PublicSuffixes: ICANN domains. Example: com, net, org etc.
//
// PrivateSuffixes: PRIVATE domains. Example: blogspot.co.uk, appspot.com etc.
//
// AllSuffixes: Both ICANN and PRIVATE domains.
func getPublicSuffixList(cacheFilePath string) ([3]([]string), error) {
	PublicSuffixes := []string{}
	PrivateSuffixes := []string{}
	AllSuffixes := []string{}

	fd, err := os.Open(cacheFilePath)
	if err != nil {
		log.Println(err)
		return [3]([]string){PublicSuffixes, PrivateSuffixes, AllSuffixes}, err
	}
	defer fd.Close()

	fileScanner := bufio.NewScanner(fd)
	fileScanner.Split(bufio.ScanLines)
	isPrivateSuffix := false
	for fileScanner.Scan() {
		line := strings.TrimSpace(fileScanner.Text())
		if "// ===BEGIN PRIVATE DOMAINS===" == line {
			isPrivateSuffix = true
		}
		if len(line) == 0 || strings.HasPrefix(line, "//") {
			continue
		}
		suffix, err := idna.ToASCII(line)
		if err != nil {
			// skip line if unable to convert to ascii
			log.Println(line, '|', err)
			continue
		}
		if isPrivateSuffix {
			PrivateSuffixes = append(PrivateSuffixes, suffix)
			if suffix != line {
				// add non-punycode version if it is different from punycode version
				PrivateSuffixes = append(PrivateSuffixes, line)
			}
		} else {
			PublicSuffixes = append(PublicSuffixes, suffix)
			if suffix != line {
				// add non-punycode version if it is different from punycode version
				PublicSuffixes = append(PublicSuffixes, line)
			}
		}
		AllSuffixes = append(AllSuffixes, suffix)
		if suffix != line {
			// add non-punycode version if it is different from punycode version
			AllSuffixes = append(AllSuffixes, line)
		}

	}
	return [3]([]string){PublicSuffixes, PrivateSuffixes, AllSuffixes}, nil
}

// Download file from url as byte slice
func downloadFile(url string) ([]byte, error) {
	// Make HTTP GET request
	var bodyBytes []byte
	resp, err := http.Get(url)
	if err != nil {
		return bodyBytes, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err = io.ReadAll(resp.Body)
	} else {
		err = errors.New("Download failed, HTTP status code :" + fmt.Sprint(resp.StatusCode))
	}
	return bodyBytes, err
}

// Get path to current module file
//
// Similar to os.path.dirname(os.path.realpath(__file__)) in Python
//
// Credits: https://andrewbrookins.com/tech/golang-get-directory-of-the-current-file
func getCurrentFilePath() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Cannot get current module file path")
	}
	return filepath.Dir(file)
}

// Update local cache of Public Suffix List
func update(file afero.File,
	publicSuffixListSources []string) error {
	download_success := false
	for _, publicSuffixListSource := range publicSuffixListSources {
		// Write GET request body to local file
		if bodyBytes, err := downloadFile(publicSuffixListSource); err != nil {
			log.Println(err)
		} else {
			file.Seek(0, 0)
			file.Write(bodyBytes)
			download_success = true
			break
		}
	}
	if download_success {
		log.Println("Public Suffix List updated.")
	} else {
		return errors.New("Failed to fetch any Public Suffix List from all mirrors.")
	}

	return nil
}

// If Public Suffix List is not custom, update its local cache
func (t *fastTLD) Update() error {
	if t.cacheFilePath != getCurrentFilePath()+string(os.PathSeparator)+defaultPSLFileName {
		return errors.New("Update() only applies to default Public Suffix List, not custom Public Suffix List.")
	}
	// Create local file at cacheFilePath
	if file, err := os.Create(t.cacheFilePath); err != nil {
		return err
	} else {
		defer file.Close()
		return update(file, publicSuffixListSources)
	}
}
