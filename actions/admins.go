package actions

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/x/responder"
	"golang.org/x/crypto/bcrypt"

	"backend_server/models"
)

type AdminsResource struct {
	buffalo.Resource
}

func (v AdminsResource) List(c buffalo.Context) error {
	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return fmt.Errorf("no transaction found")
	}

	admins := &models.Admins{}

	q := PaginateFromContext(tx, c)

	search := strings.TrimSpace(c.Param("search"))
	if search == "" {
		search = strings.TrimSpace(c.Param("q"))
	}
	if search != "" {
		q = q.Where("LOWER(email) LIKE LOWER(?)", "%"+search+"%")
	}

	status := strings.TrimSpace(c.Param("status"))
	if status == "active" || status == "true" {
		q = q.Where("is_active = ?", true)
	} else if status == "inactive" || status == "false" {
		q = q.Where("is_active = ?", false)
	}

	q = q.Order("created_at DESC")

	if err := q.All(admins); err != nil {
		return Response(c, http.StatusInternalServerError, err.Error(), nil)
	}

	activeCount, _ := tx.Where("is_active = ?", true).Count(&models.Admin{})
	inactiveCount, _ := tx.Where("is_active = ?", false).Count(&models.Admin{})
	totalAdmins := q.Paginator.TotalEntriesSize
	if totalAdmins == 0 {
		totalAdmins, _ = tx.Count(&models.Admin{})
	}

	return Response(c, http.StatusOK, "Admins retrieved successfully", map[string]interface{}{
		"admins":     admins,
		"data":       admins,
		"pagination": q.Paginator,
		"summary": map[string]interface{}{
			"total":    totalAdmins,
			"active":   activeCount,
			"inactive": inactiveCount,
		},
	})
}

func (v AdminsResource) Show(c buffalo.Context) error {
	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return fmt.Errorf("no transaction found")
	}

	admin := &models.Admin{}

	if err := tx.Find(admin, c.Param("admin_id")); err != nil {
		return Response(c, http.StatusNotFound, "Admin not found", nil)
	}

	return Response(c, http.StatusOK, "Admin retrieved successfully", map[string]interface{}{
		"admin": admin,
	})
}

func (v AdminsResource) Create(c buffalo.Context) error {
	admin := &models.Admin{}

	if err := c.Bind(admin); err != nil {
		return Response(c, http.StatusBadRequest, "Invalid request body", nil)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(admin.Password), 10)
	if err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to hash password", nil)
	}
	admin.Password = string(hashedPassword)

	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "No database transaction found", nil)
	}

	verrs, err := tx.ValidateAndCreate(admin)
	if err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to create admin", nil)
	}

	if verrs.HasAny() {
		return Response(c, http.StatusUnprocessableEntity, "Validation failed", verrs)
	}

	return Response(c, http.StatusCreated, "Admin created successfully", admin)
}

func (v AdminsResource) Update(c buffalo.Context) error {
	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return Response(c, http.StatusInternalServerError, "No database transaction found", nil)
	}

	admin := &models.Admin{}
	if err := tx.Find(admin, c.Param("admin_id")); err != nil {
		return Response(c, http.StatusNotFound, "Admin not found", nil)
	}

	if err := c.Bind(admin); err != nil {
		return Response(c, http.StatusBadRequest, "Invalid request body", nil)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(admin.Password), 10)
	if err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to hash password", nil)
	}
	admin.Password = string(hashedPassword)

	verrs, err := tx.ValidateAndUpdate(admin)
	if err != nil {
		return Response(c, http.StatusInternalServerError, "Failed to update admin", nil)
	}

	if verrs.HasAny() {
		return Response(c, http.StatusUnprocessableEntity, "Validation failed", verrs)
	}

	return Response(c, http.StatusOK, "Admin updated successfully", admin)
}

func (v AdminsResource) Destroy(c buffalo.Context) error {
	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return fmt.Errorf("no transaction found")
	}

	admin := &models.Admin{}

	if err := tx.Find(admin, c.Param("admin_id")); err != nil {
		return c.Error(http.StatusNotFound, err)
	}

	if err := tx.Destroy(admin); err != nil {
		return err
	}

	return responder.Wants("html", func(c buffalo.Context) error {
		c.Flash().Add("success", T.Translate(c, "admin.destroyed.success"))

		return c.Redirect(http.StatusSeeOther, "/admins")
	}).Wants("json", func(c buffalo.Context) error {
		return c.Render(http.StatusOK, r.JSON(admin))
	}).Wants("xml", func(c buffalo.Context) error {
		return c.Render(http.StatusOK, r.XML(admin))
	}).Respond(c)
}

func (v AdminsResource) Login(c buffalo.Context) error {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.Bind(&input); err != nil {
		return c.Error(http.StatusBadRequest, err)
	}

	tx, ok := c.Value("tx").(*pop.Connection)
	if !ok {
		return fmt.Errorf("no transaction found")
	}

	admin := &models.Admin{}
	if err := tx.Where("email = ?", input.Email).First(admin); err != nil {
		return Response(c, http.StatusInternalServerError, "Admin not found", nil)
	}

	if !admin.IsActive {
		return Response(c, http.StatusForbidden, "Admin account is inactive", nil)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(input.Password)); err != nil {
		return Response(c, http.StatusInternalServerError, "Wrong password", nil)
	}

	token, err := JWTService.CreateJWTToken(admin.ID, "admin")
	if err != nil {
		return Response(c, http.StatusInternalServerError, "Error generating token", nil)
	}

	admin.LastLogin = time.Now()
	tx.Update(admin)

	data := map[string]interface{}{
		"admin_id":       admin.ID,
		"is_super_admin": admin.IsSuperAdmin,
		"is_active":      admin.IsActive,
		"token":          token,
	}

	return Response(c, http.StatusOK, "Login successfully", data)
}
