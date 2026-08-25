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
				cors.New(cors.Options{
					AllowedOrigins:   []string{"*"}, 
					AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
					AllowedHeaders:   []string{"Authorization", "Content-Type", "Accept"}, 
					AllowCredentials: true,
				}).Handler,
			},
			SessionName: "_backend_server_session",
		})

		JWTService = jwt.Init()

		app.Use(forceSSL())
		app.Use(paramlogger.ParameterLogger)
		app.Use(contenttype.Set("application/json"))
		app.Use(popmw.Transaction(models.DB))

		auth := middlewares.AuthMiddleware(JWTService)
		adminAuth := middlewares.AdminMiddleware
		superAdminAuth := middlewares.SuperAdminMiddleware

		app.GET("/", HomeHandler)

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

		app.GET("/docs.json", func(c buffalo.Context) error {
			c.Response().Header().Set("Content-Type", "application/json")
			fileBytes, err := os.ReadFile("./public/swagger/swagger.json")
			if err != nil {
				return c.Error(http.StatusInternalServerError, err)
			}
			_, err = c.Response().Write(fileBytes)
			return err
		})

		// ================== ADMIN ROUTES ==================
		admins := AdminsResource{}
		app.POST("/admins/login", admins.Login)
		adminRoute := app.Resource("/admins", admins)
		adminRoute.Middleware.Use(auth, adminAuth, superAdminAuth)
		adminRoute.Middleware.Skip(auth, admins.Login)
		adminRoute.Middleware.Skip(adminAuth, admins.Login)
		adminRoute.Middleware.Skip(superAdminAuth, admins.Login)

		// ================== ARTICLE ROUTES ==================
		articles := NewArticleResource()
		app.POST("/articles/image", auth(adminAuth(articles.UploadImage)))
		articleRoute := app.Resource("/articles", articles)
		articleRoute.Middleware.Use(auth, adminAuth)
		articleRoute.Middleware.Skip(auth, articles.List, articles.Show)
		articleRoute.Middleware.Skip(adminAuth, articles.List, articles.Show)

		// ================== MEDIA ROUTES ==================
		media := NewMediaResource()
		app.POST("/media/image", auth(adminAuth(media.UploadImage)))
		mediaRoute := app.Resource("/media", media)
		mediaRoute.Middleware.Use(auth, adminAuth)
		mediaRoute.Middleware.Skip(auth, media.List, media.Show)
		mediaRoute.Middleware.Skip(adminAuth, media.List, media.Show)

		mediaCategories := MediaCategoriesResource{}
		mediaCategoriesRoute := app.Resource("/media-categories", mediaCategories)
		mediaCategoriesRoute.Middleware.Use(auth, adminAuth, superAdminAuth)
		mediaCategoriesRoute.Middleware.Skip(auth, mediaCategories.List)
		mediaCategoriesRoute.Middleware.Skip(adminAuth, mediaCategories.List)
		mediaCategoriesRoute.Middleware.Skip(superAdminAuth, mediaCategories.List)
		
		// ================== REGISTRANT ROUTES ==================
		registrants := NewRegistrantsResource()
		app.POST("/registrants/login", registrants.Login)
		app.POST("/registrants/auth/google", registrants.GoogleLogin)
		
		// Pendaftar Only (Validasi di dalam handler)
		app.POST("/registrants/cv", auth(registrants.UploadCV)) 
		app.GET("/registrants/me", auth(registrants.GetMe))
		app.PUT("/registrants/me", auth(registrants.UpdateMe))
		
		// Admin Only (Validasi via Middleware)
		app.PATCH("/registrants/{registrant_id}/status", auth(adminAuth(registrants.UpdateStatus)))
		app.POST("/registrants/send-schedule", auth(adminAuth(registrants.SendSchedule)))

		// List, Show, Destroy Registrant dijaga oleh adminAuth
		registrantsRoute := app.Resource("/registrants", registrants)
		registrantsRoute.Middleware.Use(auth, adminAuth)
		
		app.ServeFiles("/uploads", http.Dir("./public/uploads"))
	})

	return app
}

// ... (sisa fungsi translations dan forceSSL biarkan tetap sama)
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