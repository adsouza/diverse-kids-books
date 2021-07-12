package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
        "log"
        "os"
	"strings"

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
		b := Book{
			Title: row[0],
			Authors: importCreators(ctx, fsc, row[1]),
		}
		if len(row[2]) > 0 {
			b.Illustrators = importCreators(ctx, fsc, row[2])
		}
		if _, err := fsc.Collection("books").Doc(fmt.Sprintf("%s by %s", b.Title, row[1])).Set(ctx, b); err != nil {
			log.Fatal(err)
		}
		log.Println(row[0])
	}
	fmt.Println("--")
}

func importCreators(ctx context.Context, fsc *firestore.Client, names string) []*firestore.DocumentRef {
	var refs []*firestore.DocumentRef
	creators := strings.Split(names, " &\n")
	for _, n := range creators {
		i := Creator{Name: n}
		ref := fsc.Collection("creators").Doc(n)
		if _, err := ref.Get(ctx); err != nil {
			if _, err := ref.Set(ctx, i); err != nil {
				log.Fatal(err)
			}
		}
		refs = append(refs, ref)
	}
	return refs
}

type Creator struct {
	Name string
}

type Book struct {
	Title string
	Authors, Illustrators []*firestore.DocumentRef
}

func createClient(ctx context.Context) *firestore.Client {
        const projectID = "diverse-kids-books"
        client, err := firestore.NewClient(ctx, projectID)
        if err != nil {
                log.Fatalf("Failed to create client: %v", err)
        }
        return client
}
