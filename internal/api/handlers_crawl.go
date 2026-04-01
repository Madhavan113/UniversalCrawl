package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/madhavanp/universalcrawl/internal/crawler"
	"github.com/madhavanp/universalcrawl/internal/jobs"
	"github.com/madhavanp/universalcrawl/internal/models"
	"github.com/madhavanp/universalcrawl/internal/storage"
)

type crawlHandler struct {
	crawler *crawler.WebCrawler
	store   storage.Store
	queue   *jobs.Queue
}

// HandleCrawlStart processes POST /v1/crawl — starts an async crawl job.
func (h *crawlHandler) HandleCrawlStart(w http.ResponseWriter, r *http.Request) {
	var req models.CrawlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}
	if req.Limit == 0 {
		req.Limit = 100
	}
	if len(req.Formats) == 0 {
		req.Formats = []string{"markdown"}
	}

	jobID := generateID("crawl_")
	job := &models.CrawlJob{
		ID:        jobID,
		Status:    models.CrawlStatusScraping,
		URL:       req.URL,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := h.store.CreateJob(job); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create job: "+err.Error())
		return
	}

	h.queue.Submit(&jobs.Job{
		ID: jobID,
		Execute: func(ctx context.Context) error {
			err := h.crawler.Crawl(ctx, &req, func(result *models.ScrapeResult) {
				h.store.AddJobResult(jobID, result)
				j, _ := h.store.GetJob(jobID)
				if j != nil {
					j.Completed++
					j.Total, _ = countTotal(h.store, jobID)
					h.store.UpdateJob(j)
				}
			})

			j, _ := h.store.GetJob(jobID)
			if j != nil {
				if err != nil {
					j.Status = models.CrawlStatusFailed
				} else {
					j.Status = models.CrawlStatusCompleted
				}
				h.store.UpdateJob(j)
			}
			return err
		},
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"id":      jobID,
		"url":     "/v1/crawl/" + jobID,
	})
}

// HandleCrawlStatus processes GET /v1/crawl/{id} — returns crawl progress.
func (h *crawlHandler) HandleCrawlStatus(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "id")

	job, err := h.store.GetJob(jobID)
	if err != nil {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}

	cursor := 0
	if c := r.URL.Query().Get("cursor"); c != "" {
		cursor, _ = strconv.Atoi(c)
	}
	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		limit, _ = strconv.Atoi(l)
	}

	results, nextCursor, err := h.store.GetJobResults(jobID, cursor, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get results: "+err.Error())
		return
	}

	resp := map[string]interface{}{
		"success":   true,
		"status":    job.Status,
		"total":     job.Total,
		"completed": job.Completed,
		"expiresAt": job.ExpiresAt,
		"data":      results,
	}
	if nextCursor > 0 {
		resp["next"] = strconv.Itoa(nextCursor)
	}

	writeJSON(w, http.StatusOK, resp)
}

func countTotal(store storage.Store, jobID string) (int, error) {
	results, _, err := store.GetJobResults(jobID, 0, 100000)
	if err != nil {
		return 0, err
	}
	return len(results), nil
}

func generateID(prefix string) string {
	b := make([]byte, 8)
	rand.Read(b)
	return prefix + hex.EncodeToString(b)
}
