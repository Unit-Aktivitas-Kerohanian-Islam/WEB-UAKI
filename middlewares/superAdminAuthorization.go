package middlewares

import (
	"net/http"

	"backend_server/models"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/render"
	"github.com/gobuffalo/pop/v6"
)

// SuperAdminMiddleware memastikan hanya Super Admin yang boleh mengakses route tertentu
func SuperAdminMiddleware(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		// Ambil koneksi DB dari context
		tx, ok := c.Value("tx").(*pop.Connection)
		if !ok {
			return c.Render(http.StatusInternalServerError, render.JSON(map[string]interface{}{
				"success": false,
				"message": "Database connection not found",
			}))
		}

		// Ambil ID admin dari context (hasil validasi JWT)
		adminID, ok := c.Value("admin_id").(string)
		if !ok || adminID == "" {
			return c.Render(http.StatusUnauthorized, render.JSON(map[string]interface{}{
				"success": false,
				"message": "Unauthorized - missing admin ID in context",
			}))
		}

		// Ambil data admin dari database
		admin := &models.Admin{}
		if err := tx.Find(admin, adminID); err != nil {
			return c.Render(http.StatusUnauthorized, render.JSON(map[string]interface{}{
				"success": false,
				"message": "Admin not found",
			}))
		}

		// Cek apakah admin adalah super admin
		if !admin.IsSuperAdmin {
			return c.Render(http.StatusForbidden, render.JSON(map[string]interface{}{
				"success": false,
				"message": "Forbidden - only super admin allowed",
			}))
		}

		// Jika semua aman, lanjutkan ke handler berikutnya
		return next(c)
	}
}
