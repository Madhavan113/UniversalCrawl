package crawler

import "sync"

// State tracks visited and queued URLs for a crawl job.
type State struct {
	mu       sync.Mutex
	cond     *sync.Cond
	visited  map[string]struct{}
	queued   []queueEntry
	head     int
	total    int
	inflight int
}

type queueEntry struct {
	URL   string
	Depth int
}

// NewState creates a new crawl state tracker.
func NewState() *State {
	s := &State{
		visited: make(map[string]struct{}),
	}
	s.cond = sync.NewCond(&s.mu)
	return s
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
	s.cond.Signal()
	return true
}

// Dequeue returns the next URL to crawl. It blocks while the queue is empty
// but goroutines are still in-flight (they may enqueue new URLs). Returns
// false only when the queue is empty and no workers are running.
func (s *State) Dequeue() (string, int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for s.head >= len(s.queued) {
		if s.inflight == 0 {
			return "", 0, false
		}
		s.cond.Wait()
	}
	entry := s.queued[s.head]
	s.head++
	s.inflight++
	return entry.URL, entry.Depth, true
}

// Done marks a dequeued URL as finished, decrementing the inflight counter.
func (s *State) Done() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inflight--
	s.cond.Signal()
}

// Stats returns total discovered URL count.
func (s *State) Stats() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.total
}
