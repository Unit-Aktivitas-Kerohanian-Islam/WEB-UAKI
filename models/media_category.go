package models

import (
	"encoding/json"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
)

type MediaCategory struct {
	ID   int    `json:"id" db:"id"`
	Name string `json:"name" db:"name"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

func (m MediaCategory) String() string {
	jm, _ := json.Marshal(m)
	return string(jm)
}

type MediaCategories []MediaCategory

func (m MediaCategories) String() string {
	jm, _ := json.Marshal(m)
	return string(jm)
}

func (m *MediaCategory) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

func (m *MediaCategory) ValidateCreate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

func (m *MediaCategory) ValidateUpdate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}
