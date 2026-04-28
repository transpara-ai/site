.PHONY: css generate build run dev deploy test verify

TEMPL ?= $(HOME)/go/bin/templ

css:
	npx @tailwindcss/cli -i ./static/css/input.css -o ./static/css/site.css --minify

generate:
	$(TEMPL) generate

build: css generate
	go build -o site ./cmd/site/

run: build
	./site

dev:
	npx @tailwindcss/cli -i ./static/css/input.css -o ./static/css/site.css --watch &
	$(TEMPL) generate --watch &
	go run ./cmd/site/

deploy:
	fly deploy

test:
	go test ./...

verify: build test
