package actions

import (
	"net/http"
	"strings"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"

	"backend_server/models"
)

type MediaCategoriesResource struct {
	buffalo.Resource
}

func (v MediaCategoriesResource) List(c buffalo.Context) error {
	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "no transaction found", nil)
	}

	mediaCategories := &models.MediaCategories{}
	q := PaginateFromContext(tx, c)

	search := strings.TrimSpace(c.Param("search"))
	if search == "" {
		search = strings.TrimSpace(c.Param("q"))
	}
	if search != "" {
		q = q.Where("LOWER(name) LIKE LOWER(?)", "%"+search+"%")
	}

	q = q.Order("name ASC")

	if err := q.All(mediaCategories); err != nil {
		return Response(c, http.StatusInternalServerError, "failed to fetch media categories", err.Error())
	}

	return Response(c, http.StatusOK, "success", map[string]interface{}{
		"data":       mediaCategories,
		"pagination": q.Paginator,
	})
}

func (v MediaCategoriesResource) Show(c buffalo.Context) error {
	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "no transaction found", nil)
	}

	mediaCategory := &models.MediaCategory{}
	if err := tx.Find(mediaCategory, c.Param("media_category_id")); err != nil {
		return Response(c, http.StatusNotFound, "media category not found", err.Error())
	}

	return Response(c, http.StatusOK, "success", mediaCategory)
}

func (v MediaCategoriesResource) Create(c buffalo.Context) error {
	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "no transaction found", nil)
	}

	mediaCategory := &models.MediaCategory{}
	if err := c.Bind(mediaCategory); err != nil {
		return Response(c, http.StatusBadRequest, "invalid input", err.Error())
	}

	verrs, err := tx.ValidateAndCreate(mediaCategory)
	if err != nil {
		return Response(c, http.StatusInternalServerError, "failed to create media category", err.Error())
	}

	if verrs.HasAny() {
		return Response(c, http.StatusUnprocessableEntity, "validation errors", verrs)
	}

	return Response(c, http.StatusCreated, "media category created successfully", mediaCategory)
}

func (v MediaCategoriesResource) Update(c buffalo.Context) error {
	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "no transaction found", nil)
	}

	mediaCategory := &models.MediaCategory{}
	if err := tx.Find(mediaCategory, c.Param("media_category_id")); err != nil {
		return Response(c, http.StatusNotFound, "media category not found", err.Error())
	}

	if err := c.Bind(mediaCategory); err != nil {
		return Response(c, http.StatusBadRequest, "invalid input", err.Error())
	}

	verrs, err := tx.ValidateAndUpdate(mediaCategory)
	if err != nil {
		return Response(c, http.StatusInternalServerError, "failed to update media category", err.Error())
	}

	if verrs.HasAny() {
		return Response(c, http.StatusUnprocessableEntity, "validation errors", verrs)
	}

	return Response(c, http.StatusOK, "media category updated successfully", mediaCategory)
}

func (v MediaCategoriesResource) Destroy(c buffalo.Context) error {
	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "no transaction found", nil)
	}

	mediaCategory := &models.MediaCategory{}
	if err := tx.Find(mediaCategory, c.Param("media_category_id")); err != nil {
		return Response(c, http.StatusNotFound, "media category not found", err.Error())
	}

	if err := tx.Destroy(mediaCategory); err != nil {
		return Response(c, http.StatusInternalServerError, "failed to delete media category", err.Error())
	}

	return Response(c, http.StatusOK, "media category deleted successfully", nil)
}