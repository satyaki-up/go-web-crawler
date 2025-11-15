package crawler

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type Crawler struct {
	maxWorkers int
	maxDepth   int
	timeout    time.Duration
	visited    *sync.Map
	client     *http.Client
	rateLimit  <-chan time.Time
	robots     *Robots
}

func New(maxWorkers, maxDepth int, timeout time.Duration, requestsPerSecond int) *Crawler {
	var rateLimit <-chan time.Time
	if requestsPerSecond > 0 {
		rateLimit = time.Tick(time.Second / time.Duration(requestsPerSecond))
	}
	
	return &Crawler{
		maxWorkers: maxWorkers,
		maxDepth:   maxDepth,
		timeout:    timeout,
		visited:    &sync.Map{},
		client: &http.Client{
			Timeout: timeout,
		},
		rateLimit: rateLimit,
	}
}

func (c *Crawler) Crawl(startURL string) map[string]bool {
	results := make(map[string]bool)
	var mu sync.Mutex
	
	parsedURL, err := url.Parse(startURL)
	if err != nil {
		fmt.Printf("Error parsing URL: %v\n", err)
		return results
	}
	baseURL := parsedURL.Scheme + "://" + parsedURL.Host
	allowedHost := parsedURL.Host
	
	robots, err := NewRobots(baseURL, c.client)
	if err != nil {
		fmt.Printf("Warning: Failed to load robots.txt: %v\n", err)
		robots = &Robots{group: nil}
	}
	c.robots = robots
	fmt.Printf("robots: %v \n", robots)
	
	urlChan := make(chan crawlTask, 100)
	done := make(chan struct{})
	var pendingWork sync.WaitGroup
	
	var wg sync.WaitGroup
	for i := 0; i < c.maxWorkers; i++ {
		wg.Add(1)
		go c.worker(urlChan, done, &wg, baseURL, allowedHost, &mu, results, &pendingWork)
	}
	
	pendingWork.Add(1)
	urlChan <- crawlTask{url: startURL, depth: 0}
	
	go func() {
		pendingWork.Wait()
		time.Sleep(500 * time.Millisecond)
		close(done)
		close(urlChan)
	}()
	
	wg.Wait()
	return results
}

type crawlTask struct {
	url   string
	depth int
}

func (c *Crawler) worker(urlChan chan crawlTask, done <-chan struct{}, wg *sync.WaitGroup, baseURL string, allowedHost string, mu *sync.Mutex, results map[string]bool, pendingWork *sync.WaitGroup) {
	defer wg.Done()
	
	for {
		select {
		case <-done:
			return
		case task, ok := <-urlChan:
			if !ok {
				return
			}
			
			if task.depth > c.maxDepth {
				pendingWork.Done()
				continue
			}
			
			if _, visited := c.visited.LoadOrStore(normalizeURL(task.url), true); visited {
				pendingWork.Done()
				continue
			}
			
			if !c.robots.IsAllowed(task.url) {
				pendingWork.Done()
				continue
			}
			
			links, err := c.fetchAndParse(task.url, baseURL)
			if err != nil {
				fmt.Printf("Error fetching %s: %v\n", task.url, err)
				pendingWork.Done()
				continue
			}
			
			mu.Lock()
			results[task.url] = true
			mu.Unlock()
			
			fmt.Printf("[Depth %d] Visited: %s (found %d links)\n", task.depth, task.url, len(links))
			
			for _, link := range links {
				select {
				case <-done:
					pendingWork.Done()
					return
				default:
				}
				
				parsedLink, err := url.Parse(link)
				if err != nil || parsedLink.Host != allowedHost {
					continue
				}
				
				normalizedLink := normalizeURL(link)
				if _, visited := c.visited.Load(normalizedLink); visited {
					continue
				}
				
				if !c.robots.IsAllowed(link) {
					continue
				}
				
				select {
				case <-done:
					pendingWork.Done()
					return
				case urlChan <- crawlTask{url: link, depth: task.depth + 1}:
					pendingWork.Add(1)
				default:
				}
			}
			
			pendingWork.Done()
		}
	}
}


