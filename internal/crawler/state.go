package crawler

import "sync"

// State tracks visited and queued URLs for a crawl job.
type State struct {
	mu       sync.Mutex
	visited  map[string]struct{}
	queued   []queueEntry
	head     int
	total    int
	complete int
}

type queueEntry struct {
	URL   string
	Depth int
}

// NewState creates a new crawl state tracker.
func NewState() *State {
	return &State{
		visited: make(map[string]struct{}),
	}
}

// Enqueue adds a URL to the crawl queue if not already visited.
func (s *State) Enqueue(url string, depth int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.visited[url]; ok {
		return false
	}
	s.visited[url] = struct{}{}
	s.queued = append(s.queued, queueEntry{URL: url, Depth: depth})
	s.total++
	return true
}

// Dequeue returns the next URL to crawl, or empty string if queue is exhausted.
func (s *State) Dequeue() (string, int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.head >= len(s.queued) {
		return "", 0, false
	}
	entry := s.queued[s.head]
	s.head++
	return entry.URL, entry.Depth, true
}

// MarkComplete increments the completed counter.
func (s *State) MarkComplete() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.complete++
}

// Stats returns total discovered and completed counts.
func (s *State) Stats() (total int, complete int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.total, s.complete
}
