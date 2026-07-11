package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter は key（既定はクライアントIP）ごとに rate.Limiter を保持し、
// 認証系エンドポイントへのオンライン総当り／クレデンシャルスタッフィングを抑止する。
//
// フェーズ1は単一インスタンス前提のためインメモリ実装とする（Redis 等の外部
// ストアは持たない=YAGNI）。エントリはリクエストごとに lastSeen を更新し、
// ttl を超えて未使用のものを破棄して map の無限増加を防ぐ。
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	limit    rate.Limit // 1 秒あたりの補充レート
	burst    int        // 瞬間的に許容するバースト数（トークンバケット容量）
	ttl      time.Duration
	keyFunc  func(*gin.Context) string
	now      func() time.Time
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// staleTTL は visitor を破棄するまでの未使用許容時間。
const staleTTL = 10 * time.Minute

// NewRateLimiter は「1 分あたり perMin リクエスト・バースト burst」で制限する
// RateLimiter を生成する。perMin / burst は config で 1 以上に検証済みの前提。
func NewRateLimiter(perMin, burst int) *RateLimiter {
	return &RateLimiter{
		visitors: make(map[string]*visitor),
		limit:    rate.Limit(float64(perMin) / 60.0),
		burst:    burst,
		ttl:      staleTTL,
		keyFunc:  func(c *gin.Context) string { return c.ClientIP() },
		now:      time.Now,
	}
}

// Limit は key ごとのレート制限を適用する gin ミドルウェアを返す。
// 上限超過時は 429 を返してハンドラへ到達させない。
func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.allow(rl.keyFunc(c)) {
			c.Header("Retry-After", "60")
			abort(c, http.StatusTooManyRequests, "リクエストが多すぎます。しばらくしてから再試行してください")
			return
		}
		c.Next()
	}
}

// allow は key に対応するリミッターからトークンを1つ消費できるかを返す。
func (rl *RateLimiter) allow(key string) bool {
	now := rl.now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.cleanupLocked(now)

	v, ok := rl.visitors[key]
	if !ok {
		v = &visitor{limiter: rate.NewLimiter(rl.limit, rl.burst)}
		rl.visitors[key] = v
	}
	v.lastSeen = now
	return v.limiter.AllowN(now, 1)
}

// cleanupLocked は ttl を超えて未使用の visitor を破棄する。
// 認証系エンドポイントは低頻度アクセスのため、毎回の線形走査でも十分軽量。
func (rl *RateLimiter) cleanupLocked(now time.Time) {
	for k, v := range rl.visitors {
		if now.Sub(v.lastSeen) > rl.ttl {
			delete(rl.visitors, k)
		}
	}
}
