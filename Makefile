GODOTENV = github.com/joho/godotenv/cmd/godotenv@latest
ENV_FILE = .env


.PHONY: run
run:
	go build -o bin/main .
	go run $(GODOTENV) -f $(ENV_FILE) ./bin/main

