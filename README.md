# Render [![GoDoc](http://godoc.org/github.com/mariusor/render?status.svg)](http://godoc.org/github.com/mariusor/render) [![Test](https://github.com/mariusor/render/workflows/Test/badge.svg?branch=v1)](https://github.com/mariusor/render/actions)


Render is a package that provides functionality for easily rendering JSON, XML, text, binary data, and HTML templates.

## Usage
Render can be used with pretty much any web framework providing you can access the `http.ResponseWriter` from your handler. The rendering functions simply wraps Go's existing functionality for marshaling and rendering data.

- HTML: Uses the [html/template](http://golang.org/pkg/html/template/) package to render HTML templates.

~~~ go
// main.go
package main

import (
    "encoding/xml"
    "net/http"

    "github.com/mariusor/render"
)

func main() {
    r := render.New()
    mux := http.NewServeMux()

    mux.HandleFunc("/html", func(w http.ResponseWriter, req *http.Request) {
        // Assumes you have a template in ./templates called "example.tmpl"
        // $ mkdir -p templates && echo "<h1>Hello {{.}}.</h1>" > templates/example.tmpl
        r.HTML(w, http.StatusOK, "example", "World")
    })

    http.ListenAndServe("127.0.0.1:3000", mux)
}
~~~

~~~ html
<!-- templates/example.tmpl -->
<h1>Hello {{.}}.</h1>
~~~

### Available Options
Render comes with a variety of configuration options _(Note: these are not the default option values. See the defaults below.)_:

~~~ go
// ...
r := render.New(render.Options{
    Directory: "templates", // Specify what path to load the templates from.
    FileSystem: fs.Dir("."), // Specify filesystem from where files are loaded.
    Asset: func(name string) ([]byte, error) { // Load from an Asset function instead of file.
      return []byte("template content"), nil
    },
    AssetNames: func() []string { // Return a list of asset names for the Asset function
      return []string{"filename.tmpl"}
    },
    Layout: "layout", // Specify a layout template. Layouts can call {{ yield }} to render the current template or {{ partial "css" }} to render a partial from the current template.
    Extensions: []string{".tmpl", ".html"}, // Specify extensions to load for templates.
    Funcs: []template.FuncMap{AppHelpers}, // Specify helper function maps for templates to access.
    Delims: render.Delims{"{[{", "}]}"}, // Sets delimiters to the specified strings.
    Charset: "UTF-8", // Sets encoding for content-types. Default is "UTF-8".
    DisableCharset: true, // Prevents the charset from being appended to the content type header.
    HTMLContentType: "application/xhtml+xml", // Output XHTML content type instead of default "text/html".
    IsDevelopment: true, // Render will now recompile the templates on every HTML response.
    UseMutexLock: true, // Overrides the default no lock implementation and uses the standard `sync.RWMutex` lock.
    UnEscapeHTML: true, // Replace ensure '&<>' are output correctly (JSON only).
    StreamingJSON: true, // Streams the JSON response via json.Encoder.
    RequirePartials: true, // Return an error if a template is missing a partial used in a layout.
    DisableHTTPErrorRendering: true, // Disables automatic rendering of http.StatusInternalServerError when an error occurs.
})
// ...
~~~

### Default Options
These are the preset options for Render:

~~~ go
r := render.New()

// Is the same as the default configuration options:

r := render.New(render.Options{
    Directory: "templates",
    FileSystem: &LocalFileSystem{},
    Asset: nil,
    AssetNames: nil,
    Layout: "",
    Extensions: []string{".tmpl"},
    Funcs: []template.FuncMap{},
    Delims: render.Delims{"{{", "}}"},
    Charset: "UTF-8",
    DisableCharset: false,
    HTMLContentType: "text/html",
    IsDevelopment: false,
    UseMutexLock: false,
    UnEscapeHTML: false,
    StreamingJSON: false,
    RequirePartials: false,
    DisableHTTPErrorRendering: false,
    RenderPartialsWithoutPrefix: false,
    BufferPool: GenericBufferPool,
})
~~~

### JSON vs Streaming JSON
By default, Render does **not** stream JSON to the `http.ResponseWriter`. It instead marshalls your object into a byte array, and if no errors occurred, writes that byte array to the `http.ResponseWriter`. O

### Loading Templates
By default Render will attempt to load templates with a '.tmpl' extension from the "templates" directory. Templates are found by traversing the templates directory and are named by path and basename. For instance, the following directory structure:

~~~
templates/
  |_ admin/
  |    |__ index.tmpl
  |    |__ edit.tmpl
  |__ home.tmpl
~~~

Will provide the following templates:
~~~
admin/index
admin/edit
home
~~~

Templates can be loaded from an `embed.FS`.

~~~ go
// ...

//go:embed templates/*.html templates/*.tmpl
var embeddedTemplates embed.FS

// ...

r := render.New(render.Options{
    Directory: "templates",
    FileSystem: embeddedTemplates,
    Extensions: []string{".html", ".tmpl"},
})
// ...
~~~

You can also load templates from memory by providing the `Asset` and `AssetNames` options,
e.g. when generating an asset file using [go-bindata](https://github.com/jteeuwen/go-bindata).

### Layouts
Render provides `yield` and `partial` functions for layouts to access:
~~~ go
// ...
r := render.New(render.Options{
    Layout: "layout",
})
// ...
~~~

~~~ html
<!-- templates/layout.tmpl -->
<html>
  <head>
    <title>My Layout</title>
    <!-- Render the partial template called `css-$current_template` here -->
    {{ partial "css" }}
  </head>
  <body>
    <!-- render the partial template called `header-$current_template` here -->
    {{ partial "header" }}
    <!-- Render the current template here -->
    {{ yield }}
    <!-- render the partial template called `footer-$current_template` here -->
    {{ partial "footer" }}
  </body>
</html>
~~~

`current` can also be called to get the current template being rendered.
~~~ html
<!-- templates/layout.tmpl -->
<html>
  <head>
    <title>My Layout</title>
  </head>
  <body>
    This is the {{ current }} page.
  </body>
</html>
~~~

Partials are defined by individual templates as seen below. The partial template's
name needs to be defined as "{partial name}-{template name}".
~~~ html
<!-- templates/home.tmpl -->
{{ define "header-home" }}
<h1>Home</h1>
{{ end }}

{{ define "footer-home"}}
<p>The End</p>
{{ end }}
~~~

By default, the template is not required to define all partials referenced in the
layout. If you want an error to be returned when a template does not define a
partial, set `Options.RequirePartials = true`.

### Character Encodings
Render will automatically set the proper Content-Type header based on which function you call. See below for an example of what the default settings would output (note that UTF-8 is the default, and binary data does not output the charset):
~~~ go
// main.go
package main

import (
    "encoding/xml"
    "net/http"

    "github.com/mariusor/render"
)

func main() {
    r := render.New(render.Options{})
    mux := http.NewServeMux()

    // This will set the Content-Type header to "text/html; charset=UTF-8".
    mux.HandleFunc("/html", func(w http.ResponseWriter, req *http.Request) {
        // Assumes you have a template in ./templates called "example.tmpl"
        // $ mkdir -p templates && echo "<h1>Hello {{.}}.</h1>" > templates/example.tmpl
        r.HTML(w, http.StatusOK, "example", "World")
    })

    http.ListenAndServe("127.0.0.1:3000", mux)
}
~~~

In order to change the charset, you can set the `Charset` within the `render.Options` to your encoding value:
~~~ go
// main.go
package main

import (
    "encoding/xml"
    "net/http"

    "github.com/mariusor/render"
)

func main() {
    r := render.New(render.Options{
        Charset: "ISO-8859-1",
    })
    mux := http.NewServeMux()

    // This will set the Content-Type header to "text/html; charset=ISO-8859-1".
    mux.HandleFunc("/html", func(w http.ResponseWriter, req *http.Request) {
        // Assumes you have a template in ./templates called "example.tmpl"
        // $ mkdir -p templates && echo "<h1>Hello {{.}}.</h1>" > templates/example.tmpl
        r.HTML(w, http.StatusOK, "example", "World")
    })

    http.ListenAndServe("127.0.0.1:3000", mux)
}
~~~

### Error Handling

The rendering functions return any errors from the rendering engine.
By default, they will also write the error to the HTTP response and set the status code to 500. You can disable
this behavior so that you can handle errors yourself by setting
`Options.DisableHTTPErrorRendering: true`.

~~~go
r := render.New(render.Options{
  DisableHTTPErrorRendering: true,
})

//...

err := r.HTML(w, http.StatusOK, "example", "World")
if err != nil{
  http.Redirect(w, r, "/my-custom-500", http.StatusFound)
}
~~~

## Integration Examples

### [Echo](https://github.com/labstack/echo)
~~~ go
// main.go
package main

import (
    "io"
    "net/http"

    "github.com/labstack/echo"
    "github.com/mariusor/render"
)

type RenderWrapper struct { // We need to wrap the renderer because we need a different signature for echo.
    rnd *render.Render
}

func (r *RenderWrapper) Render(w io.Writer, name string, data interface{},c echo.Context) error {
    return r.rnd.HTML(w, 0, name, data) // The zero status code is overwritten by echo.
}

func main() {
    r := &RenderWrapper{render.New()}

    e := echo.New()

    e.Renderer = r

    e.GET("/", func(c echo.Context) error {
        return c.Render(http.StatusOK, "TemplateName", "TemplateData")
    })

    e.Logger.Fatal(e.Start("127.0.0.1:8080"))
}
~~~

