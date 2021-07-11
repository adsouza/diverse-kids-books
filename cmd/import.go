package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
        "log"
        "os"

	"cloud.google.com/go/firestore"
)

func main() {
	ctx := context.Background()
	fsc := createClient(ctx)
	defer fsc.Close()

	// Read data from CSV & write it to FireStore.
	f, err := os.Open("/tmp/diverse-kids-books.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		// Skip books that lack illustrators.
		if row[2] == "" {
			continue
		}
		i := Creator{Name: row[2]}
		iRef := fsc.Collection("creators").Doc(i.Name)
		if _, err := iRef.Get(ctx); err != nil {
			if _, err := iRef.Set(ctx, i); err != nil {
				log.Fatal(err)
			}
		}
		b := Book{
			Title: row[0],
			Illustrator: iRef,
		}
		if _, err := fsc.Collection("books").Doc(fmt.Sprintf("%s by %s", b.Title, row[1])).Set(ctx, b); err != nil {
			log.Fatal(err)
		}
		log.Println(row[0])
	}
	fmt.Println("--")
}

type Creator struct {
	Name string
}

type Book struct {
	Title string
	Illustrator *firestore.DocumentRef
}

func createClient(ctx context.Context) *firestore.Client {
        const projectID = "diverse-kids-books"
        client, err := firestore.NewClient(ctx, projectID)
        if err != nil {
                log.Fatalf("Failed to create client: %v", err)
        }
        return client
}
