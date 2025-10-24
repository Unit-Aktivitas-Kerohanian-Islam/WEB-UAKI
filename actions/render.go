package actions

import (

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/render"
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