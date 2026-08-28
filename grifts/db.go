package grifts

import (
	"fmt"
	"os"
	"strings"
	"time"

	"backend_server/models"

	"github.com/gobuffalo/grift/grift"
	"golang.org/x/crypto/bcrypt"
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
			email := strings.TrimSpace(os.Getenv("SUPERADMIN_EMAIL"))
			if email == "" {
				return fmt.Errorf("SUPERADMIN_EMAIL environment variable is not set")
			}

			rawPassword := strings.TrimSpace(os.Getenv("SUPERADMIN_PASSWORD"))
			if rawPassword == "" {
				return fmt.Errorf("SUPERADMIN_PASSWORD environment variable is not set")
			}

			var hashedPassword string
			if strings.HasPrefix(rawPassword, "$2a$") || strings.HasPrefix(rawPassword, "$2b$") || strings.HasPrefix(rawPassword, "$2y$") {
				hashedPassword = rawPassword
			} else {
				bytes, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
				if err != nil {
					return fmt.Errorf("gagal melakukan hash password superadmin: %v", err)
				}
				hashedPassword = string(bytes)
			}

			admin := &models.Admin{
				Email:        email,
				Password:     hashedPassword,
				IsSuperAdmin: true,
				IsActive:     true,
				LastLogin:    time.Now(),
			}

			err = models.DB.Create(admin)
			if err != nil {
				return fmt.Errorf("gagal membuat seed admin: %v", err)
			}
			
			fmt.Println("Berhasil: Akun Super Admin telah dibuat!")
		} else {
			fmt.Println("Dilewati: Tabel admins sudah memiliki data, seed dibatalkan.")
		}

		return nil
	})

})