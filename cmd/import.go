package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
        "log"
        "os"
	"strconv"
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
		if len(row[3]) > 0 {
			if b.MinAge, err = strconv.Atoi(row[3]); err != nil {
				log.Fatal(err)
			}
		}
		if len(row[4]) > 0 {
			if b.MaxAge, err = strconv.Atoi(row[4]); err != nil {
				log.Fatal(err)
			}
		}
		if len(row[5]) > 0 {
			b.Tags = tagsFromAppearance(row[5])
		}
		if _, err := fsc.Collection("books").Doc(fmt.Sprintf("%s by %s", b.Title, row[1])).Set(ctx, b); err != nil {
			log.Fatal(err)
		}
		log.Println(row[0])
	}
	fmt.Println("--")
}

func tagsFromAppearance(a string) []string {
	switch a {
	case "Asian":
		return []string{"east-asian"}
	case "dark":
		return []string{"melanated"}
	case "diverse":
		return []string{"diverse"}
	case "indigenous":
		return []string{"indigenous"}
	case "non-human":
		return []string{}
	case "pale":
		return []string{"pale"}
	default:
		log.Printf("Unsupported appearance type: %s.", a)
	}
	return nil
}

func importCreators(ctx context.Context, fsc *firestore.Client, names string) []*firestore.DocumentRef {
	var refs []*firestore.DocumentRef
	creators := strings.Split(names, " &\n")
	var people []string
	for _, c := range creators {
		if !strings.Contains(c, " & ") {
			people  = append(people, c)
			continue
		}
		// Handle pairs of creators (typically family) who share a last name.
		couple := strings.Split(c, " & ")
		if len(couple) != 2 {
			log.Fatalf("Only 2 people allowed per couple: %s", c)
		}
		pieces := strings.Split(couple[1], " ")
		// Does not account for unhypenated multi-word family names (e.g. von or del)
		last := pieces[len(pieces)-1]
		people = append(people, fmt.Sprintf("%s %s", couple[0], last))
		people = append(people, couple[1])
	}
	for _, p := range people {
		i := Creator{Name: p}
		ref := fsc.Collection("creators").Doc(p)
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
	MinAge, MaxAge int
	Tags []string
}

func createClient(ctx context.Context) *firestore.Client {
        const projectID = "diverse-kids-books"
        client, err := firestore.NewClient(ctx, projectID)
        if err != nil {
                log.Fatalf("Failed to create client: %v", err)
        }
        return client
}
