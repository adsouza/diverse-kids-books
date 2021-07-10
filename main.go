package main

import (
	"context"
	"fmt"
        "log"
        "net/http"
        "os"
	"strings"

	"google.golang.org/api/iterator"
	"cloud.google.com/go/firestore"
)

func main() {
	ctx := context.Background()
	fsc := createClient(ctx)
	defer fsc.Close()
	var c Creator
	iter := fsc.Collection("creators").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to iterate: %v", err)
		}
		doc.DataTo(&c)
		books, err := c.FetchBooks(ctx)
		if err != nil {
			log.Fatalf("%v", err)
		}
		if len(books) == 0 {
			continue
		}
		var titles []string
		for _, b := range books {
			titles = append(titles, b.Title)
		}
		fmt.Printf("%s: %s.\n", c.Name, strings.Join(titles, ", "))
	}

	port := os.Getenv("PORT")
        if port == "" {
                port = "8080"
                log.Printf("Defaulting to port %s", port)
        }
        log.Printf("Listening on port %s", port)
        if err := http.ListenAndServe(":"+port, nil); err != nil {
                log.Fatal(err)
        }
}

type Book struct {
	Title string
}

type Creator struct {
	Name string
	Books []*firestore.DocumentRef
}

func (c *Creator) FetchBooks(ctx context.Context) ([]*Book, error) {
	var books []*Book
	for _, r := range c.Books {
		d, err := r.Get(ctx)
		if err != nil {
			return nil, err
		}
		var b Book
		d.DataTo(&b)
		books = append(books, &b)
	}
	return books, nil
}

func createClient(ctx context.Context) *firestore.Client {
        const projectID = "diverse-kids-books"
        client, err := firestore.NewClient(ctx, projectID)
        if err != nil {
                log.Fatalf("Failed to create client: %v", err)
        }
        return client
}
