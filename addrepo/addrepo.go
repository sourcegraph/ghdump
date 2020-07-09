package addrepo

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/google/go-github/github"
)

func Main() error {
	const perPage = 100
	inDir := "api_response_dump"
	outDir := "added"
	// languages := []string{"javascript", "java", "python", "php", "ruby", "c#", "c++", "c", "shell", "objective-c", "go", "swift", "scala", "rust", "kotlin"}

	files, err := ioutil.ReadDir(inDir)
	if err != nil {
		return err
	}
	for _, file := range files {
		// TODO: check for outfile

		f, err := os.Open(filepath.Join(inDir, file.Name()))
		if err != nil {
			log.Printf("Error reading file %s: %s", file, err)
			continue
		}
		defer f.Close()
		var result github.RepositoriesSearchResult
		if err := json.NewDecoder(f).Decode(&result); err != nil {
			log.Printf("Error parsing file %s JSON: %s", file, err)
			continue
		}
		for _, repo := range result.Repositories {
			if repo.FullName == nil {
				log.Printf("Repo missing FullName: %#v", repo)
				continue
			}
			log.Printf("Full name: %s", *repo.FullName)
		}
		outFile := filepath.Join(outDir, file.Name())
		if err := ioutil.WriteFile(outFile, []byte(""), 0644); err != nil {
			log.Printf("Error writing verification file %s: %s", outFile, err)
		}
	}
	return nil
}

// func ensureRepoExists(repo string) error {

// 	graphqlAddQuery := `query {
//   repository(name: "github.com/beyang/tmp8") {
//     id
//     name
//     mirrorInfo {
//       cloned
//       cloneProgress
//       cloneInProgress
//     }
//     commit(rev: "HEAD") {
//       id
//     }
//   }
// }`

// }

// func addToIndex(repoid int) error {

// }
