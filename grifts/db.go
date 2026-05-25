package grifts

import (
	"backend_server/models"
	"fmt"
	"log"

	"github.com/gobuffalo/grift/grift"
	"golang.org/x/crypto/bcrypt"
)

var _ = grift.Namespace("db", func() {

	// write seeder here
})
