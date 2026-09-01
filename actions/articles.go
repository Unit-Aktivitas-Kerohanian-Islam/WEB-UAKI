package actions

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"

	"backend_server/models"
	"backend_server/storage"
)

var (
	nonAlphanumericRegex = regexp.MustCompile(`[^a-z0-9\s-]`)
	whitespaceRegex      = regexp.MustCompile(`[\s_]+`)
	multipleHyphenRegex  = regexp.MustCompile(`-+`)
)

func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = nonAlphanumericRegex.ReplaceAllString(s, "")
	s = whitespaceRegex.ReplaceAllString(s, "-")
	s = multipleHyphenRegex.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "article"
	}
	return s
}

func GenerateUniqueSlug(tx *pop.Connection, titleOrSlug string, currentID uuid.UUID) string {
	baseSlug := Slugify(titleOrSlug)
	slug := baseSlug
	counter := 1

	for {
		q := tx.Where("slug = ?", slug)
		if currentID != uuid.Nil {
			q = q.Where("id != ?", currentID)
		}
		exists, err := q.Exists(&models.Article{})
		if err != nil || !exists {
			break
		}
		slug = fmt.Sprintf("%s-%d", baseSlug, counter)
		counter++
	}

	return slug
}

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
	q := PaginateFromContext(tx, c)

	search := strings.TrimSpace(c.Param("search"))
	if search == "" {
		search = strings.TrimSpace(c.Param("q"))
	}
	if search != "" {
		q = q.Where("LOWER(title) LIKE LOWER(?) OR LOWER(value) LIKE LOWER(?)", "%"+search+"%", "%"+search+"%")
	}

	if category := strings.TrimSpace(c.Param("category")); category != "" && strings.ToLower(category) != "all" && strings.ToLower(category) != "semua" {
		q = q.Where("LOWER(category) = LOWER(?)", category)
	}

	if status := strings.TrimSpace(c.Param("status")); status != "" {
		if status == "active" || status == "true" {
			q = q.Where("is_active = ?", true)
		} else if status == "inactive" || status == "false" {
			q = q.Where("is_active = ?", false)
		}
	}

	q = q.Order("created_at DESC")

	if err := q.All(articles); err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to retrieve articles", nil)
	}

	activeCount, _ := tx.Where("is_active = true").Count(&models.Articles{})
	inactiveCount, _ := tx.Where("is_active = false").Count(&models.Articles{})
	totalArticles, _ := tx.Count(&models.Articles{})

	return Response(c, http.StatusOK, "Articles retrieved successfully", map[string]interface{}{
		"articles":   articles,
		"data":       articles,
		"pagination": q.Paginator,
		"summary": map[string]interface{}{
			"total":    totalArticles,
			"active":   activeCount,
			"inactive": inactiveCount,
		},
	})
}

func (v ArticlesResource) Show(c buffalo.Context) error {
	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "Database connection not found", nil)
	}

	param := c.Param("article_id")
	article := &models.Article{}
	var err error

	if _, parseErr := uuid.FromString(param); parseErr == nil {
		err = tx.Find(article, param)
	} else {
		err = tx.Where("slug = ?", param).First(article)
	}

	if err != nil {
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

	// Generate SEO-friendly unique slug
	if article.Slug != "" {
		article.Slug = GenerateUniqueSlug(tx, article.Slug, uuid.Nil)
	} else {
		article.Slug = GenerateUniqueSlug(tx, article.Title, uuid.Nil)
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

	param := c.Param("article_id")
	article := &models.Article{}
	var err error

	if _, parseErr := uuid.FromString(param); parseErr == nil {
		err = tx.Find(article, param)
	} else {
		err = tx.Where("slug = ?", param).First(article)
	}

	if err != nil {
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

	// Update slug if explicitly provided or if title changed
	if input.Slug != "" && input.Slug != article.Slug {
		article.Slug = GenerateUniqueSlug(tx, input.Slug, article.ID)
	} else if input.Title != "" && input.Title != article.Title && input.Slug == "" {
		article.Slug = GenerateUniqueSlug(tx, input.Title, article.ID)
	}

	article.Category = input.Category
	article.IsActive = input.IsActive
	if input.Title != "" {
		article.Title = input.Title
	}
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

	param := c.Param("article_id")
	article := &models.Article{}
	var err error

	if _, parseErr := uuid.FromString(param); parseErr == nil {
		err = tx.Find(article, param)
	} else {
		err = tx.Where("slug = ?", param).First(article)
	}

	if err != nil {
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