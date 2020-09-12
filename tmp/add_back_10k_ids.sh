# cat 10k_ids | awk '{ print "INSERT INTO default_repos(repo_id) VALUES (" $1 ");" }' | kubectl -n prod exec -i pgsql-767c45c76d-c9278 -- psql -U sg
# cat 10k_ids_pg2 | awk '{ print "INSERT INTO default_repos(repo_id) VALUES (" $1 ");" }' | kubectl -n prod exec -i pgsql-767c45c76d-c9278 -- psql -U sg
# cat 10k_ids_20k_to_50k | awk '{ print "INSERT INTO default_repos(repo_id) VALUES (" $1 ");" }' | kubectl -n prod exec -i pgsql-767c45c76d-c9278 -- psql -U sg
cat 50k_to_100k_ids | awk '{ print "INSERT INTO default_repos(repo_id) VALUES (" $1 ");" }' | kubectl -n prod exec -i pgsql-767c45c76d-c9278 -- psql -U sg
