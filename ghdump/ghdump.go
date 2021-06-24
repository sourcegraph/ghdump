package ghdump

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

func Main() {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_ACCESS_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := githubv4.NewClient(tc)

	var search Search

	if data, err := os.ReadFile("search.json"); err != nil {
		if !os.IsNotExist(err) {
			log.Fatal(err)
		}
	} else if err := json.Unmarshal(data, &search); err != nil {
		log.Fatal(err)
	}

	if search == (Search{}) {
		search.Stars = StarRange{From: 200_000, To: 400_000}
	}

	type query struct {
		RateLimit struct {
			ResetAt githubv4.DateTime
			Remaining githubv4.Int
		}
		Search struct {
			RepositoryCount githubv4.Int
			PageInfo        struct {
				HasNextPage githubv4.Boolean
				EndCursor   githubv4.String
			}
			Nodes []struct {
				Repository struct {
					NameWithOwner   githubv4.String
					PrimaryLanguage struct {
						Name githubv4.String
					}
					StargazerCount githubv4.Int
				} `graphql:"... on Repository"`
			}
		} `graphql:"search(query: $query, type: $type, first: $first, after: $after)"`
	}

	for {
		var q query
		vars := map[string]interface{}{
			"query": githubv4.String(search.Query()),
			"type":  githubv4.SearchTypeRepository,
			"first": githubv4.Int(100),
			"after": search.Cursor,
		}

		began := time.Now()

		err := client.Query(context.Background(), &q, vars)
		if err != nil {
			log.Fatalf("Query error: %v", err)
		}


		log.Printf("INFO: Got %d repos matching %s, took %s with remaining rate limit of %d",
			len(q.Search.Nodes), search, time.Since(began), q.RateLimit.Remaining)

		switch {
		case q.RateLimit.Remaining == 0:
			log.Printf("WARNING: Exhausted GitHub rate limit, sleeping until %s", q.RateLimit.ResetAt)
			time.Sleep(q.RateLimit.ResetAt.Sub(time.Now()))
		case q.Search.RepositoryCount > 1000: // GitHub's Search API limit
			log.Printf("WARNING: Search %q yielded more than 1000 results, refining search and retrying", search)
			if search.Refine() {
				log.Printf("INFO: Refined search to %q", search)
				continue
			} else {
				log.Printf("WARNING: Couldn't refine search %q further. Some repos will be missed.", search)
			}
		}

		for _, n := range q.Search.Nodes {
			r := n.Repository
			fmt.Printf("github.com/%s,%s,%d\n", r.NameWithOwner, r.PrimaryLanguage.Name, r.StargazerCount)
		}

		if q.Search.PageInfo.HasNextPage {
			search.Cursor = &q.Search.PageInfo.EndCursor
		} else if !search.Next() {
			break
		}

		data, err := json.Marshal(search)
		if err != nil {
			log.Fatal(err)
		}

		if err = os.WriteFile("search.json", data, 0666); err != nil {
			log.Fatal(err)
		}
	}
}

type DateRange struct{ From, To time.Time }

const dateFormat = "2006-01-02"

func (r DateRange) String() string {
	return fmt.Sprintf("%s..%s",
		r.From.Format(dateFormat),
		r.To.Format(dateFormat),
	)
}

func (r DateRange) Size() time.Duration { return r.To.Sub(r.From) }

type StarRange struct{ From, To int }

func (r StarRange) String() string {
	return fmt.Sprintf("%d..%d", r.From, r.To)
}

func (r StarRange) Size() int { return r.To - r.From }

type Search struct {
	Stars   StarRange
	Created DateRange
	Cursor   *githubv4.String
}

var minCreated = time.Date(2008, time.January, 1, 0, 0, 0, 0, time.UTC)
const minStars = 13

func (s *Search) Next() bool {
	s.Cursor = nil
	switch {
	case s.Created != (DateRange{}) && s.Created.From.After(minCreated):
		size := s.Created.Size()
		s.Created.To = s.Created.From
		if s.Created.From = s.Created.To.Add(-size); s.Created.From.Before(minCreated) {
			s.Created.From = minCreated
		}
		return true
	case s.Created.From.Equal(minCreated):
		s.Created = newTopDateRange()
		fallthrough
	case s.Stars != (StarRange{}) && s.Stars.From > minStars:
		size := s.Stars.Size()
		s.Stars.To = s.Stars.From
		if s.Stars.From = s.Stars.To - size; s.Stars.From < minStars {
			s.Stars.From = minStars
		}
		return true
	}
	return false
}

func newTopDateRange() DateRange {
	now := time.Now()
	return DateRange{
		From: now.AddDate(-2, 0, 0),
		To: now,
	}
}

// Refine does one pass at refining the search to match <= 1000 repos.
func (s *Search) Refine() bool {
	if size := s.Stars.Size(); size > 1 {
		s.Stars.From += size / 2
		return true
	}

	if s.Created == (DateRange{}) {
		s.Created = newTopDateRange()
		return true
	}

	if size := s.Created.Size(); size >= 48 * time.Hour {
		s.Created.From = s.Created.From.Add(s.Created.Size() / 2)
		return true
	}

	// Can't refine beyond a single day.
	return false
}

func (s Search) Query() string {
	return strings.Join(s.terms(), " ")
}

func (s Search) String() string {
	terms := s.terms()

	if s.Cursor != nil {
		terms = append(terms, "cursor:" + (string)(*s.Cursor))
	}

	return strings.Join(terms, " ")
}

func (s Search) terms() []string {
	terms := []string{"sort:stars-desc"}

	if s.Stars != (StarRange{}) {
		terms = append(terms, fmt.Sprintf("stars:%s", s.Stars))
	}

	if s.Created != (DateRange{}) {
		terms = append(terms, fmt.Sprintf("created:%s", s.Created))
	}

	sort.Strings(terms)

	return terms
}
