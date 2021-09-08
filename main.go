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
	titlesByCreator, err := titlesByCreatorForAgeWithTag(ctx, fsc, age(), tag())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Authors & their books:\n---------------------------")
	var n int
	for c, titles := range titlesByCreator {
		if len(titles.wrote) == 0 {
			continue
		}
		fmt.Printf("%s:\n\t%s\n", c, strings.Join(titles.wrote, "\n\t"))
		n+=len(titles.wrote)
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

type titles struct {
	illustrated, wrote []string
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
			t.illustrated = append(t.illustrated, b.Title)
			titlesByCreator[i.ID] = t
		}
		for _, i := range b.Authors {
			t := titlesByCreator[i.ID]
			t.wrote = append(t.wrote, b.Title)
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
	Title string
	Authors, Illustrators []*firestore.DocumentRef
	MinAge, MaxAge int
}

type Creator struct {
	Name string
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
