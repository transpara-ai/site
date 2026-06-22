.PHONY: css generate-personas generate build test vet verify verify-canonical-paths run dev deploy

LEGACY_OPERATION_REPO := transpara-ai/civilization-operation

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

verify-canonical-paths:
	test -f graph/review_console.go
	grep -Fq 'https://github.com/transpara-ai/operation/issues/26' graph/review_console.go
	grep -Fq 'SourceRepo:     "transpara-ai/operation"' graph/review_console.go
	@# Keep scanning narrow so historical docs and submodule history remain preserved.
	@output=$$(grep -RnF "$(LEGACY_OPERATION_REPO)" graph/review_console.go); status=$$?; \
	if [ $$status -eq 0 ]; then \
		printf '%s\n' "$$output"; \
		echo "canonical-repo violation: use transpara-ai/operation"; \
		exit 1; \
	fi; \
	if [ $$status -ne 1 ]; then \
		echo "canonical-repo scan failed"; \
		exit $$status; \
	fi

verify: verify-canonical-paths build test vet

run: build
	./site

dev:
	npx @tailwindcss/cli -i ./static/css/input.css -o ./static/css/site.css --watch &
	templ generate --watch &
	go run ./cmd/site/

# on-prem deploy — private to Transpara-AI; build + restart the systemd user service (see deploy.sh)
deploy:
	./deploy.sh
