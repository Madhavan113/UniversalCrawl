package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/madhavanp/universalcrawl/internal/models"
	bolt "go.etcd.io/bbolt"
)

var (
	bucketJobs       = []byte("jobs")
	bucketJobResults = []byte("job_results")
	bucketCache      = []byte("cache")
)

// Store defines the persistence interface for UniversalCrawl.
type Store interface {
	// Job operations
	CreateJob(job *models.CrawlJob) error
	GetJob(id string) (*models.CrawlJob, error)
	UpdateJob(job *models.CrawlJob) error
	AddJobResult(jobID string, result *models.ScrapeResult) error
	GetJobResults(jobID string, cursor int, limit int) ([]*models.ScrapeResult, int, error)

	// Cache operations
	CacheGet(key string) (*models.ScrapeResult, error)
	CacheSet(key string, result *models.ScrapeResult, ttl time.Duration) error

	// Lifecycle
	Close() error
}

type cacheEntry struct {
	Result    *models.ScrapeResult `json:"result"`
	ExpiresAt time.Time            `json:"expiresAt"`
}

// BoltStore implements Store using bbolt.
type BoltStore struct {
	db *bolt.DB
}

// NewBoltStore opens or creates a bbolt database at dataDir/universalcrawl.db.
func NewBoltStore(dataDir string) (*BoltStore, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	dbPath := filepath.Join(dataDir, "universalcrawl.db")
	db, err := bolt.Open(dbPath, 0o600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("open bolt db: %w", err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		for _, b := range [][]byte{bucketJobs, bucketJobResults, bucketCache} {
			if _, err := tx.CreateBucketIfNotExists(b); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("init buckets: %w", err)
	}
	return &BoltStore{db: db}, nil
}

// Close closes the underlying database.
func (s *BoltStore) Close() error {
	return s.db.Close()
}

// CreateJob persists a new crawl job.
func (s *BoltStore) CreateJob(job *models.CrawlJob) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		data, err := json.Marshal(job)
		if err != nil {
			return err
		}
		return tx.Bucket(bucketJobs).Put([]byte(job.ID), data)
	})
}

// GetJob retrieves a crawl job by ID.
func (s *BoltStore) GetJob(id string) (*models.CrawlJob, error) {
	var job models.CrawlJob
	err := s.db.View(func(tx *bolt.Tx) error {
		data := tx.Bucket(bucketJobs).Get([]byte(id))
		if data == nil {
			return &models.ErrNotFound{Resource: "job", ID: id}
		}
		return json.Unmarshal(data, &job)
	})
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// UpdateJob overwrites an existing crawl job.
func (s *BoltStore) UpdateJob(job *models.CrawlJob) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		data, err := json.Marshal(job)
		if err != nil {
			return err
		}
		return tx.Bucket(bucketJobs).Put([]byte(job.ID), data)
	})
}

// AddJobResult appends a scrape result to a crawl job's result list.
func (s *BoltStore) AddJobResult(jobID string, result *models.ScrapeResult) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketJobResults)
		key := []byte(jobID)

		var results []*models.ScrapeResult
		if existing := b.Get(key); existing != nil {
			if err := json.Unmarshal(existing, &results); err != nil {
				return err
			}
		}
		results = append(results, result)

		data, err := json.Marshal(results)
		if err != nil {
			return err
		}
		return b.Put(key, data)
	})
}

// GetJobResults returns a paginated slice of results for a crawl job.
func (s *BoltStore) GetJobResults(jobID string, cursor int, limit int) ([]*models.ScrapeResult, int, error) {
	var page []*models.ScrapeResult
	nextCursor := 0

	err := s.db.View(func(tx *bolt.Tx) error {
		data := tx.Bucket(bucketJobResults).Get([]byte(jobID))
		if data == nil {
			return nil
		}
		var all []*models.ScrapeResult
		if err := json.Unmarshal(data, &all); err != nil {
			return err
		}
		if cursor >= len(all) {
			return nil
		}
		end := cursor + limit
		if end > len(all) {
			end = len(all)
		}
		page = all[cursor:end]
		if end < len(all) {
			nextCursor = end
		}
		return nil
	})
	return page, nextCursor, err
}

// CacheGet retrieves a cached scrape result, returning nil if expired or missing.
func (s *BoltStore) CacheGet(key string) (*models.ScrapeResult, error) {
	var result *models.ScrapeResult
	err := s.db.View(func(tx *bolt.Tx) error {
		data := tx.Bucket(bucketCache).Get([]byte(key))
		if data == nil {
			return nil
		}
		var entry cacheEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			return err
		}
		if time.Now().After(entry.ExpiresAt) {
			return nil
		}
		result = entry.Result
		return nil
	})
	return result, err
}

// CacheSet stores a scrape result with the given TTL.
func (s *BoltStore) CacheSet(key string, result *models.ScrapeResult, ttl time.Duration) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		entry := cacheEntry{
			Result:    result,
			ExpiresAt: time.Now().Add(ttl),
		}
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		return tx.Bucket(bucketCache).Put([]byte(key), data)
	})
}
