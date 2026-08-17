package actions

import (
	"net/http"
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

var ENV = envy.Get("GO_ENV", "development")
var JWTService jwt.Interface

var (
	app     *buffalo.App
	appOnce sync.Once
	T       *i18n.Translator
)

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

		app.Use(forceSSL())
		app.Use(paramlogger.ParameterLogger)
		app.Use(contenttype.Set("application/json"))
		app.Use(popmw.Transaction(models.DB))

		auth := middlewares.AuthMiddleware(JWTService)
		superAdminAuth := middlewares.SuperAdminMiddleware

		app.GET("/", HomeHandler)

		// Serve Local Uploads
		app.ServeFiles("/uploads", http.Dir("./public/uploads"))
		
		// Serve Swagger UI
		app.ServeFiles("/swagger", http.Dir("./public/swagger"))

		admins := AdminsResource{}
		app.POST("/admins/login", admins.Login)
		adminRoute := app.Resource("/admins", admins)
		adminRoute.Middleware.Use(auth, superAdminAuth)
		adminRoute.Middleware.Skip(auth, admins.Login)
		adminRoute.Middleware.Skip(superAdminAuth, admins.Login)

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
		mediaCategoriesRoute.Middleware.Skip(auth, mediaCategories.List)
		mediaCategoriesRoute.Middleware.Skip(superAdminAuth, mediaCategories.List)
		
		// Registrants Routing
		registrants := NewRegistrantsResource()
		app.POST("/registrants/login", registrants.Login)
		app.POST("/registrants/cv", registrants.UploadCV) // Endpoint Upload CV terpisah
		
		registrantsRoute := app.Resource("/registrants", registrants)
		registrantsRoute.Middleware.Use(auth)
		
		// Bebaskan akses publik untuk Create (Submit JSON), UploadCV, dan Login
		registrantsRoute.Middleware.Skip(auth, registrants.Create, registrants.UploadCV)
	})

	return app
}

func translations() buffalo.MiddlewareFunc {
	var err error
	if T, err = i18n.New(locales.FS(), "en-US"); err != nil {
		app.Stop(err)
	}
	return T.Middleware()
}

func forceSSL() buffalo.MiddlewareFunc {
	return forcessl.Middleware(secure.Options{
		SSLRedirect:     ENV == "production",
		SSLProxyHeaders: map[string]string{"X-Forwarded-Proto": "https"},
	})
}