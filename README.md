## Usage

Run the script that pulls all repositories of greater than 20 stars of each major language from GitHub API:

```
GITHUB_ACCESS_TOKEN=<token> GO111MODULE=off go run main.go
```

This will dump GitHub search API responses to the `api_response_dump` directory. Each file
corresponds to one API request.

Note: the GitHub search API limits total results for any given query to 1000, so for lower star
counts, we only get the first 1000 repositories with that star count. This is fine for now.

Once you've collected some number of files in `api_response_dump`, run the script to add these
repositories to Sourcegraph (in order of highest star count first), ensure they're queued for
cloning, and added to the global search index:

```
GO111MODULE=off go run main.go add <file_filter_text>
# Example to add all Python repos: GO111MODULE=off go run main.go add python
```

Once the repositories from a given file in `api_response_dump/` have been added, this script will
write a file with the same name to the `added/` directory. If there were errors, these will be
written to the file in `added/`; if there were no errors, that file will be empty.

Note: currently, parallelism is set to 5, so 5 goroutines will be simultaneously reading files and
adding these repositories to Sourcegraph.
