package main

import (
	"fmt"
	"net/http"
	"io"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

func main() {
	domains := []string{"https://github.com/fabritsius", "https://fabritsius.github.io/"}
	seeds := []string{"https://github.com/fabritsius?tab=repositories"}

	crawlDomains(domains, seeds)
}

// crawlDomains starts crawl function for each element in seeds
func crawlDomains(domains []string, seeds []string) {
	crawled := make(map[string]struct{})

	var wg sync.WaitGroup
	
	for _, uri := range seeds {
		wg.Add(1)
		go crawl(uri, 2, domains, crawled, &wg)
	}

	wg.Wait()
	
	for uri := range crawled {
		fmt.Println(uri)
	}
}

// crawl function crawls the page and
// recursively calls crawl for every valid uri on the page
func crawl(uri string, depth int, domains []string,
		   crawled map[string]struct{}, wg *sync.WaitGroup) {
	
	defer wg.Done()

	content, err := buildIndex(uri)
	if err != nil {
		return
	}

	crawled[content.uri] = struct{}{}

	if depth > 0 {	
		for _, next := range content.links {
			if strings.HasPrefix(next, "/") {
				next = strings.Join(strings.Split(content.uri, "/")[0:3], "/") + next
			}
			if validDomain(next, domains) {
				if _, exists := crawled[next]; !exists {					
					wg.Add(1)
					go crawl(next, depth - 1, domains, crawled, wg)
				}
			}
		}
	}
}

// validDomain checks if a link is to any an allowed domain
func validDomain(link string, domains []string) bool {
	for _, d := range domains {
		if strings.HasPrefix(link, d) {
			return true
		}
	}

	return false
}

// webPage represents a content of a web page
type webPage struct {
	uri   string
	words map[string]struct{}
	links []string
}

// addWords fills words "set" with from input slice
func (p *webPage) addWords(words []string) error {
	for _, word := range words {
		if word != "" {
			wordLower := strings.ToLower(word)
			p.words[wordLower] = struct{}{}
		}
	}
	
	return nil
}

// buildIndex fetches the uri and
// returns it's content as a pointer to a webPage struct
func buildIndex(uri string) (*webPage, error) {
	resp, err := http.Get(uri)
	if err != nil {
		return nil, fmt.Errorf("Can't fetch %s", uri)
	}	

	tokenizer := html.NewTokenizer(resp.Body)
	defer resp.Body.Close()

	var content webPage
	content.uri = uri
	content.words = make(map[string]struct{})
	recording := true

	splitExpr := regexp.MustCompile(`\W`)

	for {
		tokenType := tokenizer.Next()
		
		btag, hasAttr := tokenizer.TagName()
		tag := string(btag)

		switch tokenType {
		case html.ErrorToken:
			err := tokenizer.Err()
			if err != io.EOF {
				fmt.Println(err)
			}
			return &content, nil
		case html.StartTagToken:
			if tag == "a" && hasAttr {
				attrs := getAttrVals(tokenizer)
				content.links = append(content.links, attrs["href"])
			} else if tag == "script" {
				recording = false
			}
		case html.EndTagToken:
			if tag == "script" {
				recording = true
			}
		case html.TextToken:
			if recording {
				words := splitExpr.Split(string(tokenizer.Raw()), -1)
				content.addWords(words)
			}
		}
	}
}

// getAttrVals function returns map with tag attributes
func getAttrVals(t *html.Tokenizer) map[string]string {
	var result = make(map[string]string)
	for {
		attrName, attrVal, moreAttr := t.TagAttr()
		result[string(attrName)] = string(attrVal)
		if !moreAttr {
			return result
		}
	}
}