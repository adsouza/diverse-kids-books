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
	var (
		c Creator
		b Book
	)
	creators := fsc.Collection("creators").Documents(ctx)
	for {
		cSnap, err := creators.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to iterate over creators: %v", err)
		}
		cSnap.DataTo(&c)
		var titles []string
		books := fsc.Collection("books").Where("Illustrator", "==", cSnap.Ref).Documents(ctx)
		for {
			bSnap, err := books.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Fatalf("Failed to iterate over books for %s: %v", c.Name, err)
			}
			bSnap.DataTo(&b)
			titles = append(titles, b.Title)
		}
		fmt.Printf("%s: %s.\n", c.Name, strings.Join(titles, ", "))
	}

	// Start server.
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
	Illustrator *firestore.DocumentRef
}

type Creator struct {
	Name string
}

func createClient(ctx context.Context) *firestore.Client {
        const projectID = "diverse-kids-books"
        client, err := firestore.NewClient(ctx, projectID)
        if err != nil {
                log.Fatalf("Failed to create client: %v", err)
        }
        return client
}
