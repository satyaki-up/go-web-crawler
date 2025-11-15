package main

import (
	"flag"
	"fmt"
	"time"

	"go-web-crawler/crawler"
)

func main() {
	startURL := "https://crawlme.monzo.com/"
	maxWorkers := flag.Int("workers", 5, "Number of concurrent workers")
	maxDepth := flag.Int("depth", 2, "Maximum crawl depth")
	timeout := flag.Duration("timeout", 10*time.Second, "Request timeout")
	rateLimit := flag.Int("rate", 2, "Requests per second")
	flag.Parse()

	c := crawler.New(*maxWorkers, *maxDepth, *timeout, *rateLimit)
	
	fmt.Printf("Starting crawl of %s with %d workers (max depth: %d)\n", startURL, *maxWorkers, *maxDepth)
	
	results := c.Crawl(startURL)
	
	fmt.Printf("\n=== Crawl Results ===\n")
	fmt.Printf("Total URLs visited: %d\n", len(results))
	
	for url := range results {
		fmt.Println(url)
	}
}

