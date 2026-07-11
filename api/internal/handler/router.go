package handler

import (
	"net/http"

	"pmo-agent/api/internal/middleware"

	"github.com/gin-gonic/gin"
)

// Deps はルーター構築に必要なハンドラ・ミドルウェア一式。
type Deps struct {
	Auth          *AuthHandler
	User          *UserHandler
	Category      *CategoryHandler
	Meta          *MetaHandler
	Program       *ProgramHandler
	Project       *ProjectHandler
	Member        *MemberHandler
	Attribute     *AttributeHandler
	MW            *middleware.Middleware
	RateLimit     *middleware.RateLimiter
	AllowedOrigin string
}

// NewEngine はルーティングを構築した Gin エンジンを返す。全エンドポイントは /api 配下。
func NewEngine(d Deps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(corsMiddleware(d.AllowedOrigin))

	api := r.Group("/api")

	api.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	// --- 認証（公開・レート制限あり） ---
	// 資格情報を扱う公開エンドポイントはオンライン総当り／クレデンシャル
	// スタッフィングの標的になるため、IP 単位のレート制限を適用する。
	pub := api.Group("")
	pub.Use(d.RateLimit.Limit())
	{
		pub.POST("/auth/login", d.Auth.Login)
		pub.POST("/auth/refresh", d.Auth.Refresh)
		pub.GET("/auth/set-password/:token", d.Auth.VerifySetToken)
		pub.POST("/auth/set-password", d.Auth.SetPassword)
	}

	// --- 認証（要ログイン） ---
	auth := api.Group("")
	auth.Use(d.MW.Authenticate())
	{
		auth.POST("/auth/logout", d.Auth.Logout)
		auth.GET("/auth/me", d.Auth.Me)
		auth.POST("/auth/change-password", d.Auth.ChangePassword)

		// メンバー管理（manage_users）
		auth.GET("/users", d.MW.RequireFunction("manage_users"), d.User.List)
		auth.POST("/users", d.MW.RequireFunction("manage_users"), d.User.Create)
		auth.GET("/users/:id", d.MW.RequireFunction("manage_users"), d.User.Get)
		auth.PUT("/users/:id", d.MW.RequireFunction("manage_users"), d.User.Update)
		auth.DELETE("/users/:id", d.MW.RequireFunction("manage_users"), d.User.Delete)
		auth.POST("/users/:id/reissue-link", d.MW.RequireFunction("manage_users"), d.User.ReissueLink)
		auth.GET("/roles", d.MW.RequireFunction("manage_users"), d.Meta.Roles)

		// プログラム管理
		auth.GET("/programs", d.MW.RequireFunction("view_project_detail"), d.Program.List)
		auth.POST("/programs", d.MW.RequireFunction("issue_project_code"), d.Program.Create)
		auth.GET("/programs/:id", d.MW.RequireFunction("view_project_detail"), d.Program.Get)
		auth.PUT("/programs/:id", d.MW.RequireFunction("issue_project_code"), d.Program.Update)
		auth.DELETE("/programs/:id", d.MW.RequireFunction("issue_project_code"), d.Program.Delete)
		auth.GET("/programs/:id/projects", d.MW.RequireFunction("view_project_detail"), d.MW.ResolveProjectScope(), d.Program.ListProjects)
		auth.POST("/programs/:id/projects", d.MW.RequireFunction("manage_projects"), d.Project.CreateUnderProgram)

		// プロジェクト管理（参照系は担当PJスコープを適用）
		auth.GET("/projects", d.MW.RequireFunction("view_project_detail"), d.MW.ResolveProjectScope(), d.Project.List)
		auth.GET("/projects/:id", d.MW.RequireFunction("view_project_detail"), d.MW.ResolveProjectScope(), d.Project.Get)
		auth.PUT("/projects/:id", d.MW.RequireFunction("manage_projects"), d.Project.Update)
		auth.DELETE("/projects/:id", d.MW.RequireFunction("manage_projects"), d.Project.Delete)
		auth.POST("/projects/:id/issue-code", d.MW.RequireFunction("issue_project_code"), d.Project.IssueCode)

		// プロジェクトメンバー（担当PJスコープを適用。参照は view_project_detail、変更は assign_project_members）
		auth.GET("/projects/:id/members", d.MW.RequireFunction("view_project_detail"), d.MW.ResolveProjectScope(), d.Member.List)
		auth.POST("/projects/:id/members", d.MW.RequireFunction("assign_project_members"), d.MW.ResolveProjectScope(), d.Member.Assign)
		auth.PUT("/projects/:id/members/:userId", d.MW.RequireFunction("assign_project_members"), d.MW.ResolveProjectScope(), d.Member.Update)
		auth.DELETE("/projects/:id/members/:userId", d.MW.RequireFunction("assign_project_members"), d.MW.ResolveProjectScope(), d.Member.Delete)

		// プロジェクト属性（参照は view_project_detail、変更は manage_projects）
		auth.GET("/projects/:id/attributes", d.MW.RequireFunction("view_project_detail"), d.Attribute.List)
		auth.POST("/projects/:id/attributes", d.MW.RequireFunction("manage_projects"), d.Attribute.Assign)
		auth.DELETE("/projects/:id/attributes/:valueId", d.MW.RequireFunction("manage_projects"), d.Attribute.Delete)

		// 属性カテゴリ（参照は認証のみ、変更は manage_categories）
		auth.GET("/categories", d.Category.List)
		auth.POST("/categories", d.MW.RequireFunction("manage_categories"), d.Category.Create)
		auth.PUT("/categories/:id", d.MW.RequireFunction("manage_categories"), d.Category.Update)
		auth.DELETE("/categories/:id", d.MW.RequireFunction("manage_categories"), d.Category.Delete)
		auth.POST("/categories/:id/reactivate", d.MW.RequireFunction("manage_categories"), d.Category.Reactivate)
		auth.GET("/categories/:id/values", d.Category.ListValues)
		auth.POST("/categories/:id/values", d.MW.RequireFunction("manage_categories"), d.Category.CreateValue)
		auth.PUT("/categories/:id/values/:valueId", d.MW.RequireFunction("manage_categories"), d.Category.UpdateValue)
		auth.DELETE("/categories/:id/values/:valueId", d.MW.RequireFunction("manage_categories"), d.Category.DeleteValue)
		auth.POST("/categories/:id/values/:valueId/reactivate", d.MW.RequireFunction("manage_categories"), d.Category.ReactivateValue)
	}

	return r
}

// corsMiddleware は Cookie 認証を成立させるため credentials 付き CORS を許可する。
func corsMiddleware(origin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Vary", "Origin")
		}
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
