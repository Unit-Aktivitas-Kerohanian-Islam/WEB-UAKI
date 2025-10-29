package actions

import (
	"sync"

	"backend_server/jwt"
	"backend_server/locales"
	"backend_server/middlewares"
	"backend_server/models"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo-pop/v3/pop/popmw"
	"github.com/gobuffalo/envy"
	"github.com/gobuffalo/middleware/contenttype"
	"github.com/gobuffalo/middleware/forcessl"
	"github.com/gobuffalo/middleware/i18n"
	"github.com/gobuffalo/middleware/paramlogger"
	"github.com/gobuffalo/x/sessions"
	"github.com/rs/cors"
	"github.com/unrolled/secure"
)

// ENV is used to help switch settings based on where the
// application is being run. Default is "development".
var ENV = envy.Get("GO_ENV", "development")
var JWTService jwt.Interface

var (
	app     *buffalo.App
	appOnce sync.Once
	T       *i18n.Translator
)

// App is where all routes and middleware for buffalo
// should be defined. This is the nerve center of your
// application.
//
// Routing, middleware, groups, etc... are declared TOP -> DOWN.
// This means if you add a middleware to `app` *after* declaring a
// group, that group will NOT have that new middleware. The same
// is true of resource declarations as well.
//
// It also means that routes are checked in the order they are declared.
// `ServeFiles` is a CATCH-ALL route, so it should always be
// placed last in the route declarations, as it will prevent routes
// declared after it to never be called.
func App() *buffalo.App {
	appOnce.Do(func() {
		app = buffalo.New(buffalo.Options{
			Env:          ENV,
			SessionStore: sessions.Null{},
			PreWares: []buffalo.PreWare{
				cors.Default().Handler,
			},
			SessionName: "_backend_server_session",
		})

		JWTService = jwt.Init()

		// Automatically redirect to SSL
		app.Use(forceSSL())

		// Log request parameters (filters apply).
		app.Use(paramlogger.ParameterLogger)

		// Set the request content type to JSON
		app.Use(contenttype.Set("application/json"))

		// Wraps each request in a transaction.
		//   c.Value("tx").(*pop.Connection)
		// Remove to disable this.
		app.Use(popmw.Transaction(models.DB))

		//middleware for translations
		auth := middlewares.AuthMiddleware(JWTService)
		superAdminAuth := middlewares.SuperAdminMiddleware

		//routes
		app.GET("/", HomeHandler)

		// app.POST("/admin/login", Login)

		admins := AdminsResource{}
		app.POST("/admins/login", admins.Login)
		adminRoute := app.Resource("/admins", admins)
		adminRoute.Middleware.Use(auth, superAdminAuth)
		adminRoute.Middleware.Skip(auth, admins.Login)
		adminRoute.Middleware.Skip(superAdminAuth, admins.Login)
	

		adminRoute.Middleware.Skip(superAdminAuth, admins.Create)
		adminRoute.Middleware.Skip(auth, admins.Create)

		articles := NewArticleResource()
		app.POST("/articles/image", auth(articles.UploadImage))
		articleRoute := app.Resource("/articles", articles)
		articleRoute.Middleware.Use(auth)
		articleRoute.Middleware.Skip(auth, articles.List, articles.Show)

		media := NewMediaResource()
		app.POST("/media/image", auth(media.UploadImage))
		mediaRoute := app.Resource("/media", media)
		mediaRoute.Middleware.Use(auth)
		mediaRoute.Middleware.Skip(auth, media.List, media.Show)

		mediaCategories := MediaCategoriesResource{}
		mediaCategoriesRoute := app.Resource("/media-categories", mediaCategories)
		mediaCategoriesRoute.Middleware.Use(auth)
		mediaCategoriesRoute.Middleware.Use(superAdminAuth)
		mediaCategoriesRoute.Middleware.Skip(superAdminAuth, mediaCategories.List)
	})

	return app
}

// translations will load locale files, set up the translator `actions.T`,
// and will return a middleware to use to load the correct locale for each
// request.
// for more information: https://gobuffalo.io/en/docs/localization
func translations() buffalo.MiddlewareFunc {
	var err error
	if T, err = i18n.New(locales.FS(), "en-US"); err != nil {
		app.Stop(err)
	}
	return T.Middleware()
}

// forceSSL will return a middleware that will redirect an incoming request
// if it is not HTTPS. "http://example.com" => "https://example.com".
// This middleware does **not** enable SSL. for your application. To do that
// we recommend using a proxy: https://gobuffalo.io/en/docs/proxy
// for more information: https://github.com/unrolled/secure/
func forceSSL() buffalo.MiddlewareFunc {
	return forcessl.Middleware(secure.Options{
		SSLRedirect:     ENV == "production",
		SSLProxyHeaders: map[string]string{"X-Forwarded-Proto": "https"},
	})
}
