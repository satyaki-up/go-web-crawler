package crawler

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/temoto/robotstxt"
)

type Robots struct {
	group *robotstxt.Group
}

func NewRobots(baseURL string, client *http.Client) (*Robots, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("Invalid base URL: %w", err)
	}

	robotsURL := fmt.Sprintf("%s://%s/robots.txt", parsedURL.Scheme, parsedURL.Host)

	resp, err := client.Get(robotsURL)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch robots.txt: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &Robots{group: nil}, nil
	}

	robotsData, err := robotstxt.FromResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse robots.txt: %w", err)
	}

	group := robotsData.FindGroup("*")
	return &Robots{group: group}, nil
}

func (r *Robots) IsAllowed(targetURL string) bool {
	if r.group == nil {
		// fmt.Printf("No robots.txt found for targetURL: %v \n", targetURL)
		return true
	}

	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return false
	}

	path := parsedURL.Path
	if path == "" {
		path = "/"
	}

	// if r.group.Test(path) {
	// 	fmt.Printf("Allowed: %v for targetURL: %v \n", path, targetURL)
	// } else {
	// 	fmt.Printf("Not allowed: %v for targetURL: %v \n", path, targetURL)
	// }
	return r.group.Test(path)
}
