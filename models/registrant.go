package models

import (
	"encoding/json"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gobuffalo/validate/v3/validators"
	"github.com/gofrs/uuid"
)

type Registrant struct {
	ID              uuid.UUID `json:"id" db:"id"`
	Name            string    `json:"name" db:"name"`
	NIM             string    `json:"nim" db:"nim"`
	Angkatan        string    `json:"angkatan" db:"angkatan"`
	Prodi           string    `json:"prodi" db:"prodi"`
	Fakultas        string    `json:"fakultas" db:"fakultas"`
	Domicile        string    `json:"domicile" db:"domicile"`
	Phone           string    `json:"phone" db:"phone"`
	Email           string    `json:"email" db:"email"`
	Password        string    `json:"-" db:"password"` // Jangan return password di JSON
	Division1       string    `json:"division_1" db:"division_1"`
	Division2       string    `json:"division_2" db:"division_2"`
	SwotS           string    `json:"swot_s" db:"swot_s"`
	SwotW           string    `json:"swot_w" db:"swot_w"`
	SwotO           string    `json:"swot_o" db:"swot_o"`
	SwotT           string    `json:"swot_t" db:"swot_t"`
	OrganizationExp string    `json:"organization_exp" db:"organization_exp"`
	Commitment      string    `json:"commitment" db:"commitment"`
	CvUrl           string    `json:"cv_url" db:"cv_url"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

func (r Registrant) String() string {
	jr, _ := json.Marshal(r)
	return string(jr)
}

type Registrants []Registrant

func (r Registrants) String() string {
	jr, _ := json.Marshal(r)
	return string(jr)
}

func (r *Registrant) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.Validate(
		&validators.StringIsPresent{Field: r.Name, Name: "Name"},
		&validators.StringIsPresent{Field: r.NIM, Name: "NIM"},
		&validators.StringIsPresent{Field: r.Email, Name: "Email"},
		&validators.StringIsPresent{Field: r.Password, Name: "Password"},
		&validators.StringIsPresent{Field: r.Division1, Name: "Division 1"},
	), nil
}

func (r *Registrant) ValidateCreate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

func (r *Registrant) ValidateUpdate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}