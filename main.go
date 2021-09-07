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
	titlesByIllustrator, err := booksByIllustratorForAge(ctx, fsc, age())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Illustrators & their books:\n---------------------------")
	var n int
	for c, titles := range titlesByIllustrator {
		if len(titles) == 0 {
			continue
		}
		fmt.Printf("%s:\n\t%s\n", c, strings.Join(titles, "\n\t"))
		n+=len(titles)
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

func booksByIllustratorForAge(ctx context.Context, fsc *firestore.Client, age int) (map[string][]string, error) {
	titlesByIllustrator := map[string][]string{}
	creators := fsc.Collection("creators").Documents(ctx)
	for {
		cSnap, err := creators.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate over creators: %v", err)
		}
		var c Creator
		cSnap.DataTo(&c)
		titlesByIllustrator[c.Name] = []string{}
	}
	books := fsc.Collection("books").Where("MinAge", "<=", age).Documents(ctx)
	for {
		bSnap, err := books.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return titlesByIllustrator, fmt.Errorf("failed to iterate over books: %v", err)
		}
		var b Book
		bSnap.DataTo(&b)
		if b.MaxAge < age {
			// Skip books that would be too simple for the specified age.
			continue
		}
		for _, i := range b.Illustrators {
			titlesByIllustrator[i.ID] = append(titlesByIllustrator[i.ID], b.Title)
		}
	}
	return titlesByIllustrator, nil
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
	books []*Book
}

func createClient(ctx context.Context) *firestore.Client {
        const projectID = "diverse-kids-books"
        client, err := firestore.NewClient(ctx, projectID)
        if err != nil {
                log.Fatalf("Failed to create client: %v", err)
        }
        return client
}
