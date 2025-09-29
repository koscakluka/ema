GODOTENV = github.com/joho/godotenv/cmd/godotenv@latest
ENV_FILE = .env


.PHONY: run
run:
	go build -o bin/main .
	go run $(GODOTENV) -f $(ENV_FILE) ./bin/main

.PHONY: bump-patch
bump-patch:
	(git describe --tags --abbrev=0 --match 'v*') | awk -F. '{ $$3 ++; print $$1 "." $$2 "." $$3}' | xargs -I {} sh -c 'git tag "{}" && echo "Tagged {}"'

.PHONY: bump-minor
bump-minor:
	(git describe --tags --abbrev=0 --match 'v*') | awk -F. '{ $$2 ++; $$3 = 0; print $$1 "." $$2 "." $$3 }' | xargs -I {} sh -c 'git tag "{}" && echo "Tagged {}"'

.PHONY: bump-major
bump-major:
	(git describe --tags --abbrev=0 --match 'v*') | sed s/v// | awk -F. '{ $$1 ++; $$2 = 0; $$3 = 0; print "v" $$1 "." $$2 "." $$3 }' | xargs -I {} sh -c 'git tag "{}" && echo "Tagged {}"'
