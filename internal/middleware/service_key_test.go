package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestServiceKeyHeaderContract(t *testing.T) {
	t.Setenv("SERVICE_API_KEY", "test-service-key")
	router := gin.New()
	router.GET("/protected", ServiceKeyMiddleware(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	for _, check := range []struct {
		name   string
		header string
		status int
	}{
		{"canonical header", "X-Service-Key", http.StatusNoContent},
		{"incorrect SDK alias", "X-Codex-Service-Key", http.StatusUnauthorized},
		{"missing header", "", http.StatusUnauthorized},
	} {
		t.Run(check.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if check.header != "" {
				request.Header.Set(check.header, "test-service-key")
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			if recorder.Code != check.status {
				t.Fatalf("status=%d, want %d", recorder.Code, check.status)
			}
		})
	}
}
