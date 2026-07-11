package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// newTestRateLimiter は now と keyFunc を制御可能にしたテスト用リミッターを返す。
// keyFunc はリクエストのクエリ ?k= を鍵に使い、IP に依存せずキー分離を検証できる。
func newTestRateLimiter(perMin, burst int, now func() time.Time) *RateLimiter {
	rl := NewRateLimiter(perMin, burst)
	rl.now = now
	rl.keyFunc = func(c *gin.Context) string { return c.Query("k") }
	return rl
}

func doReq(r http.Handler, key string) int {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/x?k="+key, nil)
	r.ServeHTTP(w, req)
	return w.Code
}

func TestRateLimiter_Limit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("バースト内は通過し超過で429を返す", func(t *testing.T) {
		now := time.Unix(0, 0)
		rl := newTestRateLimiter(60, 3, func() time.Time { return now }) // 1req/s・バースト3
		r := gin.New()
		r.POST("/x", rl.Limit(), func(c *gin.Context) { c.Status(http.StatusOK) })

		// バースト3までは即時に通過する。
		for i := range 3 {
			assert.Equal(t, http.StatusOK, doReq(r, "a"), "バースト内の%d回目は通過", i+1)
		}
		// 4回目はトークンを使い切って429。
		assert.Equal(t, http.StatusTooManyRequests, doReq(r, "a"), "バースト超過は429")
	})

	t.Run("時間経過でトークンが補充され回復する", func(t *testing.T) {
		now := time.Unix(0, 0)
		rl := newTestRateLimiter(60, 1, func() time.Time { return now }) // 1req/s・バースト1
		r := gin.New()
		r.POST("/x", rl.Limit(), func(c *gin.Context) { c.Status(http.StatusOK) })

		assert.Equal(t, http.StatusOK, doReq(r, "a"), "最初の1回は通過")
		assert.Equal(t, http.StatusTooManyRequests, doReq(r, "a"), "直後は429")

		now = now.Add(time.Second) // 1秒経過 → 1トークン補充。
		assert.Equal(t, http.StatusOK, doReq(r, "a"), "補充後は再び通過")
	})

	t.Run("キー（IP）ごとに独立して計数する", func(t *testing.T) {
		now := time.Unix(0, 0)
		rl := newTestRateLimiter(60, 1, func() time.Time { return now }) // バースト1
		r := gin.New()
		r.POST("/x", rl.Limit(), func(c *gin.Context) { c.Status(http.StatusOK) })

		assert.Equal(t, http.StatusOK, doReq(r, "a"), "キーaの1回目は通過")
		assert.Equal(t, http.StatusTooManyRequests, doReq(r, "a"), "キーaの2回目は429")
		// 別キーは影響を受けない。
		assert.Equal(t, http.StatusOK, doReq(r, "b"), "キーbは独立して通過")
	})

	t.Run("超過時はRetry-Afterヘッダを付与する", func(t *testing.T) {
		now := time.Unix(0, 0)
		rl := newTestRateLimiter(60, 1, func() time.Time { return now })
		r := gin.New()
		r.POST("/x", rl.Limit(), func(c *gin.Context) { c.Status(http.StatusOK) })

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/x?k=a", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Equal(t, "60", w.Header().Get("Retry-After"))
	})
}

func TestRateLimiter_Cleanup(t *testing.T) {
	now := time.Unix(0, 0)
	rl := newTestRateLimiter(60, 1, func() time.Time { return now })

	rl.allow("a")
	assert.Len(t, rl.visitors, 1, "アクセスで visitor が登録される")

	// ttl を超えて別キーへアクセスすると、古い visitor は破棄される。
	now = now.Add(staleTTL + time.Minute)
	rl.allow("b")
	_, hasOld := rl.visitors["a"]
	assert.False(t, hasOld, "ttl 超過の visitor は破棄される")
	assert.Contains(t, rl.visitors, "b", "新しい visitor は保持される")
}
