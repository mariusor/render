package render

import (
	"bytes"
	"html/template"
	"io"
	"net/http"
)

// Engine is the generic interface for all responses.
type Engine interface {
	Render(io.Writer, interface{}) error
}

// Head defines the basic ContentType and Status fields.
type Head struct {
	ContentType string
	Status      int
}

// HTML built-in renderer.
type HTML struct {
	Head
	Name      string
	Templates *template.Template

	bp GenericBufferPool
}

// Write outputs the header content.
func (h Head) Write(w http.ResponseWriter) {
	w.Header().Set(ContentType, h.ContentType)
	w.WriteHeader(h.Status)
}

// Render a HTML response.
func (h HTML) Render(w io.Writer, binding interface{}) error {
	var buf *bytes.Buffer
	if h.bp != nil {
		// If we have a bufferpool, allocate from it
		buf = h.bp.Get()
		defer h.bp.Put(buf)
	}

	err := h.Templates.ExecuteTemplate(buf, h.Name, binding)
	if err != nil {
		return err
	}

	if hw, ok := w.(http.ResponseWriter); ok {
		h.Head.Write(hw)
	}
	_, _ = buf.WriteTo(w)

	return nil
}
