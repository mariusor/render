package render

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	// ContentHTML header value for HTML data.
	ContentHTML = "text/html"
	// ContentLength header constant.
	ContentLength = "Content-Length"
	// ContentType header constant.
	ContentType = "Content-Type"
	// Default character encoding.
	defaultCharset = "UTF-8"
)

// helperFuncs had to be moved out. See helpers.go|helpers_pre16.go files.

// Delims represents a set of Left and Right delimiters for HTML template rendering.
type Delims struct {
	// Left delimiter, defaults to {{.
	Left string
	// Right delimiter, defaults to }}.
	Right string
}

// Options is a struct for specifying configuration options for the render.Render object.
type Options struct {
	// Directory to load templates. Default is "templates".
	Directory string
	// FileSystem to access files
	FileSystem fs.FS
	// Layout template name. Will not render a layout if blank (""). Defaults to blank ("").
	Layout string
	// Extensions to parse template files from. Defaults to [".tmpl"].
	Extensions []string
	// Funcs is a slice of FuncMaps to apply to the template upon compilation. This is useful for helper functions. Defaults to empty map.
	Funcs []template.FuncMap
	// Delims sets the action delimiters to the specified strings in the Delims struct.
	Delims Delims
	// Appends the given character set to the Content-Type header. Default is "UTF-8".
	Charset string
	// If DisableCharset is set to true, it will not append the above Charset value to the Content-Type header. Default is false.
	DisableCharset bool
	// Prefixes the JSON output with the given bytes. Default is false.
	PrefixJSON []byte
	// Allows changing the binary content type.
	HTMLContentType string
	// If IsDevelopment is set to true, this will recompile the templates on every request. Default is false.
	IsDevelopment bool
	// If UseMutexLock is set to true, the standard `sync.RWMutex` lock will be used instead of the lock free implementation. Default is false.
	// Note that when `IsDevelopment` is true, the standard `sync.RWMutex` lock is always used. Lock free is only a production feature.
	UseMutexLock bool
	// Unescape HTML characters "&<>" to their original values. Default is false.
	UnEscapeHTML bool
	// Require that all partials executed in the layout are implemented in all templates using the layout. Default is false.
	RequirePartials bool
	// Disables automatic rendering of http.StatusInternalServerError when an error occurs. Default is false.
	DisableHTTPErrorRendering bool
	// Enables using partials without the current filename suffix which allows use of the same template in multiple files. e.g {{ partial "carosuel" }} inside the home template will match carosel-home or carosel.
	// ***NOTE*** - This option should be named RenderPartialsWithoutSuffix as that is what it does. "Prefix" is a typo. Maintaining the existing name for backwards compatibility.
	RenderPartialsWithoutPrefix bool
	// BufferPool to use when rendering HTML templates. If none is supplied
	// defaults to SizedBufferPool of size 32 with 512KiB buffers.
	BufferPool GenericBufferPool
}

// HTMLOptions is a struct for overriding some rendering Options for specific HTML call.
type HTMLOptions struct {
	// Layout template name. Overrides Options.Layout.
	Layout string
	// Funcs added to Options.Funcs.
	Funcs template.FuncMap
}

// Render is a service that provides functions for easily writing JSON, XML,
// binary data, and HTML templates out to a HTTP Response.
type Render struct {
	lock rwLock

	// Customize Secure with an Options struct.
	opt             Options
	templates       *template.Template
	compiledCharset string
}

// New constructs a new Render instance with the supplied options.
func New(options ...Options) *Render {
	var o Options
	if len(options) > 0 {
		o = options[0]
	}

	r := Render{opt: o}

	r.prepareOptions()

	return &r
}

func (r *Render) prepareOptions() {
	// Fill in the defaults if need be.
	if len(r.opt.Charset) == 0 {
		r.opt.Charset = defaultCharset
	}
	if !r.opt.DisableCharset {
		r.compiledCharset = "; charset=" + r.opt.Charset
	}
	if len(r.opt.Directory) == 0 {
		r.opt.Directory = "templates"
	}
	if r.opt.FileSystem == nil {
		r.opt.FileSystem = os.DirFS(".")
	}
	if len(r.opt.Extensions) == 0 {
		r.opt.Extensions = []string{".tmpl"}
	}
	if len(r.opt.HTMLContentType) == 0 {
		r.opt.HTMLContentType = ContentHTML
	}
	if r.opt.BufferPool == nil {
		r.opt.BufferPool = NewSizedBufferPool(32, 1<<19) // 32 buffers of size 512KiB each
	}
	if r.opt.IsDevelopment || r.opt.UseMutexLock {
		r.lock = &sync.RWMutex{}
	} else {
		r.lock = &emptyLock{}
	}
}

func (r *Render) CompileTemplates() error {
	return r.compileTemplatesFromDir()
}

func (r *Render) compileTemplatesFromDir() error {
	dir := r.opt.Directory
	r.templates = template.New(dir)
	r.templates.Delims(r.opt.Delims.Left, r.opt.Delims.Right)

	r.lock.Lock()
	defer r.lock.Unlock()

	// Add our funcmaps.
	for _, funcs := range r.opt.Funcs {
		r.templates.Funcs(funcs)
	}
	r.templates.Funcs(helperFuncs)
	// Walk the supplied directory and compile any files that match our extension list.
	// We don't use template.ParseFS because it uses a different logic for naming the children templates.
	return fs.WalkDir(r.opt.FileSystem, dir, func(path string, info fs.DirEntry, err error) error {
		if info == nil || info.IsDir() || err != nil {
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		ext := ""
		if strings.Contains(rel, ".") {
			ext = filepath.Ext(rel)
		}

		for _, extension := range r.opt.Extensions {
			if ext == extension {
				var buf []byte
				buf, err = fs.ReadFile(r.opt.FileSystem, path)
				if err != nil {
					return err
				}

				name := rel[0 : len(rel)-len(ext)]
				tmpl := r.templates.New(filepath.ToSlash(name))

				// Break out if this parsing fails. We don't want any silent server starts.
				if tmpl, err = tmpl.Parse(string(buf)); err != nil {
					break
				}
			}
		}
		return err
	})
}

// TemplateLookup is a wrapper around template.Lookup and returns
// the template with the given name that is associated with t, or nil
// if there is no such template.
func (r *Render) TemplateLookup(t string) *template.Template {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.templates.Lookup(t)
}

func (r *Render) execute(templates *template.Template, name string, binding interface{}) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	return buf, templates.ExecuteTemplate(buf, name, binding)
}

func (r *Render) layoutFuncs(templates *template.Template, name string, binding interface{}) template.FuncMap {
	return template.FuncMap{
		"yield": func() (template.HTML, error) {
			buf, err := r.execute(templates, name, binding)
			// Return safe HTML here since we are rendering our own template.
			return template.HTML(buf.String()), err
		},
		"current": func() (string, error) {
			return name, nil
		},
		"partial": func(partialName string) (template.HTML, error) {
			fullPartialName := fmt.Sprintf("%s-%s", partialName, name)
			if templates.Lookup(fullPartialName) == nil && r.opt.RenderPartialsWithoutPrefix {
				fullPartialName = partialName
			}
			if r.opt.RequirePartials || templates.Lookup(fullPartialName) != nil {
				buf, err := r.execute(templates, fullPartialName, binding)
				// Return safe HTML here since we are rendering our own template.
				return template.HTML(buf.String()), err
			}
			return "", nil
		},
	}
}

func (r *Render) prepareHTMLOptions(htmlOpt []HTMLOptions) HTMLOptions {
	layout := r.opt.Layout
	funcs := template.FuncMap{}

	for _, tmp := range r.opt.Funcs {
		for k, v := range tmp {
			funcs[k] = v
		}
	}

	if len(htmlOpt) > 0 {
		opt := htmlOpt[0]
		if len(opt.Layout) > 0 {
			layout = opt.Layout
		}

		for k, v := range opt.Funcs {
			funcs[k] = v
		}
	}

	return HTMLOptions{
		Layout: layout,
		Funcs:  funcs,
	}
}

// Render is the generic function called by XML, JSON, Data, HTML, and can be called by custom implementations.
func (r *Render) Render(w io.Writer, e Engine, data interface{}) error {
	err := e.Render(w, data)
	if hw, ok := w.(http.ResponseWriter); err != nil && !r.opt.DisableHTTPErrorRendering && ok {
		http.Error(hw, err.Error(), http.StatusInternalServerError)
	}
	return err
}

// HTML builds up the response from the specified template and bindings.
func (r *Render) HTML(w io.Writer, status int, name string, binding interface{}, htmlOpt ...HTMLOptions) error {
	opt := r.prepareHTMLOptions(htmlOpt)

	// If we are in development mode, recompile the templates on every HTML request.
	if r.opt.IsDevelopment {
		r.templates = nil
	}

	r.lock.RLock() // rlock here because we're reading the hasWatcher
	if r.templates == nil {
		r.lock.RUnlock() // runlock here because CompileTemplates will lock

		if len(opt.Funcs) > 0 {
			r.opt.Funcs = append(r.opt.Funcs, opt.Funcs)
		}
		if err := r.CompileTemplates(); err != nil {
			return err
		}
		r.lock.RLock()
	}
	templates := r.templates
	if len(opt.Funcs) > 0 {
		templates.Funcs(opt.Funcs)
	}
	r.lock.RUnlock()

	if tpl := templates.Lookup(name); tpl != nil {
		if len(opt.Layout) > 0 {
			tpl.Funcs(r.layoutFuncs(templates, name, binding))
			name = opt.Layout
		}
	}

	head := Head{
		ContentType: r.opt.HTMLContentType + r.compiledCharset,
		Status:      status,
	}

	h := HTML{
		Head:      head,
		Name:      name,
		Templates: templates,
		bp:        r.opt.BufferPool,
	}

	return r.Render(w, h, binding)
}
