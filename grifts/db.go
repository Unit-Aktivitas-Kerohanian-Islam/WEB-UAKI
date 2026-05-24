package grifts

import (
	"backend_server/models"
	"fmt"
	"log"

	"github.com/gobuffalo/grift/grift"
	"golang.org/x/crypto/bcrypt"
)

var _ = grift.Namespace("db", func() {

	grift.Desc("seed", "Seeds UAKI database with default superadmin and media categories")
	grift.Add("seed", func(c *grift.Context) error {
		log.Println("🌱 Starting database seeding...")

		// 1. Seed Super Admin
		email := "superadmin@uaki.org"
		password := "UakiSuperPassword123!"

		// Check if admin already exists
		admin := &models.Admin{}
		exists, err := models.DB.Where("email = ?", email).Exists(admin)
		if err != nil {
			return fmt.Errorf("failed to check if superadmin exists: %v", err)
		}

		if !exists {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
			if err != nil {
				return fmt.Errorf("failed to hash password: %v", err)
			}

			newAdmin := &models.Admin{
				Email:        email,
				Password:     string(hashedPassword),
				IsSuperAdmin: true,
				IsActive:     true,
			}

			err = models.DB.Create(newAdmin)
			if err != nil {
				return fmt.Errorf("failed to create superadmin: %v", err)
			}
			log.Printf("✅ Super Admin berhasil dibuat! Email: %s, Password: %s\n", email, password)
		} else {
			log.Println("ℹ️ Super Admin sudah ada di database.")
		}

		// 2. Seed Media Categories
		categories := []string{"Kajian", "Artikel", "Dokumentasi", "Pengumuman"}
		for _, catName := range categories {
			cat := &models.MediaCategory{}
			exists, err := models.DB.Where("name = ?", catName).Exists(cat)
			if err != nil {
				return fmt.Errorf("failed to check if media category %s exists: %v", catName, err)
			}

			if !exists {
				newCat := &models.MediaCategory{
					Name: catName,
				}
				err = models.DB.Create(newCat)
				if err != nil {
					return fmt.Errorf("failed to create media category %s: %v", catName, err)
				}
				log.Printf("✅ Kategori media '%s' berhasil dibuat!\n", catName)
			} else {
				log.Printf("ℹ️ Kategori media '%s' sudah ada.\n", catName)
			}
		}

		log.Println("🌱 Database seeding completed successfully!")
		return nil
	})

})
