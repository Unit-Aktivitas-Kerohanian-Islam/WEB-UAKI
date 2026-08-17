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

	for i := range *registrants {
		(*registrants)[i].Password = ""
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

	registrant.Password = ""
	return Response(c, http.StatusOK, "Success", registrant)
}

func (v RegistrantsResource) Create(c buffalo.Context) error {
	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "Database error", nil)
	}

	registrant := &models.Registrant{}
	if err := c.Bind(registrant); err != nil {
		return Response(c, http.StatusBadRequest, "Invalid JSON data", err.Error())
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(registrant.Password), 10)
	if err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to hash password", nil)
	}
	registrant.Password = string(hashedPassword)
	registrant.Status = "PENDING" // Pastikan default aman

	verrs, err := tx.ValidateAndCreate(registrant)
	if err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to save data. NIM or Email might already exist.", err.Error())
	}
	if verrs.HasAny() {
		return Response(c, http.StatusUnprocessableEntity, "Validation error", verrs)
	}

	registrant.Password = ""
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

func (v RegistrantsResource) UploadCV(c buffalo.Context) error {
	c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, 3<<20)

	file, err := c.File("cv")
	if err != nil {
		return Response(c, http.StatusBadRequest, "CV is required or file too large (Max 3MB)", err.Error())
	}
	defer file.Close()

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

	token, err := JWTService.CreateJWTToken(registrant.ID)
	if err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to generate token", err.Error())
	}

	data := map[string]interface{}{
		"registrant_id": registrant.ID,
		"name":          registrant.Name,
		"nim":           registrant.NIM,
		"email":         registrant.Email,
		"status":        registrant.Status,
		"token":         token,
	}

	return Response(c, http.StatusOK, "Login successful", data)
}

// Fitur Baru: Melihat Profil Sendiri (Token Based)
func (v RegistrantsResource) GetMe(c buffalo.Context) error {
	userIDStr, ok := c.Value("admin_id").(string)
	if !ok || userIDStr == "" {
		return Response(c, http.StatusUnauthorized, "Unauthorized", nil)
	}

	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "Database error", nil)
	}

	registrant := &models.Registrant{}
	if err := tx.Find(registrant, userIDStr); err != nil {
		return Response(c, http.StatusNotFound, "Registrant not found", nil)
	}

	registrant.Password = ""
	return Response(c, http.StatusOK, "Success", registrant)
}

// Fitur Baru: Edit Pendaftaran Sendiri
func (v RegistrantsResource) UpdateMe(c buffalo.Context) error {
	userIDStr, ok := c.Value("admin_id").(string)
	if !ok || userIDStr == "" {
		return Response(c, http.StatusUnauthorized, "Unauthorized", nil)
	}

	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "Database error", nil)
	}

	registrant := &models.Registrant{}
	if err := tx.Find(registrant, userIDStr); err != nil {
		return Response(c, http.StatusNotFound, "Registrant not found", nil)
	}

	var input models.Registrant
	if err := c.Bind(&input); err != nil {
		return Response(c, http.StatusBadRequest, "Invalid JSON data", err.Error())
	}

	// Ganti file CV lama jika url berubah
	if input.CvUrl != "" && input.CvUrl != registrant.CvUrl {
		if registrant.CvUrl != "" {
			key := v.storageService.ExtractObjectKey(registrant.CvUrl)
			v.storageService.Delete(c.Request().Context(), key)
		}
		registrant.CvUrl = input.CvUrl
	}

	// Hanya kolom ini yang diizinkan untuk di-update pendaftar
	registrant.Name = input.Name
	registrant.Angkatan = input.Angkatan
	registrant.Prodi = input.Prodi
	registrant.Fakultas = input.Fakultas
	registrant.Domicile = input.Domicile
	registrant.Phone = input.Phone
	registrant.Division1 = input.Division1
	registrant.Division2 = input.Division2
	registrant.SwotS = input.SwotS
	registrant.SwotW = input.SwotW
	registrant.SwotO = input.SwotO
	registrant.SwotT = input.SwotT
	registrant.OrganizationExp = input.OrganizationExp
	registrant.Commitment = input.Commitment

	verrs, err := tx.ValidateAndUpdate(registrant)
	if err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to update profile", err.Error())
	}
	if verrs.HasAny() {
		return Response(c, http.StatusUnprocessableEntity, "Validation error", verrs)
	}

	registrant.Password = ""
	return Response(c, http.StatusOK, "Profile updated successfully", registrant)
}

// Fitur Baru: Admin Update Status Pendaftar
func (v RegistrantsResource) UpdateStatus(c buffalo.Context) error {
	var input struct {
		Status string `json:"status"`
	}
	if err := c.Bind(&input); err != nil {
		return Response(c, http.StatusBadRequest, "Invalid JSON body", nil)
	}

	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "Database error", nil)
	}

	registrant := &models.Registrant{}
	if err := tx.Find(registrant, c.Param("registrant_id")); err != nil {
		return Response(c, http.StatusNotFound, "Registrant not found", nil)
	}

	registrant.Status = input.Status
	if err := tx.Update(registrant); err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to update status", err.Error())
	}

	registrant.Password = ""
	return Response(c, http.StatusOK, "Status updated successfully", registrant)
}