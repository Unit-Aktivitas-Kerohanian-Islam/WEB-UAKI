package models

import (
	"encoding/json"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

type Admin struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	Password     string    `json:"password" db:"password"`
	IsSuperAdmin bool      `json:"is_super_admin" db:"is_super_admin"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	LastLogin    time.Time `json:"last_login" db:"last_login"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

func (a Admin) String() string {
	ja, _ := json.Marshal(a)
	return string(ja)
}

type Admins []Admin

func (a Admins) String() string {
	ja, _ := json.Marshal(a)
	return string(ja)
}

func (a *Admin) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

func (a *Admin) ValidateCreate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

func (a *Admin) ValidateUpdate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}
