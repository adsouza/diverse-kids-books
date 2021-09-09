package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

func main() {
	ctx := context.Background()
	fsc := createClient(ctx)
	defer fsc.Close()

	// Ensure that data can be loaded from Firestore & parsed.
	titlesByCreator, err := titlesByCreatorForAgeWithTag(ctx, fsc, age(), tag())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Authors & their books:\n---------------------------")
	var n int
	for c, titles := range titlesByCreator {
		if len(titles.Wrote) == 0 {
			continue
		}
		fmt.Printf("%s:\n\t%s\n", c, strings.Join(titles.Wrote, "\n\t"))
		n += len(titles.Wrote)
	}
	fmt.Printf("Found a total of %d books.\n", n)

	// Load HTML templates & configure HTTP request handler.
	tmpl := template.Must(template.ParseGlob("*.tmpl"))
	http.Handle("/", &handler{fsc: fsc, tmpl: tmpl})

	// Start server.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}
	log.Printf("Listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

type handler struct {
	fsc  *firestore.Client
	tmpl *template.Template
}

func (h *handler) respond(w http.ResponseWriter, d *bookList) {
	if err := h.tmpl.ExecuteTemplate(w, "index.tmpl", d); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err)
	}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ageStr := r.FormValue("age")
	if ageStr == "" {
		h.respond(w, nil)
		return
	}
	age, err := strconv.Atoi(ageStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, err)
		return
	}
	titlesByCreator, err := titlesByCreatorForAgeWithTag(ctx, h.fsc, age, tag())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err)
		return
	}
	books := bookList{
		Age:    age,
		Titles: titlesByCreator,
	}
	h.respond(w, &books)
}

type bookList struct {
	Age    int
	Titles map[string]titles
}

type titles struct {
	Illustrated, Wrote []string
}

func titlesByCreatorForAgeWithTag(ctx context.Context, fsc *firestore.Client, age int, tag string) (map[string]titles, error) {
	titlesByCreator := map[string]titles{}
	books := fsc.Collection("books").Where("MinAge", "<=", age).Where("Tags", "array-contains", tag).Documents(ctx)
	for {
		bSnap, err := books.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return titlesByCreator, fmt.Errorf("failed to iterate over books: %v", err)
		}
		var b Book
		bSnap.DataTo(&b)
		if b.MaxAge < age {
			// Skip books that would be too simple for the specified age.
			continue
		}
		for _, i := range b.Illustrators {
			t := titlesByCreator[i.ID]
			t.Illustrated = append(t.Illustrated, b.Title)
			titlesByCreator[i.ID] = t
		}
		for _, i := range b.Authors {
			t := titlesByCreator[i.ID]
			t.Wrote = append(t.Wrote, b.Title)
			titlesByCreator[i.ID] = t
		}
	}
	return titlesByCreator, nil
}

func age() int {
	const defaultAge = 6
	if len(os.Args) < 2 {
		return defaultAge
	}
	if age, err := strconv.Atoi(os.Args[1]); err != nil {
		log.Fatalf("Unable to parse 1st arg as int: %s.", err)
	} else {
		log.Printf("Using %d as the target age...\n", age)
		return age
	}
	return defaultAge
}

func tag() string {
	const defaultTag = "melanated"
	if len(os.Args) < 3 {
		return defaultTag
	}
	return os.Args[2]
}

type Book struct {
	Title                 string
	Authors, Illustrators []*firestore.DocumentRef
	MinAge, MaxAge        int
}

type Creator struct {
	Name               string
	illustrated, wrote []*Book
}

func createClient(ctx context.Context) *firestore.Client {
	const projectID = "diverse-kids-books"
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	return client
}
