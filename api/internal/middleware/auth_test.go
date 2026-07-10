package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type fakeVerifier struct {
	id  int
	err error
}

func (f fakeVerifier) Verify(string) (int, error) { return f.id, f.err }

type fakeActive struct {
	active bool
	err    error
}

func (f fakeActive) IsActive(context.Context, int) (bool, error) { return f.active, f.err }

type fakeFuncs struct{}

func (fakeFuncs) FunctionsByUserID(context.Context, int) ([]string, error) { return nil, nil }

func TestAuthenticate_IsActive(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name     string
		verifier fakeVerifier
		active   fakeActive
		want     int
	}{
		{"有効なユーザーは通過", fakeVerifier{id: 1}, fakeActive{active: true}, http.StatusOK},
		{"無効化済みユーザーは401", fakeVerifier{id: 1}, fakeActive{active: false}, http.StatusUnauthorized},
		{"トークン不正は401", fakeVerifier{err: assertErr}, fakeActive{active: true}, http.StatusUnauthorized},
		{"有効性照会のDBエラーは500", fakeVerifier{id: 1}, fakeActive{err: assertErr}, http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mw := New(tc.verifier, fakeFuncs{}, tc.active)
			r := gin.New()
			r.GET("/x", mw.Authenticate(), func(c *gin.Context) { c.Status(http.StatusOK) })

			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			req.Header.Set("Authorization", "Bearer tok")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tc.want, w.Code)
		})
	}
}

var assertErr = &staticErr{"boom"}

type staticErr struct{ s string }

func (e *staticErr) Error() string { return e.s }
