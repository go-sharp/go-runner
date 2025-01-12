package main

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed assets/css/*
var assetsFS embed.FS

// Template cache
var templates = template.Must(template.ParseFS(templatesFS, "templates/*.html"))

var caser = cases.Title(language.German)

func main() {
	// Static assets handler
	http.Handle("/assets/", http.FileServer(http.FS(assetsFS)))

	// Routes
	http.HandleFunc("/", servePage("index"))
	http.HandleFunc("/about", servePage("about"))
	http.HandleFunc("/contact", servePage("contact"))

	fmt.Println(templates.DefinedTemplates())

	// Start server
	log.Println("Server running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func servePage(page string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		err := templates.ExecuteTemplate(&buf, page, nil)

		if err != nil {
			http.Error(w, "Page not found", http.StatusNotFound)
			return
		}

		content, _ := io.ReadAll(&buf)
		data := struct {
			Title   string
			Content template.HTML
		}{
			Title:   caser.String(page),
			Content: template.HTML(content),
		}

		err = templates.ExecuteTemplate(w, "base", data)
		if err != nil {
			fmt.Println(err)
		}
	}
}
