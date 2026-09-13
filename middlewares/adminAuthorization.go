package middlewares

import (
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/render"
)

func AdminMiddleware(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		role, ok := c.Value("role").(string)
		
		if !ok || role != "admin" {
			return c.Render(http.StatusForbidden, render.JSON(map[string]interface{}{
				"success": false,
				"message": "Forbidden - Strictly for Admin",
			}))
		}
		
		return next(c)
	}
}