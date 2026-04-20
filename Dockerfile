# ── CSS build stage ─────────────────────────────────────────────────
# Node is used only here to run the Tailwind v4 CLI. Output CSS is
# copied into the go builder stage; node_modules never leaves.
FROM node:20-alpine AS css-builder

WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

# Content sources that Tailwind scans for utility classes + input.css.
COPY static ./static
COPY views ./views
COPY graph ./graph
COPY cmd ./cmd
COPY handlers ./handlers

RUN npx @tailwindcss/cli \
    -i ./static/css/input.css \
    -o ./static/css/site.css \
    --minify

# ── Go build stage ──────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

RUN go install github.com/a-h/templ/cmd/templ@latest

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=css-builder /app/static/css/site.css ./static/css/site.css
RUN templ generate
RUN CGO_ENABLED=0 go build -o /site ./cmd/site/

# ── Final stage ─────────────────────────────────────────────────────
FROM alpine:3.21
RUN apk add --no-cache ca-certificates nodejs npm
RUN npm install -g @anthropic-ai/claude-code
COPY --from=builder /site /site
COPY --from=builder /app/static /static

EXPOSE 8080
CMD ["/site"]
