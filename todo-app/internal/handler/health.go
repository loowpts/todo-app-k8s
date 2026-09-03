package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type HealthHandler struct {
	pool  *pgxpool.Pool
	cache *redis.Client
}

func NewHealthHandler(pool *pgxpool.Pool, cache *redis.Client) *HealthHandler {
	return &HealthHandler{pool: pool, cache: cache}
}

// Healthz is the liveness probe: the process is up and can serve HTTP.
// It intentionally does not touch Postgres/Redis so a transient dependency
// outage never triggers a pod restart loop.
func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// Readyz is the readiness probe: only report ready when the app can
// actually serve traffic, i.e. Postgres and Redis are reachable.
func (h *HealthHandler) Readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	result := map[string]string{}
	ready := true

	if err := h.pool.Ping(ctx); err != nil {
		ready = false
		result["postgres"] = err.Error()
	} else {
		result["postgres"] = "ok"
	}

	if err := h.cache.Ping(ctx).Err(); err != nil {
		ready = false
		result["redis"] = err.Error()
	} else {
		result["redis"] = "ok"
	}

	status := http.StatusOK
	if !ready {
		status = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}
