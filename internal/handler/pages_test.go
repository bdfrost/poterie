package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdminAuth(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	protected := adminAuth("admin", "secret")(handler)

	t.Run("no auth", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/admin", nil)
		w := httptest.NewRecorder()
		protected.ServeHTTP(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("wrong password", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/admin", nil)
		r.SetBasicAuth("admin", "wrong")
		w := httptest.NewRecorder()
		protected.ServeHTTP(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("correct auth", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/admin", nil)
		r.SetBasicAuth("admin", "secret")
		w := httptest.NewRecorder()
		protected.ServeHTTP(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestParseInt(t *testing.T) {
	assert.Equal(t, 5, parseInt("5", 0))
	assert.Equal(t, 0, parseInt("", 0))
	assert.Equal(t, 99, parseInt("", 99))
	assert.Equal(t, 0, parseInt("abc", 0))
	assert.Equal(t, -1, parseInt("-1", 0))
}
