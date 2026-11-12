package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/web3connected/codexsecure-api/internal/database"
)

// ServiceTracker tracks endpoint hits for metrics
type ServiceTracker struct {
	db *database.DB
}

// NewServiceTracker creates a new service tracker
func NewServiceTracker(db *database.DB) *ServiceTracker {
	return &ServiceTracker{db: db}
}

// TrackRequest middleware logs endpoint hits to database
func (st *ServiceTracker) TrackRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Record hit after response
		duration := time.Since(start)
		endpoint := c.Request.Method + " " + c.FullPath()
		statusCode := c.Writer.Status()
		userAgent := c.Request.UserAgent()
		ipAddress := c.ClientIP()

		// Log to database asynchronously
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := st.recordHit(ctx, endpoint, statusCode, duration, userAgent, ipAddress); err != nil {
				log.Warn().Err(err).
					Str("endpoint", endpoint).
					Msg("Failed to record endpoint hit")
			}
		}()
	}
}

// recordHit stores the hit in request_logs using tenant_id to identify this service.
func (st *ServiceTracker) recordHit(ctx context.Context, endpoint string, statusCode int, duration time.Duration, userAgent, ipAddress string) error {
	// endpoint is "METHOD /path" — split for the method/url columns
	method := endpoint
	url := ""
	if i := len(endpoint); i > 0 {
		for j, ch := range endpoint {
			if ch == ' ' {
				method = endpoint[:j]
				url = endpoint[j+1:]
				break
			}
		}
	}

	query := `
		INSERT INTO request_logs
		(tenant_id, method, url, status_code, response_time_ms, user_agent, ip_address, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
	`

	_, err := st.db.Pool.Exec(ctx, query,
		"codexsecure",
		method,
		url,
		statusCode,
		duration.Milliseconds(),
		userAgent,
		ipAddress,
	)

	return err
}

// GetMetrics retrieves service metrics
func (st *ServiceTracker) GetMetrics(ctx context.Context, hours int) (map[string]interface{}, error) {
	query := `
		SELECT
			method || ' ' || url           AS endpoint,
			COUNT(*)                        AS total_hits,
			AVG(response_time_ms)::BIGINT   AS avg_response_time,
			MAX(updated_at)                 AS last_hit
		FROM request_logs
		WHERE tenant_id = $1
			AND created_at >= NOW() - INTERVAL '1 hour' * $2
		GROUP BY method, url
		ORDER BY total_hits DESC
	`

	rows, err := st.db.Pool.Query(ctx, query, "codexsecure", hours)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	endpoints := []map[string]interface{}{}
	var totalHits int64

	for rows.Next() {
		var endpoint string
		var hits int64
		var avgTime float64
		var lastHit time.Time

		if err := rows.Scan(&endpoint, &hits, &avgTime, &lastHit); err != nil {
			return nil, err
		}

		totalHits += hits
		endpoints = append(endpoints, map[string]interface{}{
			"endpoint":          endpoint,
			"hits":              hits,
			"avg_response_ms":   avgTime,
			"last_hit":          lastHit,
		})
	}

	return map[string]interface{}{
		"service":       "codexsecure",
		"total_hits":    totalHits,
		"endpoints":     endpoints,
		"period_hours":  hours,
	}, nil
}
