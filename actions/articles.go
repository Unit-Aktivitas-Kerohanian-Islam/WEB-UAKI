package actions

import (
	"log"
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"

	"backend_server/models"
	"backend_server/storage"
)

type ArticlesResource struct {
	buffalo.Resource
	storageService *storage.StorageService
}

func NewArticleResource() ArticlesResource {
	return ArticlesResource{
		storageService: storage.NewStorageService(),
	}
}

func (v ArticlesResource) List(c buffalo.Context) error {
	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "Database connection not found", nil)
	}

	articles := &models.Articles{}
	q := tx.PaginateFromParams(c.Params())

	if err := q.All(articles); err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to retrieve articles", nil)
	}

	return Response(c, http.StatusOK, "Articles retrieved successfully", articles)
}

func (v ArticlesResource) Show(c buffalo.Context) error {
	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "Database connection not found", nil)
	}

	article := &models.Article{}
	if err := tx.Find(article, c.Param("article_id")); err != nil {
		return Response(c, http.StatusNotFound, "Article not found", nil)
	}

	return Response(c, http.StatusOK, "Article retrieved successfully", article)
}

func (v ArticlesResource) Create(c buffalo.Context) error {
	article := &models.Article{}

	if err := c.Bind(article); err != nil {
		return Response(c, http.StatusBadRequest, "Invalid article data", nil)
	}

	// PERBAIKAN: Ubah admin_id menjadi user_id
	userIDStr, ok := c.Value("user_id").(string)
	if ok && userIDStr != "" {
		adminID, _ := uuid.FromString(userIDStr)
		article.AdminID = adminID
	}

	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "Database connection not found", nil)
	}

	verrs, err := tx.ValidateAndCreate(article)
	if err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to create article", err.Error())
	}
	if verrs.HasAny() {
		return Response(c, http.StatusUnprocessableEntity, "Validation error", verrs)
	}

	return Response(c, http.StatusCreated, "Article created successfully", article)
}

func (v ArticlesResource) Update(c buffalo.Context) error {
	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "Database connection not found", nil)
	}

	article := &models.Article{}
	if err := tx.Find(article, c.Param("article_id")); err != nil {
		return Response(c, http.StatusNotFound, "Article not found", nil)
	}

	oldImage := article.ImgURL
	var input models.Article
	if err := c.Bind(&input); err != nil {
		return Response(c, http.StatusBadRequest, "Invalid article data", nil)
	}

	if input.ImgURL != "" && input.ImgURL != oldImage {
		if oldImage != "" {
			key := v.storageService.ExtractObjectKey(oldImage)
			if key != "" {
				if err := v.storageService.Delete(c.Request().Context(), key); err != nil {
					log.Printf("⚠️ gagal hapus file lokal: %v", err)
				}
			}
		}
	}

	article.Category = input.Category
	article.IsActive = input.IsActive
	article.Title = input.Title
	article.Value = input.Value
	article.ImgURL = input.ImgURL

	verrs, err := tx.ValidateAndUpdate(article)
	if err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to update article", nil)
	}
	if verrs.HasAny() {
		return Response(c, http.StatusUnprocessableEntity, "Validation error", verrs)
	}

	return Response(c, http.StatusOK, "Article updated successfully", article)
}

func (v ArticlesResource) Destroy(c buffalo.Context) error {
	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "Database connection not found", nil)
	}

	article := &models.Article{}
	if err := tx.Find(article, c.Param("article_id")); err != nil {
		return Response(c, http.StatusNotFound, "Article not found", nil)
	}

	if article.ImgURL != "" {
		key := v.storageService.ExtractObjectKey(article.ImgURL)
		if err := v.storageService.Delete(c.Request().Context(), key); err != nil {
			log.Printf("⚠️ gagal hapus file lokal: %v", err)
		}
	}

	if err := tx.Destroy(article); err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to delete article", nil)
	}

	return Response(c, http.StatusOK, "Article deleted successfully", nil)
}

func (v ArticlesResource) UploadImage(c buffalo.Context) error {
	file, err := c.File("image")
	if err != nil {
		return Response(c, http.StatusBadRequest, "failed to read image file", err.Error())
	}
	defer file.Close()

	// 2. Generate UUID menggunakan library gofrs
	uid, _ := uuid.NewV4()
	url, err := v.storageService.Upload(c.Request().Context(), "articles/"+uid.String()+"-"+file.Filename, file)
	if err != nil {
		return Response(c, http.StatusInternalServerError, "failed to upload image", err.Error())
	}

	return Response(c, http.StatusOK, "upload success", map[string]string{
		"url": url,
	})
}