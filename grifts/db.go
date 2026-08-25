package grifts

import (
	"fmt"
	"time"

	"backend_server/models"

	"github.com/gobuffalo/grift/grift"
)

var _ = grift.Namespace("db", func() {

	grift.Desc("seed", "Seeds a database with initial data")
	grift.Add("seed", func(c *grift.Context) error {
		// Cek apakah tabel admin sudah ada isinya agar tidak terjadi duplikasi
		count, err := models.DB.Count("admins")
		if err != nil {
			return err
		}

		if count == 0 {
			admin := &models.Admin{
				Email:        "admin@uaki.ub.ac.id",
				Password:     "$2a$10$xiQGxfPAgcbjxWcXoXhczegziU8VqD34PQXk/He5b.u9qhbrpQcIW",
				IsSuperAdmin: true,
				IsActive:     true,
				LastLogin:    time.Now(),
			}

			err = models.DB.Create(admin)
			if err != nil {
				return fmt.Errorf("gagal membuat seed admin: %v", err)
			}
			
			fmt.Println("✅ Berhasil: Akun Super Admin (admin@uaki.ub.ac.id) telah dibuat!")
		} else {
			fmt.Println("⚠️ Dilewati: Tabel admins sudah memiliki data, seed dibatalkan.")
		}

		return nil
	})

})