package middlewares

import (
	"backend_server/jwt"
	"net/http"
	"strings"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/render"
)

func AuthMiddleware(jwtService jwt.Interface) buffalo.MiddlewareFunc {
	return func(next buffalo.Handler) buffalo.Handler {
		return func(c buffalo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.Render(http.StatusUnauthorized, render.JSON(map[string]interface{}{
					"success": false,
					"message": "Missing Authorization header",
				}))
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				return c.Render(http.StatusBadRequest, render.JSON(map[string]interface{}{
					"success": false,
					"message": "Invalid Authorization format, use 'Bearer <token>'",
				}))
			}

			userID, role, err := jwtService.ValidateToken(tokenString)
			if err != nil {
				return c.Render(http.StatusUnauthorized, render.JSON(map[string]interface{}{
					"success": false,
					"message": "Invalid or expired token",
				}))
			}

			// Simpan user_id dan role ke context
			c.Set("user_id", userID.String())
			c.Set("role", role)

			return next(c)
		}
	}
}