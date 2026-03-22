// Command site serves lovyou.ai — the home of the hive's products.
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	_ "github.com/lib/pq"

	"github.com/lovyou-ai/site/auth"
	"github.com/lovyou-ai/site/content"
	"github.com/lovyou-ai/site/graph"
	"github.com/lovyou-ai/site/views"
)

func main() {
	port := flag.String("port", "", "HTTP port (default: $PORT or 8080)")
	flag.Parse()

	p := *port
	if p == "" {
		p = os.Getenv("PORT")
	}
	if p == "" {
		p = "8080"
	}

	// Load content.
	posts, err := content.LoadPosts()
	if err != nil {
		log.Fatalf("load posts: %v", err)
	}
	log.Printf("loaded %d blog posts", len(posts))

	layers := content.LoadLayers()
	agentPrims := content.LoadAgentPrimitives()

	// Build lookups for individual pages.
	primsBySlug := map[string]views.Primitive{}
	layersByNum := map[int]views.Layer{}
	totalPrims := 0
	for _, layer := range layers {
		layersByNum[layer.Number] = layer
		totalPrims += len(layer.Primitives)
		for _, prim := range layer.Primitives {
			primsBySlug[prim.Slug] = prim
		}
	}
	for _, prim := range agentPrims {
		primsBySlug[prim.Slug] = prim
	}
	log.Printf("loaded %d layers, %d primitives, %d agent primitives",
		len(layers), totalPrims, len(agentPrims))

	grammars, err := content.LoadGrammars()
	if err != nil {
		log.Fatalf("load grammars: %v", err)
	}
	baseGrammar := content.LoadBaseGrammar()
	cognitiveGrammar := content.LoadCognitiveGrammar()
	log.Printf("loaded %d grammars + base grammar + cognitive grammar", len(grammars))

	// Blog handlers.
	handleHome, handleBlogIndex, handleBlogPost := makeHandlers(posts)

	mux := http.NewServeMux()

	// Static files.
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Pages (home is registered after auth setup to support redirect).
	mux.HandleFunc("GET /blog", handleBlogIndex)
	mux.HandleFunc("GET /blog/{slug}", handleBlogPost)

	// Reference.
	mux.HandleFunc("GET /reference", func(w http.ResponseWriter, r *http.Request) {
		views.ReferenceIndex(layers, agentPrims, grammars).Render(r.Context(), w)
	})
	mux.HandleFunc("GET /reference/grammar", func(w http.ResponseWriter, r *http.Request) {
		views.BaseGrammarPage(baseGrammar).Render(r.Context(), w)
	})
	mux.HandleFunc("GET /reference/cognitive-grammar", func(w http.ResponseWriter, r *http.Request) {
		views.CognitiveGrammarPage(cognitiveGrammar).Render(r.Context(), w)
	})
	mux.HandleFunc("GET /reference/layers/{num}", func(w http.ResponseWriter, r *http.Request) {
		num, err := strconv.Atoi(r.PathValue("num"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if layer, ok := layersByNum[num]; ok {
			views.LayerPage(layer, layers).Render(r.Context(), w)
			return
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("GET /reference/agents", func(w http.ResponseWriter, r *http.Request) {
		views.AgentPrimitivesPage(agentPrims).Render(r.Context(), w)
	})
	mux.HandleFunc("GET /reference/primitives/{slug}", func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		if prim, ok := primsBySlug[slug]; ok {
			views.PrimitivePage(prim).Render(r.Context(), w)
			return
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("GET /reference/grammars", func(w http.ResponseWriter, r *http.Request) {
		views.GrammarIndex(grammars).Render(r.Context(), w)
	})
	mux.HandleFunc("GET /reference/grammars/{slug}", func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		for _, g := range grammars {
			if g.Slug == slug {
				views.GrammarPage(g, grammars).Render(r.Context(), w)
				return
			}
		}
		http.NotFound(w, r)
	})

	// Discover page (public spaces) — registered after DB setup below.
	var graphStore *graph.Store

	// Unified product with auth.
	dsn := os.Getenv("DATABASE_URL")
	if dsn != "" {
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			log.Fatalf("open db: %v", err)
		}
		defer db.Close()
		if err := db.Ping(); err != nil {
			log.Fatalf("ping db: %v", err)
		}

		// Auth middleware: Google OAuth if configured, otherwise anonymous passthrough.
		var readWrap, writeWrap func(http.HandlerFunc) http.Handler
		clientID := os.Getenv("GOOGLE_CLIENT_ID")
		clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")

		if clientID != "" && clientSecret != "" {
			redirectURL := os.Getenv("AUTH_REDIRECT_URL")
			if redirectURL == "" {
				redirectURL = "https://lovyou.ai/auth/callback"
			}
			secure := redirectURL[:5] == "https"

			authService, err := auth.New(db, clientID, clientSecret, redirectURL, secure)
			if err != nil {
				log.Fatalf("auth: %v", err)
			}
			authService.Register(mux)

			// API key management page.
			mux.Handle("GET /app/keys", authService.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
				user := auth.UserFromContext(r.Context())
				keys, _ := authService.ListAPIKeys(r.Context(), user.ID)
				var viewKeys []graph.ViewAPIKey
				for _, k := range keys {
					viewKeys = append(viewKeys, graph.ViewAPIKey{
						ID:        k.ID,
						Name:      k.Name,
						AgentName: k.AgentName,
						CreatedAt: k.CreatedAt.Format("Jan 2, 2006"),
					})
				}
				graph.APIKeysView(viewKeys, graph.ViewUser{Name: user.Name, Picture: user.Picture}).Render(r.Context(), w)
			}))

			writeWrap = authService.RequireAuth
			readWrap = authService.OptionalAuth
			log.Println("auth enabled (Google OAuth)")
		} else {
			anonWrap := func(next http.HandlerFunc) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					ctx := auth.ContextWithUser(r.Context(), &auth.User{
						ID: "anonymous", Name: "Anonymous", Email: "anon@lovyou.ai",
					})
					next.ServeHTTP(w, r.WithContext(ctx))
				})
			}
			readWrap = anonWrap
			writeWrap = anonWrap
			log.Println("auth disabled (no GOOGLE_CLIENT_ID) — anonymous mode")
		}

		graphStore, err = graph.NewStore(db)
		if err != nil {
			log.Fatalf("graph store: %v", err)
		}
		graphHandlers := graph.NewHandlers(graphStore, readWrap, writeWrap)
		graphHandlers.Register(mux)
		log.Println("app enabled (DATABASE_URL set)")

		// Wire Mind auto-reply if Claude token is set.
		if claudeToken := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN"); claudeToken != "" {
			mind := graph.NewMind(db, graphStore, claudeToken)
			graphHandlers.SetMind(mind)
			log.Println("mind enabled (CLAUDE_CODE_OAUTH_TOKEN set)")
		} else {
			log.Println("mind disabled (no CLAUDE_CODE_OAUTH_TOKEN)")
		}

		// Home: redirect logged-in users to /app, show landing with live stats for anonymous.
		mux.Handle("GET /{$}", readWrap(func(w http.ResponseWriter, r *http.Request) {
			user := auth.UserFromContext(r.Context())
			if user != nil && user.ID != "anonymous" {
				http.Redirect(w, r, "/app", http.StatusSeeOther)
				return
			}
			ps := graphStore.GetPlatformStats(r.Context())
			views.Home(views.HomeStats{
				Spaces: ps.Spaces, Tasks: ps.Tasks,
				Users: ps.Users, AgentOps: ps.AgentOps,
			}).Render(r.Context(), w)
		}))

		// Redirect old /work to /app.
		mux.HandleFunc("GET /work", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/app", http.StatusMovedPermanently)
		})
	} else {
		log.Println("app disabled (no DATABASE_URL)")
		mux.HandleFunc("GET /{$}", handleHome)
		mux.HandleFunc("GET /app", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "App requires DATABASE_URL", http.StatusServiceUnavailable)
		})
		mux.HandleFunc("GET /work", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/app", http.StatusMovedPermanently)
		})
	}

	// Discover page — list public spaces (no auth required).
	mux.HandleFunc("GET /discover", func(w http.ResponseWriter, r *http.Request) {
		if graphStore == nil {
			views.DiscoverPage(nil).Render(r.Context(), w)
			return
		}
		spaces, err := graphStore.ListPublicSpaces(r.Context())
		if err != nil {
			log.Printf("discover: %v", err)
			views.DiscoverPage(nil).Render(r.Context(), w)
			return
		}
		ds := make([]views.DiscoverSpace, len(spaces))
		for i, sp := range spaces {
			ds[i] = views.DiscoverSpace{
				Slug:         sp.Slug,
				Name:         sp.Name,
				Description:  sp.Description,
				Kind:         sp.Kind,
				CreatedAt:    sp.CreatedAt,
				NodeCount:    sp.NodeCount,
				LastActivity: sp.LastActivity,
				MemberCount:  sp.MemberCount,
				HasAgent:     sp.HasAgent,
			}
		}
		views.DiscoverPage(ds).Render(r.Context(), w)
	})

	// User profiles — identity from action history (Layer 8).
	mux.HandleFunc("GET /user/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if graphStore == nil {
			http.NotFound(w, r)
			return
		}
		u, err := graphStore.GetUserProfile(r.Context(), name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		// Get recent public activity for this user.
		ops, _ := graphStore.ListPublicActivity(r.Context(), 50)
		var recentOps []views.ActivityItem
		for _, o := range ops {
			if o.ActorID != u.ID {
				continue
			}
			spaceName, spaceSlug := "", ""
			if sp, _ := graphStore.GetSpaceByID(r.Context(), o.SpaceID); sp != nil {
				spaceName = sp.Name
				spaceSlug = sp.Slug
			}
			recentOps = append(recentOps, views.ActivityItem{
				Actor: o.Actor, ActorKind: o.ActorKind, Op: o.Op,
				SpaceName: spaceName, SpaceSlug: spaceSlug, CreatedAt: o.CreatedAt,
			})
		}
		views.ProfilePage(views.UserProfile{
			Name: u.Name, Kind: u.Kind,
			TasksDone: u.TasksDone, OpCount: u.OpCount,
			RecentOps: recentOps,
		}).Render(r.Context(), w)
	})

	// Global activity — transparent audit trail (Layer 7).
	mux.HandleFunc("GET /activity", func(w http.ResponseWriter, r *http.Request) {
		if graphStore == nil {
			views.GlobalActivityPage(nil).Render(r.Context(), w)
			return
		}
		ops, err := graphStore.ListPublicActivity(r.Context(), 100)
		if err != nil {
			log.Printf("activity: %v", err)
		}
		var items []views.ActivityItem
		for _, o := range ops {
			spaceName, spaceSlug := "", ""
			if sp, _ := graphStore.GetSpaceByID(r.Context(), o.SpaceID); sp != nil {
				spaceName = sp.Name
				spaceSlug = sp.Slug
			}
			items = append(items, views.ActivityItem{
				Actor: o.Actor, ActorKind: o.ActorKind, Op: o.Op,
				SpaceName: spaceName, SpaceSlug: spaceSlug, CreatedAt: o.CreatedAt,
			})
		}
		views.GlobalActivityPage(items).Render(r.Context(), w)
	})

	// Market — available tasks across public spaces.
	mux.HandleFunc("GET /market", func(w http.ResponseWriter, r *http.Request) {
		if graphStore == nil {
			views.MarketPage(nil).Render(r.Context(), w)
			return
		}
		query := r.URL.Query().Get("q")
		nodes, err := graphStore.ListAvailableTasks(r.Context(), query, 50)
		if err != nil {
			log.Printf("market: %v", err)
		}
		// Resolve space info for links.
		var tasks []views.MarketTask
		for _, n := range nodes {
			slug, name := "", ""
			if sp, _ := graphStore.GetSpaceByID(r.Context(), n.SpaceID); sp != nil {
				slug = sp.Slug
				name = sp.Name
			}
			tasks = append(tasks, views.MarketTask{
				ID: n.ID, SpaceSlug: slug, SpaceName: name, Title: n.Title,
				Body: n.Body, Priority: n.Priority, Author: n.Author,
			})
		}
		views.MarketPage(tasks).Render(r.Context(), w)
	})

	// Health check for Fly.io.
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	// Robots and sitemap.
	mux.HandleFunc("GET /robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "User-agent: *\nAllow: /\nSitemap: https://lovyou.ai/sitemap.xml\n")
	})
	mux.HandleFunc("GET /sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		var b strings.Builder
		b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
		b.WriteString("\n")
		b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
		b.WriteString("\n")
		addURL := func(path string) {
			b.WriteString("<url><loc>https://lovyou.ai")
			b.WriteString(path)
			b.WriteString("</loc></url>\n")
		}
		// Static pages.
		addURL("/")
		addURL("/blog")
		addURL("/discover")
		addURL("/market")
		addURL("/activity")
		addURL("/reference")
		addURL("/reference/grammar")
		addURL("/reference/cognitive-grammar")
		addURL("/reference/grammars")
		addURL("/reference/agents")
		// Blog posts.
		for _, post := range posts {
			addURL("/blog/" + post.Slug)
		}
		// Layers.
		for _, layer := range layers {
			addURL(fmt.Sprintf("/reference/layers/%d", layer.Number))
		}
		// Primitives.
		for _, layer := range layers {
			for _, prim := range layer.Primitives {
				addURL("/reference/primitives/" + prim.Slug)
			}
		}
		for _, prim := range agentPrims {
			addURL("/reference/primitives/" + prim.Slug)
		}
		// Grammars.
		for _, g := range grammars {
			addURL("/reference/grammars/" + g.Slug)
		}
		b.WriteString("</urlset>\n")
		fmt.Fprint(w, b.String())
	})

	addr := ":" + p
	log.Printf("lovyou.ai listening on %s", addr)
	if err := http.ListenAndServe(addr, canonicalHost(noCache(mux))); err != nil {
		log.Fatal(err)
	}
}

// ────────────────────────────────────────────────────────────────────
// Middleware
// ────────────────────────────────────────────────────────────────────

// canonicalHost redirects non-canonical hostnames to lovyou.ai.
// Skips health checks (Fly probes via internal IP) and localhost.
func canonicalHost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		host := r.Host
		if host != "" && host != "lovyou.ai" && !strings.HasPrefix(host, "localhost") && !strings.HasPrefix(host, "127.0.0.1") {
			target := "https://lovyou.ai" + r.URL.RequestURI()
			http.Redirect(w, r, target, http.StatusMovedPermanently)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/static/") {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		}
		next.ServeHTTP(w, r)
	})
}

// ────────────────────────────────────────────────────────────────────
// Handlers
// ────────────────────────────────────────────────────────────────────

func makeHandlers(posts []views.Post) (home, blogIndex, blogPost http.HandlerFunc) {
	home = func(w http.ResponseWriter, r *http.Request) {
		views.Home(views.HomeStats{}).Render(r.Context(), w)
	}
	blogIndex = func(w http.ResponseWriter, r *http.Request) {
		views.BlogIndex(posts).Render(r.Context(), w)
	}
	blogPost = func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		for i, post := range posts {
			if post.Slug == slug {
				var nav views.PostNav
				if i > 0 {
					nav.Prev = &posts[i-1]
				}
				if i < len(posts)-1 {
					nav.Next = &posts[i+1]
				}
				views.BlogPost(post, nav).Render(r.Context(), w)
				return
			}
		}
		http.NotFound(w, r)
	}
	return
}
