package grifts

import (
	"fmt"
	"time"

	"github.com/gobuffalo/grift/grift"
	"github.com/gofrs/uuid"
	"golang.org/x/crypto/bcrypt"
	"github.com/gobuffalo/envy"

	"backend_server/models"
)

var _ = grift.Namespace("db", func() {

	grift.Desc("seed", "Seeds the database with initial data (superadmin account)")
	var _ = grift.Add("seed", func(c *grift.Context) error {
		// Hash password sebelum disimpan
		password := envy.Get("SUPERADMIN_PASSWORD", "")
		plainPassword := password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}

		// Cek apakah superadmin sudah ada
		email := envy.Get("SUPERADMIN_EMAIL", "")
		existingAdmin := &models.Admin{}
		err = models.DB.Where("email = ?", email).First(existingAdmin)
		if err == nil {
			// Superadmin sudah ada, skip
			fmt.Println("⚠️  Superadmin already exists, skipping seed...")
			return nil
		}

		// Generate UUID baru
		adminID, err := uuid.NewV4()
		if err != nil {
			return fmt.Errorf("failed to generate UUID: %w", err)
		}

		// Buat superadmin baru
		superAdmin := &models.Admin{
			ID:           adminID,
			Email:        email,
			Password:     string(hashedPassword),
			IsSuperAdmin: true,
			IsActive:     true,
			LastLogin:    time.Now(),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		// Simpan ke database
		if err := models.DB.Create(superAdmin); err != nil {
			return fmt.Errorf("failed to create superadmin: %w", err)
		}

		fmt.Printf("✅ Superadmin created successfully!\n")
		fmt.Printf("   Email: %s\n", superAdmin.Email)
		fmt.Printf("   ID: %s\n", superAdmin.ID)
		fmt.Printf("   Is Super Admin: %v\n", superAdmin.IsSuperAdmin)
		fmt.Printf("   Is Active: %v\n", superAdmin.IsActive)

		return nil
	})
})
