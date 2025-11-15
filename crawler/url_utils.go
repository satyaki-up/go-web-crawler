package crawler

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

func (c *Crawler) fetchAndParse(targetURL, baseURL string) ([]string, error) {
	if c.rateLimit != nil {
		<-c.rateLimit
	}
	
	resp, err := c.client.Get(targetURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	return c.extractLinks(string(body), baseURL), nil
}

func (c *Crawler) extractLinks(html, baseURL string) []string {
	var links []string
	seen := make(map[string]bool)
	
	hrefRegex := regexp.MustCompile(`(?i)href\s*=\s*["']([^"']+)["']`)
	matches := hrefRegex.FindAllStringSubmatch(html, -1)
	
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		
		link := strings.TrimSpace(match[1])
		if link == "" || link == "#" {
			continue
		}
		
		absoluteURL, err := c.resolveURL(link, baseURL)
		if err != nil {
			continue
		}
		
		if !seen[absoluteURL] {
			seen[absoluteURL] = true
			links = append(links, absoluteURL)
		}
	}
	
	return links
}

func normalizeURL(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return u
	}
	parsed.Fragment = ""
	parsed.RawQuery = ""
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	return parsed.String()
}

func (c *Crawler) resolveURL(link, baseURL string) (string, error) {
	parsedLink, err := url.Parse(link)
	if err != nil {
		return "", err
	}
	
	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	
	absoluteURL := parsedBase.ResolveReference(parsedLink)
	return absoluteURL.String(), nil
}

