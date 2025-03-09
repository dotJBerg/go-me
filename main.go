package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

type Post struct {
	PostName string
	Title    string
	Date     time.Time
	Content  template.HTML
}

type PageData struct {
	ActivePage string
	Posts      []Post
}

func mdToHTML(md []byte) []byte {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	result := markdown.Render(doc, renderer)
	return result
}

func loadMarkdownPosts() ([]Post, error) {
	var posts []Post
	files, err := filepath.Glob("posts/*.md")
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		postName := strings.TrimSuffix(filepath.Base(file), ".md")

		content, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		lines := strings.Split(string(content), "\n")
		if len(lines) < 2 {
			log.Println("Is the file empty?")
			continue
		}

		rawTitle := strings.TrimSpace(strings.TrimPrefix(lines[0], "#"))

		// Improvement: This is a bit weird. Maybe grabbing the date could be simpler or done a better way.
		rawDate := strings.TrimSpace(lines[1])
		rawDate = strings.TrimPrefix(rawDate, "<small>")
		rawDate = strings.TrimSuffix(rawDate, "</small>")
		rawDate = strings.TrimSpace(rawDate)

		parsedDate, err := time.Parse("2 January 2006", rawDate)
		if err != nil {
			log.Println("No date!")
		}

		body := strings.Join(lines[2:], "\n")
		htmlContent := mdToHTML([]byte(body))

		post := Post{
			PostName: postName,
			Title:    rawTitle,
			Date:     parsedDate,
			Content:  template.HTML(htmlContent),
		}
		posts = append(posts, post)
	}

	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Date.After(posts[j].Date)
	})
	return posts, nil
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		ActivePage: "home",
	}
	tmpl := template.Must(template.ParseFiles("templates/index.html"))
	tmpl.Execute(w, data)
}

func blogHandler(w http.ResponseWriter, r *http.Request) {
	posts, err := loadMarkdownPosts()
	if err != nil {
		http.Error(w, "Error loading posts", http.StatusInternalServerError)
		return
	}
	data := PageData{
		ActivePage: "blog",
		Posts:      posts,
	}
	tmpl := template.Must(template.ParseFiles("templates/blog.html"))
	tmpl.Execute(w, data)
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	postName := strings.TrimPrefix(r.URL.Path, "/post/")
	filePath := "posts/" + postName + ".md"

	content, err := os.ReadFile(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	htmlContent := mdToHTML(content)
	post := Post{Title: postName, Content: template.HTML(htmlContent)}

	tmpl := template.Must(template.ParseFiles("templates/post.html"))
	tmpl.Execute(w, post)
}

func main() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/blog", blogHandler)
	http.HandleFunc("/post/", postHandler)

	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
