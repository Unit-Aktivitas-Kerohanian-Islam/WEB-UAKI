package models

import (
	"encoding/json"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gobuffalo/validate/v3/validators"
	"github.com/gofrs/uuid"
)

type Registrant struct {
	ID              uuid.UUID    `json:"id" db:"id"`
	Name            string       `json:"name" db:"name"`
	Email           string       `json:"email" db:"email"`
	NIM             nulls.String `json:"nim" db:"nim"`
	Angkatan        nulls.String `json:"angkatan" db:"angkatan"`
	Prodi           nulls.String `json:"prodi" db:"prodi"`
	Fakultas        nulls.String `json:"fakultas" db:"fakultas"`
	Domicile        nulls.String `json:"domicile" db:"domicile"`
	Phone           nulls.String `json:"phone" db:"phone"`
	Password        nulls.String `json:"password,omitempty" db:"password"`
	Division1       nulls.String `json:"division_1" db:"division_1"`
	Division2       nulls.String `json:"division_2" db:"division_2"`
	SwotS           nulls.String `json:"swot_s" db:"swot_s"`
	SwotW           nulls.String `json:"swot_w" db:"swot_w"`
	SwotO           nulls.String `json:"swot_o" db:"swot_o"`
	SwotT           nulls.String `json:"swot_t" db:"swot_t"`
	OrganizationExp nulls.String `json:"organization_exp" db:"organization_exp"`
	Commitment      nulls.String `json:"commitment" db:"commitment"`
	CvUrl           nulls.String `json:"cv_url" db:"cv_url"`
	Status          string       `json:"status" db:"status"` 
	
	// Jadwal Screening
	ScreeningDate     nulls.Time   `json:"screening_date" db:"screening_date"`
	ScreeningLocation nulls.String `json:"screening_location" db:"screening_location"`
	ScreeningLink     nulls.String `json:"screening_link" db:"screening_link"`

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
		&validators.StringIsPresent{Field: r.Email, Name: "Email"},
	), nil
}

func (r *Registrant) ValidateCreate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

func (r *Registrant) ValidateUpdate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}