package main

import (
	"context"
	"fmt"
        "log"
        "net/http"
        "os"
	"strconv"
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
		n int
	)
	creators := fsc.Collection("creators").Documents(ctx)
	age := age()
	fmt.Println("Illustrators & their books:\n---------------------------")
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
		books := fsc.Collection("books").Where("Illustrators", "array-contains", cSnap.Ref).Where("MinAge", "<=", age).Documents(ctx)
		for {
			bSnap, err := books.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Fatalf("Failed to iterate over books for %s: %v", c.Name, err)
			}
			bSnap.DataTo(&b)
			if b.MaxAge < age {
				// Skip books that would be too simple for the specified age.
				continue
			}
			titles = append(titles, b.Title)
			n++
		}
		if len(titles) == 0 {
			continue
		}
		fmt.Printf("%s:\n\t%s\n", c.Name, strings.Join(titles, "\n\t"))
	}
	fmt.Printf("Found a total of %d books.\n", n)

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

type Book struct {
	Title string
	Authors, Illustrators []*firestore.DocumentRef
	MinAge, MaxAge int
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
