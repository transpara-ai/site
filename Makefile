.PHONY: css generate build run dev deploy

css:
	npx @tailwindcss/cli -i ./static/css/input.css -o ./static/css/site.css --minify

generate:
	templ generate

build: css generate
	go build -o site ./cmd/site/

run: build
	./site

dev:
	npx @tailwindcss/cli -i ./static/css/input.css -o ./static/css/site.css --watch &
	templ generate --watch &
	go run ./cmd/site/

deploy:
	fly deploy
