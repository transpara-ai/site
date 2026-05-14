.PHONY: css generate-personas generate build test vet verify run dev deploy

css:
	npx @tailwindcss/cli -i ./static/css/input.css -o ./static/css/site.css --minify

generate-personas:
	go generate ./graph/personas/...

generate: generate-personas
	templ generate

build: css generate
	go build -o site ./cmd/site/

test:
	go test -count=1 ./...

vet:
	go vet ./...

verify: build test vet

run: build
	./site

dev:
	npx @tailwindcss/cli -i ./static/css/input.css -o ./static/css/site.css --watch &
	templ generate --watch &
	go run ./cmd/site/

deploy:
	go mod vendor
	fly deploy
