package ghdump

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type searchParams struct {
	language string
	minStars int
	perPage  int
	page     int
}

func filename(p searchParams) string {
	return fmt.Sprintf("lang-%s__star-%d__ppg-%d__pg-%d.json", p.language, p.minStars, p.perPage, p.page)
}

var paramsFromFilenameRegexp = regexp.MustCompile(`lang\-(?P<language>[^_]+)__star\-(?P<stars>[0-9]+)__ppg\-(?P<perPage>[0-9]+)__pg\-(?P<page>[0-9]+)\.json`)

func paramsFromFilename(fn string) *searchParams {
	matches := paramsFromFilenameRegexp.FindStringSubmatch(fn)
	if len(matches) == 0 {
		return nil
	}
	minStars, _ := strconv.Atoi(matches[2])
	perPage, _ := strconv.Atoi(matches[3])
	page, _ := strconv.Atoi(matches[4])
	return &searchParams{
		language: matches[1],
		minStars: minStars,
		perPage:  perPage,
		page:     page,
	}

}

func ghquery(p searchParams) string {
	return fmt.Sprintf("language:%s stars:>=%d", p.language, p.minStars)
}

func getLatestParams(outDir string, language string) (*searchParams, error) {
	files, err := ioutil.ReadDir(outDir)
	if err != nil {
		return nil, err
	}

	latestParams := &searchParams{language: language, minStars: 20, page: 1, perPage: 100}
	for _, file := range files {
		params := paramsFromFilename(file.Name())
		if params == nil {
			continue
		}
		if params.language != language {
			continue
		}
		if params.perPage != latestParams.perPage {
			continue
		}
		if params.minStars > latestParams.minStars {
			latestParams = params
			continue
		}
		if params.minStars == latestParams.minStars && params.page > latestParams.page {
			latestParams = params
			continue
		}
	}
	return latestParams, nil
}

func Main() {
	const perPage = 100
	outDir := "api_response_dump"
	languages := []string{"javascript", "java", "python", "php", "ruby", "c#", "c++", "c", "shell", "objective-c", "go", "swift", "scala", "rust", "kotlin"}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_ACCESS_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	for _, language := range languages {
		searchCh := make(chan searchParams, 100)

		startParams, err := getLatestParams(outDir, language)
		if err != nil {
			log.Printf("Error getting latest params for language %s: %s", language, err)
			continue
		}
		log.Printf("Start params: %#v", *startParams)
		searchCh <- *startParams

		for params := range searchCh {
			func() {
				params := params
				lastStars := 0
				retry := false
				defer func() {
					newParams := params
					switch {
					case retry:
						searchCh <- params
					case params.page < 10:
						newParams.page++
						searchCh <- newParams
					case params.minStars < 10000:
						newParams.minStars++
						newParams.page = 1
						if lastStars-1 > newParams.minStars {
							newParams.minStars = lastStars - 1
						}
						searchCh <- newParams
					default:
						close(searchCh)
					}
				}()

				log.Printf("params: %v", params)
				fp := filepath.Join(outDir, filename(params))

				res, _, err := client.Search.Repositories(ctx, ghquery(params), &github.SearchOptions{
					Sort:  "stars",
					Order: "asc",
					ListOptions: github.ListOptions{
						Page:    params.page,
						PerPage: params.perPage,
					},
				})
				if err != nil {
					log.Printf("GitHub API error: %s", err)
					if strings.Contains(err.Error(), "rate limit") {
						log.Printf("Sleeping for 30 seconds before retrying")
						retry = true
						time.Sleep(30 * time.Second)
					}
					return
				}
				outfile, err := os.Create(fp)
				if err != nil {
					log.Printf("Error creating outfile: %s", err)
					return
				}
				defer outfile.Close()
				if err := json.NewEncoder(outfile).Encode(res); err != nil {
					log.Printf("Error encoding JSON: %s", err)
					return
				}
			}()
		}
	}
}

// ################################
// GraphQL
// ################################

// func main() {
// 	// const perPage = 100
// 	// languages := []string{"javascript"}
// 	// for _, language := range languages {
// 	// }

// 	src := oauth2.StaticTokenSource(
// 		&oauth2.Token{AccessToken: secretToken},
// 	)
// 	httpClient := oauth2.NewClient(context.Background(), src)
// 	client := githubv4.NewClient(httpClient)

// 	cursor := (*githubv4.String)(nil)
// 	for i := 0; i < 11; i++ {
// 		var query struct {
// 			Search struct {
// 				RepositoryCount githubv4.Int
// 				Edges           []struct {
// 					Cursor githubv4.String
// 					Node   struct {
// 						Repository struct {
// 							NameWithOwner githubv4.String
// 							Stargazers    struct {
// 								TotalCount githubv4.Int
// 							}
// 						} `graphql:"... on Repository"`
// 					}
// 				}
// 			} `graphql:"search(query: $query, type: $type, first: $first, after: $after)"`
// 		}
// 		variables := map[string]interface{}{
// 			"query": githubv4.String("test"),
// 			"type":  githubv4.SearchTypeRepository,
// 			"first": githubv4.Int(100),
// 			"after": cursor,
// 		}
// 		err := client.Query(context.Background(), &query, variables)
// 		if err != nil {
// 			log.Fatalf("Query error: %v", err)
// 		}

// 		// Print results
// 		if cursor != nil {
// 			log.Printf("# cursor: %v", *cursor)
// 		}
// 		// for _, res := range query.Search.Edges {
// 		// 	fmt.Printf("%s: %d\n", res.Node.Repository.NameWithOwner, res.Node.Repository.Stargazers.TotalCount)
// 		// }
// 		if len(query.Search.Edges) == 0 {
// 			break
// 		}
// 		nextCursor := query.Search.Edges[len(query.Search.Edges)-1].Cursor
// 		cursor = &nextCursor
// 	}
// }
