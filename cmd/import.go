package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
        "log"
        "os"
	"strings"

	"google.golang.org/api/iterator"
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

	// Read data from FireStore & print it to console.
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
