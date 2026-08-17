package actions

import (
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"backend_server/models"
	"backend_server/storage"
)

type RegistrantsResource struct {
	buffalo.Resource
	storageService *storage.StorageService
}

func NewRegistrantsResource() RegistrantsResource {
	return RegistrantsResource{
		storageService: storage.NewStorageService(),
	}
}

func (v RegistrantsResource) List(c buffalo.Context) error {
	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "Database error", nil)
	}

	registrants := &models.Registrants{}
	q := tx.PaginateFromParams(c.Params())
	if err := q.All(registrants); err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to retrieve data", err.Error())
	}

	return Response(c, http.StatusOK, "Success", map[string]interface{}{
		"data":       registrants,
		"pagination": q.Paginator,
	})
}

func (v RegistrantsResource) Show(c buffalo.Context) error {
	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "Database error", nil)
	}

	registrant := &models.Registrant{}
	if err := tx.Find(registrant, c.Param("registrant_id")); err != nil {
		return Response(c, http.StatusNotFound, "Registrant not found", nil)
	}

	return Response(c, http.StatusOK, "Success", registrant)
}

// Create sekarang HANYA menerima application/json murni
func (v RegistrantsResource) Create(c buffalo.Context) error {
	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "Database error", nil)
	}

	registrant := &models.Registrant{}
	// Bind JSON ke struct
	if err := c.Bind(registrant); err != nil {
		return Response(c, http.StatusBadRequest, "Invalid JSON data", err.Error())
	}

	// Hash Password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(registrant.Password), 10)
	if err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to hash password", nil)
	}
	registrant.Password = string(hashedPassword)

	// Simpan ke Database
	verrs, err := tx.ValidateAndCreate(registrant)
	if err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to save data. NIM or Email might already exist.", err.Error())
	}
	if verrs.HasAny() {
		return Response(c, http.StatusUnprocessableEntity, "Validation error", verrs)
	}

	return Response(c, http.StatusCreated, "Registration successful", registrant)
}

func (v RegistrantsResource) Destroy(c buffalo.Context) error {
	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "Database error", nil)
	}

	registrant := &models.Registrant{}
	if err := tx.Find(registrant, c.Param("registrant_id")); err != nil {
		return Response(c, http.StatusNotFound, "Registrant not found", nil)
	}

	if registrant.CvUrl != "" {
		key := v.storageService.ExtractObjectKey(registrant.CvUrl)
		if err := v.storageService.Delete(c.Request().Context(), key); err != nil {
			log.Printf("⚠️ gagal hapus CV lokal: %v", err)
		}
	}

	if err := tx.Destroy(registrant); err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to delete", err.Error())
	}

	return Response(c, http.StatusOK, "Deleted successfully", nil)
}

// Endpoint Upload Terpisah (Sama seperti Artikel & Media)
func (v RegistrantsResource) UploadCV(c buffalo.Context) error {
	// Batasi ukuran request maks 3MB
	c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, 3<<20)

	file, err := c.File("cv")
	if err != nil {
		return Response(c, http.StatusBadRequest, "CV is required or file too large (Max 3MB)", err.Error())
	}
	defer file.Close()

	// Cek Ekstensi Wajib PDF
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".pdf" {
		return Response(c, http.StatusBadRequest, "Only PDF files are allowed", nil)
	}

	safeFilename := strings.ReplaceAll(file.Filename, " ", "_")
	url, err := v.storageService.Upload(c.Request().Context(), "cv/"+uuid.New().String()+"-"+safeFilename, file)
	if err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to save CV", err.Error())
	}

	return Response(c, http.StatusOK, "Upload success", map[string]string{
		"url": url,
	})
}

func (v RegistrantsResource) Login(c buffalo.Context) error {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.Bind(&input); err != nil {
		return Response(c, http.StatusBadRequest, "Invalid JSON body", nil)
	}

	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "Database error", nil)
	}

	registrant := &models.Registrant{}
	if err := tx.Where("email = ?", input.Email).First(registrant); err != nil {
		return Response(c, http.StatusUnauthorized, "Email not found", nil)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(registrant.Password), []byte(input.Password)); err != nil {
		return Response(c, http.StatusUnauthorized, "Wrong password", nil)
	}

	data := map[string]interface{}{
		"registrant_id": registrant.ID,
		"name":          registrant.Name,
		"nim":           registrant.NIM,
		"email":         registrant.Email,
		"token":         "mock_jwt_token_for_now_until_sso_is_ready",
	}

	return Response(c, http.StatusOK, "Login successful", data)
}