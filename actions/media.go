package actions

import (
	"log"
	"net/http"
	"strings"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"

	"backend_server/models"
	"backend_server/storage"
)

type MediaResource struct {
	buffalo.Resource
	storageService *storage.StorageService
}

func NewMediaResource() MediaResource {
	return MediaResource{
		storageService: storage.NewStorageService(),
	}
}

func (v MediaResource) List(c buffalo.Context) error {
	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "no transaction found", nil)
	}

	media := &[]models.Media{}
	q := PaginateFromContext(tx, c)

	search := strings.TrimSpace(c.Param("search"))
	if search == "" {
		search = strings.TrimSpace(c.Param("q"))
	}
	if search != "" {
		q = q.Where("LOWER(title) LIKE LOWER(?)", "%"+search+"%")
	}

	if categoryID := strings.TrimSpace(c.Param("category_id")); categoryID != "" && categoryID != "0" && strings.ToUpper(categoryID) != "ALL" && strings.ToLower(categoryID) != "semua" {
		q = q.Where("category_id = ?", categoryID)
	}

	q = q.Order("created_at DESC")

	if err := q.All(media); err != nil {
		return Response(c, http.StatusInternalServerError, "failed to retrieve media", err.Error())
	}

	return Response(c, http.StatusOK, "success", map[string]interface{}{
		"media":      media,
		"data":       media,
		"pagination": q.Paginator,
	})
}

func (v MediaResource) Show(c buffalo.Context) error {
	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "no transaction found", nil)
	}

	media := &models.Media{}
	if err := tx.Find(media, c.Param("media_id")); err != nil {
		return Response(c, http.StatusNotFound, "media not found", err.Error())
	}

	return Response(c, http.StatusOK, "success", media)
}

func (v MediaResource) Create(c buffalo.Context) error {
	media := &models.Media{}
	if err := c.Bind(media); err != nil {
		return Response(c, http.StatusBadRequest, "invalid request body", err.Error())
	}

	// PERBAIKAN: Ubah admin_id menjadi user_id
	userIDStr, ok := c.Value("user_id").(string)
	if ok && userIDStr != "" {
		adminID, _ := uuid.FromString(userIDStr)
		media.AdminID = adminID
	}

	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "no transaction found", nil)
	}

	verrs, err := tx.ValidateAndCreate(media)
	if err != nil {
		return Response(c, http.StatusInternalServerError, "failed to create media", err.Error())
	}
	if verrs.HasAny() {
		return Response(c, http.StatusUnprocessableEntity, "validation failed", verrs)
	}

	return Response(c, http.StatusCreated, "media created successfully", media)
}
func (v MediaResource) Update(c buffalo.Context) error {
	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "Database connection not found", nil)
	}

	media := &models.Media{}
	if err := tx.Find(media, c.Param("media_id")); err != nil {
		return Response(c, http.StatusNotFound, "Media not found", nil)
	}

	oldFile := media.Img_Url
	var input models.Media
	if err := c.Bind(&input); err != nil {
		return Response(c, http.StatusBadRequest, "Invalid media data", nil)
	}

	if input.Img_Url != "" && input.Img_Url != oldFile {
		if oldFile != "" {
			key := v.storageService.ExtractObjectKey(oldFile)
			if key != "" {
				if err := v.storageService.Delete(c.Request().Context(), key); err != nil {
					log.Printf("⚠️ gagal hapus file lama lokal: %v", err)
				}
			}
		}
	}

	media.Title = input.Title
	media.Img_Url = input.Img_Url
	media.CategoryID = input.CategoryID

	verrs, err := tx.ValidateAndUpdate(media)
	if err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to update media", err.Error())
	}
	if verrs.HasAny() {
		return Response(c, http.StatusUnprocessableEntity, "Validation error", verrs)
	}

	return Response(c, http.StatusOK, "Media updated successfully", media)
}

func (v MediaResource) Destroy(c buffalo.Context) error {
	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "no transaction found", nil)
	}

	media := &models.Media{}
	if err := tx.Find(media, c.Param("media_id")); err != nil {
		return Response(c, http.StatusNotFound, "media not found", err.Error())
	}

	if media.Img_Url != "" {
		key := v.storageService.ExtractObjectKey(media.Img_Url)
		if err := v.storageService.Delete(c.Request().Context(), key); err != nil {
			log.Printf("⚠️ gagal hapus file lokal: %v", err)
		}
	}

	if err := tx.Destroy(media); err != nil {
		return Response(c, http.StatusInternalServerError, "failed to delete media", err.Error())
	}

	return Response(c, http.StatusOK, "media deleted successfully", nil)
}

func (v MediaResource) UploadImage(c buffalo.Context) error {
	file, err := c.File("image")
	if err != nil {
		return Response(c, http.StatusBadRequest, "failed to read image file", err.Error())
	}
	defer file.Close()

	// 2. Generate UUID menggunakan library gofrs
	uid, _ := uuid.NewV4()
	url, err := v.storageService.Upload(c.Request().Context(), "media/"+uid.String()+"-"+file.Filename, file)
	if err != nil {
		return Response(c, http.StatusInternalServerError, "failed to upload image", err.Error())
	}

	return Response(c, http.StatusOK, "upload success", map[string]string{
		"url": url,
	})
}