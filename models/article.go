package models

import (
	"encoding/json"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

type Article struct {
	ID       uuid.UUID `json:"id" db:"id"`
	AdminID  uuid.UUID `json:"admin_id" db:"admin_id"`
	Category string    `json:"category" db:"category"`
	IsActive bool      `json:"is_active" db:"is_active"`
	Title    string    `json:"title" db:"title"`
	Slug     string    `json:"slug" db:"slug"`
	Value    string    `json:"value" db:"value"`
	ImgURL   string    `json:"img_url" db:"img_url"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

func (a Article) String() string {
	ja, _ := json.Marshal(a)
	return string(ja)
}

type Articles []Article

func (a Articles) String() string {
	ja, _ := json.Marshal(a)
	return string(ja)
}

func (a *Article) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

func (a *Article) ValidateCreate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

func (a *Article) ValidateUpdate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}
