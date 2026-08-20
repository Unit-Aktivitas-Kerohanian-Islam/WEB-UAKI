package middlewares

import (
	"net/http"

	"backend_server/models"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/render"
	"github.com/gobuffalo/pop/v6"
)

func SuperAdminMiddleware(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		role, _ := c.Value("role").(string)
		if role != "admin" {
			return c.Render(http.StatusForbidden, render.JSON(map[string]interface{}{
				"success": false,
				"message": "Forbidden - only super admin allowed",
			}))
		}

		tx, ok := c.Value("tx").(*pop.Connection)
		if !ok {
			return c.Render(http.StatusInternalServerError, render.JSON(map[string]interface{}{
				"success": false,
				"message": "Database connection not found",
			}))
		}

		userID, ok := c.Value("user_id").(string)
		if !ok || userID == "" {
			return c.Render(http.StatusUnauthorized, render.JSON(map[string]interface{}{
				"success": false,
				"message": "Unauthorized - missing user ID",
			}))
		}

		admin := &models.Admin{}
		if err := tx.Find(admin, userID); err != nil {
			return c.Render(http.StatusUnauthorized, render.JSON(map[string]interface{}{
				"success": false,
				"message": "Admin not found",
			}))
		}

		if !admin.IsSuperAdmin {
			return c.Render(http.StatusForbidden, render.JSON(map[string]interface{}{
				"success": false,
				"message": "Forbidden - only super admin allowed",
			}))
		}

		return next(c)
	}
}