package models

import (
	"encoding/json"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

type Media struct {
	ID         uuid.UUID `json:"id" db:"id"`
	AdminID    uuid.UUID `json:"admin_id" db:"admin_id"`
	CategoryID int       `json:"category_id" db:"category_id"`
	Title      string    `json:"title" db:"title"`
	Img_Url    string    `json:"img_url" db:"img_url"`
	
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

func (m Media) String() string {
	jm, _ := json.Marshal(m)
	return string(jm)
}

type Medias []Media

func (m Medias) String() string {
	jm, _ := json.Marshal(m)
	return string(jm)
}

func (m *Media) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

func (m *Media) ValidateCreate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

func (m *Media) ValidateUpdate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}
