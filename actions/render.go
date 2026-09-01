package actions

import (
	"strconv"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/render"
	"github.com/gobuffalo/pop/v6"
)

var r *render.Engine

func init() {
	r = render.New(render.Options{
		DefaultContentType: "application/json",
	})
}

type APIResponse struct {
    Success bool        `json:"success"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

func Response(c buffalo.Context, status int, message string, data interface{}) error {
    success := status >= 200 && status < 300

    resp := APIResponse{
        Success: success,
        Message: message,
        Data:    data,
    }

    return c.Render(status, r.JSON(resp))
}

func PaginateFromContext(tx *pop.Connection, c buffalo.Context) *pop.Query {
	page, _ := strconv.Atoi(c.Param("page"))
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(c.Param("per_page"))
	if perPage < 1 {
		perPage, _ = strconv.Atoi(c.Param("limit"))
	}
	if perPage < 1 {
		perPage = 10
	}
	if perPage > 100 {
		perPage = 100
	}

	return tx.Paginate(page, perPage)
}