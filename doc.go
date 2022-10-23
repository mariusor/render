/*Package render is a package that provides functionality for easily rendering JSON, XML, binary data, and HTML templates.

  package main

  import (
      "encoding/xml"
      "net/http"

      "github.com/mariusor/render"
  )

  func main() {
      r := render.New()
      mux := http.NewServeMux()

      mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
          w.Write([]byte("Welcome, visit sub pages now."))
      })

      mux.HandleFunc("/html", func(w http.ResponseWriter, req *http.Request) {
          // Assumes you have a template in ./templates called "example.tmpl".
          // $ mkdir -p templates && echo "<h1>Hello HTML world.</h1>" > templates/example.tmpl
          r.HTML(w, http.StatusOK, "example", nil)
      })

      http.ListenAndServe("127.0.0.1:3000", mux)
  }
*/
package render
