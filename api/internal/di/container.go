package di

import (
	"pmo-agent/api/internal/config"
	"pmo-agent/api/internal/handler"
	"pmo-agent/api/internal/infra"
	"pmo-agent/api/internal/middleware"
	"pmo-agent/api/internal/repository"
	"pmo-agent/api/internal/usecase"

	"go.uber.org/dig"
)

// BuildContainer は全依存グラフを宣言した dig コンテナを返す。
// 各層はインターフェースに依存し、具体実装は dig.As でバインドする。
func BuildContainer() (*dig.Container, error) {
	return buildContainer()
}

// buildContainer は dig オプションを差し込める内部ビルダ。テストでは dig.DryRun を渡して
// DB 接続なしにグラフの解決可能性のみを検証する。
func buildContainer(opts ...dig.Option) (*dig.Container, error) {
	c := dig.New(opts...)

	provide := func(ctor any, opts ...dig.ProvideOption) {
		if err := c.Provide(ctor, opts...); err != nil {
			panic(err)
		}
	}

	// 設定・インフラ
	provide(config.Load)
	provide(infra.NewDB)
	provide(infra.NewPasswordHasher, dig.As(new(usecase.Hasher)))
	provide(infra.NewTokenManager, dig.As(new(usecase.TokenManager)))
	provide(provideJWT, dig.As(new(usecase.AccessTokenIssuer), new(middleware.TokenVerifier)))

	// リポジトリ（インターフェースとしてバインド）
	provide(repository.NewUserRepo, dig.As(new(usecase.UserRepository), new(middleware.FunctionResolver), new(middleware.ActiveResolver)))
	provide(repository.NewRoleRepo, dig.As(new(usecase.RoleRepository)))
	provide(repository.NewCategoryRepo, dig.As(new(usecase.CategoryRepository)))
	provide(repository.NewPasswordSetTokenRepo, dig.As(new(usecase.PasswordSetTokenRepository)))
	provide(repository.NewRefreshTokenRepo, dig.As(new(usecase.RefreshTokenRepository)))
	provide(repository.NewProgramRepo, dig.As(new(usecase.ProgramRepository)))
	provide(repository.NewProjectRepo, dig.As(new(usecase.ProjectRepository)))
	provide(repository.NewMemberRepo, dig.As(new(usecase.ProjectMemberRepository)))
	provide(repository.NewAttributeRepo, dig.As(new(usecase.AttributeRepository)))

	// ユースケース
	provide(provideAuthUsecase)
	provide(provideUserUsecase)
	provide(usecase.NewCategoryUsecase)
	provide(usecase.NewProgramUsecase)
	provide(usecase.NewProjectUsecase)
	provide(usecase.NewMemberUsecase)
	provide(usecase.NewScopeUsecase, dig.As(new(middleware.ScopeResolver)))
	provide(usecase.NewAttributeUsecase)

	// ハンドラ・ミドルウェア・ルーター
	provide(provideAuthHandler)
	provide(handler.NewUserHandler)
	provide(handler.NewCategoryHandler)
	provide(handler.NewMetaHandler)
	provide(handler.NewProgramHandler)
	provide(handler.NewProjectHandler)
	provide(handler.NewMemberHandler)
	provide(handler.NewAttributeHandler)
	provide(middleware.New)
	provide(provideRateLimiter)
	provide(provideDeps)
	provide(handler.NewEngine)

	return c, nil
}

func provideJWT(cfg config.Config) *infra.JWTService {
	return infra.NewJWTService(cfg.JWTSecret, cfg.AccessTokenTTL)
}

func provideRateLimiter(cfg config.Config) *middleware.RateLimiter {
	return middleware.NewRateLimiter(cfg.AuthRateLimitPerMin, cfg.AuthRateLimitBurst)
}

func provideAuthUsecase(
	cfg config.Config,
	users usecase.UserRepository,
	set usecase.PasswordSetTokenRepository,
	ref usecase.RefreshTokenRepository,
	h usecase.Hasher,
	jwt usecase.AccessTokenIssuer,
	tm usecase.TokenManager,
) *usecase.AuthUsecase {
	return usecase.NewAuthUsecase(users, set, ref, h, jwt, tm, cfg.RefreshTokenTTL)
}

func provideUserUsecase(
	cfg config.Config,
	users usecase.UserRepository,
	roles usecase.RoleRepository,
	set usecase.PasswordSetTokenRepository,
	ref usecase.RefreshTokenRepository,
	tm usecase.TokenManager,
) *usecase.UserUsecase {
	return usecase.NewUserUsecase(users, roles, set, ref, tm, cfg.SetTokenTTL, cfg.AppBaseURL)
}

func provideAuthHandler(uc *usecase.AuthUsecase, cfg config.Config) *handler.AuthHandler {
	return handler.NewAuthHandler(uc, handler.CookieConfig{
		AccessMaxAge:  int(cfg.AccessTokenTTL.Seconds()),
		RefreshMaxAge: int(cfg.RefreshTokenTTL.Seconds()),
		Secure:        cfg.CookieSecure,
	})
}

func provideDeps(
	cfg config.Config,
	auth *handler.AuthHandler,
	user *handler.UserHandler,
	cat *handler.CategoryHandler,
	meta *handler.MetaHandler,
	program *handler.ProgramHandler,
	project *handler.ProjectHandler,
	member *handler.MemberHandler,
	attribute *handler.AttributeHandler,
	mw *middleware.Middleware,
	rateLimit *middleware.RateLimiter,
) handler.Deps {
	return handler.Deps{
		Auth: auth, User: user, Category: cat, Meta: meta,
		Program: program, Project: project, Member: member, Attribute: attribute, MW: mw,
		RateLimit:     rateLimit,
		AllowedOrigin: cfg.AppBaseURL,
	}
}
