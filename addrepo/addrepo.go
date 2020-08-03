package addrepo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/google/go-github/github"
)

func Main(filterText string) error {
	const perPage = 100
	inDir := "api_response_dump"
	outDir := "added"

	allFiles, err := ioutil.ReadDir(inDir)
	if err != nil {
		return err
	}
	var files []os.FileInfo
	if filterText == "" {
		files = allFiles
	} else {
		for _, file := range allFiles {
			if strings.Contains(file.Name(), filterText) {
				files = append(files, file)
			}
		}
	}
	sort.Sort(FileSorter(files))
	parallelism := 3
	fileCh := make(chan string)
	var wg sync.WaitGroup
	wg.Add(parallelism)
	for i := 0; i < parallelism; i++ {
		go func(gid int, fileCh <-chan string) {
			for filename := range fileCh {
				outFile := filepath.Join(outDir, filename)
				if _, err := os.Stat(outFile); err == nil {
					log.Printf("[Goroutine %d] Skipping, file %s already exists", gid, outFile)
					continue
				} else if !os.IsNotExist(err) {
					log.Printf("[Goroutine %d] Error stat'ing file %s: %s", gid, outFile, err)
					continue
				}

				f, err := os.Open(filepath.Join(inDir, filename))
				if err != nil {
					log.Printf("[Goroutine %d] Error reading file %s: %s", gid, filename, err)
					continue
				}
				defer f.Close()
				var result github.RepositoriesSearchResult
				if err := json.NewDecoder(f).Decode(&result); err != nil {
					log.Printf("[Goroutine %d] Error parsing file %s JSON: %s", gid, filename, err)
					continue
				}

				var repoErrs []string
				repoNames := toRepoNames(result.Repositories)
				traunches := toTraunches(repoNames, 10)
				log.Printf("[Goroutine %d] Processing file %s, errors: %d", gid, filename, len(repoErrs))
				for i, repos := range traunches {
					log.Printf("[Goroutine %d] File %s\ttraunch %d", gid, filename, i)
					if err := bulkEnsureRepos(repos); err != nil {
						repoErr := fmt.Sprintf("[Goroutine %d] Failed to ensure repo traunch %v for file %s: %s", gid, repos, filename, err)
						log.Print(repoErr)
						repoErrs = append(repoErrs, repoErr)
					}
				}

				if err := ioutil.WriteFile(outFile, []byte(strings.Join(repoErrs, "\n")), 0644); err != nil {
					log.Printf("[Goroutine %d] Error writing verification file %s: %s", gid, outFile, err)
				}

			}
			wg.Done()
		}(i, fileCh)
	}
	for _, file := range files {
		fileCh <- file.Name()
	}
	close(fileCh)
	wg.Wait()

	return nil
}

func toRepoNames(repos []*github.Repository) []string {
	repoNames := make([]string, 0, len(repos))
	for _, repo := range repos {
		if repo.FullName != nil {
			repoNames = append(repoNames, *repo.FullName)
		}
	}
	return repoNames
}

func toTraunches(arr []string, traunchSize int) (traunches [][]string) {
	if len(arr) == 0 {
		return nil
	}
	traunches = make([][]string, 0, len(arr)/traunchSize+1)
	for i := 0; i < len(arr); i += traunchSize {
		if i+traunchSize > len(arr) {
			traunches = append(traunches, arr[i:])
		} else {
			traunches = append(traunches, arr[i:i+traunchSize])
		}
	}
	return traunches
}

func bulkEnsureRepos(repos []string) error {
	gqlParts := make([]string, len(repos))
	for i := 0; i < len(repos); i++ {
		gqlParts[i] = fmt.Sprintf(`r%d:repository(name: %q) {
	  id
	  name
	  mirrorInfo {
	    cloned
	    cloneProgress
	    cloneInProgress
	  }
	  commit(rev: "HEAD") {
	    id
	  }
	}`, i, "github.com/"+repos[i])
	}
	gqlQuery := "query {\n" + strings.Join(gqlParts, "\n") + "\n}"

	reqBody := fmt.Sprintf(`{ "query": %q }`, gqlQuery)
	req, err := http.NewRequest("POST", "https://sourcegraph.com/.api/graphql", strings.NewReader(reqBody))
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if _, err := ioutil.ReadAll(resp.Body); err != nil {
		return err
	}

	sqlParts := make([]string, len(repos))
	for i := 0; i < len(repos); i++ {
		sqlParts[i] = fmt.Sprintf(`'github.com/%s'`, repos[i])
	}
	sqlRepoNames := "(" + strings.Join(sqlParts, ", ") + ")"
	sqlQuery := `insert into default_repos(repo_id) select id from repo where name in ` + sqlRepoNames + ` and not exists (select * from default_repos where default_repos.repo_id=repo.id)`
	bashCmd := fmt.Sprintf(`kubectl -nprod exec $(kubectl -nprod get pod -l app=pgsql -o jsonpath="{.items[0].metadata.name}") -- psql -t -U sg -c %q`, sqlQuery)
	cmd := exec.Command("bash", "-c", bashCmd)
	out, err := cmd.Output()
	if err != nil {
		return err
	}
	log.Printf("Inserted into default_repos: %v", repos)
	log.Printf("kubectl out: %s", string(out))
	return nil
}

type FileSorter []os.FileInfo

func (s FileSorter) Len() int           { return len(s) }
func (s FileSorter) Less(i, j int) bool { return strings.Compare(s[i].Name(), s[j].Name()) > 0 }
func (s FileSorter) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
