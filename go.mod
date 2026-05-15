module github.com/transpara-ai/site

go 1.25.0

require (
	github.com/a-h/templ v0.3.1001
	github.com/lib/pq v1.12.0
	github.com/transpara-ai/eventgraph/go v0.0.0
)

require (
	github.com/yuin/goldmark v1.7.17
	golang.org/x/oauth2 v0.36.0
)

require (
	cloud.google.com/go/compute/metadata v0.5.0 // indirect
	github.com/anthropics/anthropic-sdk-go v1.26.0 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/transpara-ai/eventgraph/go => ../eventgraph/go
