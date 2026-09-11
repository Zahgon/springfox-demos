package web

import (
	"net/http"

	"github.com/springfox/springfox-demos/internal/api"
	"github.com/springfox/springfox-demos/internal/web"
)

// FileUploadController is springfoxdemo.boot.swagger.web.FileUploadController.
type FileUploadController struct{}

// NewFileUploadController creates the controller.
func NewFileUploadController() *FileUploadController { return &FileUploadController{} }

// Register mounts POST /upload.
func (c *FileUploadController) Register(app *web.App) {
	app.Handle(web.Route{
		Method: http.MethodPost, Pattern: "/upload",
		Consumes: []string{web.MediaTypeMultipart},
		Produces: []string{web.MediaTypeAll},
		Handler:  c.uploadFile,
		Op: api.Operation{
			Name: "uploadFile", Tag: "file-upload-controller", Summary: "uploadFile",
			Consumes: []string{web.MediaTypeMultipart},
			Produces: []string{web.MediaTypeAll},
			// Springfox reports the two @RequestPart arguments in this order,
			// with no description and not required.
			Params: []api.Parameter{
				{Name: "file", In: api.InFormData, Schema: api.Schema{Type: api.TypeFile}},
				{Name: "description", In: api.InFormData, Schema: api.Schema{Type: api.TypeString}},
			},
			Responses: []api.Response{{Code: http.StatusOK, Description: "OK"}},
		},
	})
}

// uploadFile answers POST /upload with an empty 200 once both parts are present.
func (c *FileUploadController) uploadFile(r *web.Request) (web.ResponseEntity, error) {
	if _, err := r.RequirePart("description"); err != nil {
		return web.ResponseEntity{}, err
	}
	if _, err := r.RequireFilePart("file"); err != nil {
		return web.ResponseEntity{}, err
	}
	return web.OKEmpty(), nil
}
