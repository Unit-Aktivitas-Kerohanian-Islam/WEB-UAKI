package actions

import (
	"net/http"
	"os"
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

		// ==========================================
		// CARA BULLETPROOF SERVE SWAGGER (Anti-Loop & Anti-Cache)
		// ==========================================
		// 1. Endpoint HTML (URL Baru: /docs)
		app.GET("/docs", func(c buffalo.Context) error {
			c.Response().Header().Set("Content-Type", "text/html")
			html := `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Swagger UI - UKM UAKI</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui.css" />
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-bundle.js"></script>
    <script>
      window.onload = () => {
        window.ui = SwaggerUIBundle({
          url: '/docs.json',
          dom_id: '#swagger-ui',
        });
      };
    </script>
  </body>
</html>`
			_, err := c.Response().Write([]byte(html))
			return err
		})

		// 2. Endpoint JSON (Membaca langsung dari folder public)
		app.GET("/docs.json", func(c buffalo.Context) error {
			c.Response().Header().Set("Content-Type", "application/json")
			fileBytes, err := os.ReadFile("./public/swagger/swagger.json")
			if err != nil {
				return c.Error(http.StatusInternalServerError, err)
			}
			_, err = c.Response().Write(fileBytes)
			return err
		})
		// ==========================================

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
		
		registrants := NewRegistrantsResource()
		app.POST("/registrants/login", registrants.Login)
		app.POST("/registrants/cv", registrants.UploadCV) // Endpoint Upload CV
		
		app.GET("/registrants/me", auth(registrants.GetMe))
		app.PUT("/registrants/me", auth(registrants.UpdateMe))
		app.PATCH("/registrants/{registrant_id}/status", auth(superAdminAuth(registrants.UpdateStatus)))

		registrantsRoute := app.Resource("/registrants", registrants)
		registrantsRoute.Middleware.Use(auth)
		
		// Bebaskan akses publik
		registrantsRoute.Middleware.Skip(auth, registrants.Create, registrants.UploadCV)

		// Serve folder uploads untuk gambar dan PDF
		app.ServeFiles("/uploads", http.Dir("./public/uploads"))
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