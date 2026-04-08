// Command site serves lovyou.ai — the home of the hive's products.
package main

import (
	"bytes"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
	"github.com/yuin/goldmark"

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
	codeGraph := content.LoadCodeGraph()
	higherOrderOps := content.LoadHigherOrderOps()
	log.Printf("loaded %d grammars + base grammar + cognitive grammar + code graph + higher-order ops", len(grammars))

	// Blog handlers.
	handleHome, handleBlogIndex, handleBlogPost := makeHandlers(posts)

	mux := http.NewServeMux()

	// Static files.
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Pages (home is registered after auth setup to support redirect).
	mux.HandleFunc("GET /blog", handleBlogIndex)
	mux.HandleFunc("GET /blog/{slug}", handleBlogPost)

	// Forward-declare DB-dependent vars so closures can capture them.
	// They're set later in the DB init block.
	var graphStore *graph.Store
	_ = graphStore // used in vision handler closures

	// Vision.
	mux.HandleFunc("GET /vision", func(w http.ResponseWriter, r *http.Request) {
		views.VisionPage(layers).Render(r.Context(), w)
	})
	var hiveSpaceID string // resolved lazily on first vision layer request

	mux.HandleFunc("GET /vision/layer/{num}", func(w http.ResponseWriter, r *http.Request) {
		// Lazy-resolve hive space ID.
		if hiveSpaceID == "" && graphStore != nil {
			if sp, err := graphStore.GetSpaceBySlug(r.Context(), "hive"); err == nil {
				hiveSpaceID = sp.ID
			}
		}
		num, err := strconv.Atoi(r.PathValue("num"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		layer, ok := layersByNum[num]
		if !ok {
			http.NotFound(w, r)
			return
		}

		// Load goals from DB: find the layer goal node, then its children.
		var goals []views.Goal
		if graphStore != nil {
			layerTitle := fmt.Sprintf("Layer %d:", num)
			// Find the layer's goal node by title prefix.
			allGoals, _ := graphStore.ListNodes(r.Context(), graph.ListNodesParams{
				SpaceID: hiveSpaceID,
				Kind:    "goal",
				ParentID: "root",
			})
			for _, lg := range allGoals {
				if len(lg.Title) >= len(layerTitle) && lg.Title[:len(layerTitle)] == layerTitle && lg.State != "done" && lg.State != "closed" {
					// Found the layer node — get its children.
					children, _ := graphStore.ListNodes(r.Context(), graph.ListNodesParams{
						SpaceID:  hiveSpaceID,
						ParentID: lg.ID,
					})
					for _, child := range children {
						if child.State != "done" && child.State != "closed" {
							goals = append(goals, views.Goal{
								ID:    child.ID,
								Title: child.Title,
								Body:  child.Body,
								State: child.State,
							})
						}
					}
					break
				}
			}
		}

		views.VisionLayerPage(layer, goals, layers).Render(r.Context(), w)
	})

	mux.HandleFunc("GET /vision/goal/{id}", func(w http.ResponseWriter, r *http.Request) {
		if hiveSpaceID == "" && graphStore != nil {
			if sp, err := graphStore.GetSpaceBySlug(r.Context(), "hive"); err == nil {
				hiveSpaceID = sp.ID
			}
		}
		id := r.PathValue("id")
		if graphStore == nil {
			http.NotFound(w, r)
			return
		}

		goal, err := graphStore.GetNode(r.Context(), id)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		// Build breadcrumb: walk up parent chain.
		var breadcrumbs []views.VisionBreadcrumb
		parentID := goal.ParentID
		for parentID != "" {
			parent, err := graphStore.GetNode(r.Context(), parentID)
			if err != nil {
				break
			}
			breadcrumbs = append([]views.VisionBreadcrumb{{
				ID: parent.ID, Title: parent.Title, Kind: parent.Kind,
			}}, breadcrumbs...)
			parentID = parent.ParentID
		}

		// Get children of this goal.
		children, _ := graphStore.ListNodes(r.Context(), graph.ListNodesParams{
			SpaceID:  hiveSpaceID,
			ParentID: id,
		})
		var childNodes []views.VisionChild
		for _, c := range children {
			if c.State == "done" || c.State == "closed" {
				continue
			}
			// Count grandchildren for progress.
			grandchildren, _ := graphStore.ListNodes(r.Context(), graph.ListNodesParams{
				SpaceID:  hiveSpaceID,
				ParentID: c.ID,
			})
			total, done := 0, 0
			for _, gc := range grandchildren {
				total++
				if gc.State == "done" || gc.State == "closed" {
					done++
				}
			}
			childNodes = append(childNodes, views.VisionChild{
				ID: c.ID, Title: c.Title, Body: c.Body,
				Kind: c.Kind, State: c.State,
				ChildCount: total, DoneCount: done,
			})
		}

		// Find nodes that reference this goal in their causes (multi-parent).
		alsoServes, _ := graphStore.ListNodes(r.Context(), graph.ListNodesParams{
			SpaceID:  hiveSpaceID,
			CausedBy: id,
		})
		var crossRefs []views.VisionBreadcrumb
		for _, ref := range alsoServes {
			if ref.State != "done" && ref.State != "closed" && ref.ID != goal.ID {
				crossRefs = append(crossRefs, views.VisionBreadcrumb{
					ID: ref.ID, Title: ref.Title, Kind: ref.Kind,
				})
			}
		}

		views.VisionGoalPage(
			views.VisionBreadcrumb{ID: goal.ID, Title: goal.Title, Kind: goal.Kind},
			goal.Body, goal.State,
			breadcrumbs, childNodes, crossRefs,
		).Render(r.Context(), w)
	})

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
	mux.HandleFunc("GET /reference/higher-order-ops", func(w http.ResponseWriter, r *http.Request) {
		views.HigherOrderOpsPage(higherOrderOps).Render(r.Context(), w)
	})
	mux.HandleFunc("GET /reference/code-graph", func(w http.ResponseWriter, r *http.Request) {
		views.CodeGraphPage(codeGraph).Render(r.Context(), w)
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
	var mind *graph.Mind

	// Auth middleware wrappers — initialized in the DB block, used by routes below.
	noop := func(hf http.HandlerFunc) http.Handler { return hf }
	readWrap, writeWrap := noop, noop

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
		demoSlug := graphStore.SeedDemoSpace(context.Background())
		graphStore.EnsureAgentsSpace(context.Background())
		graphStore.SeedAgentPersonas(context.Background())
		graphHandlers := graph.NewHandlers(graphStore, readWrap, writeWrap)
		graphHandlers.Register(mux)
		log.Println("app enabled (DATABASE_URL set)")

		// Wire pub/sub webhook for hive event notifications.
		if webhookURL := os.Getenv("HIVE_WEBHOOK_URL"); webhookURL != "" {
			graphStore.OnOp(graph.WebhookSubscriber(webhookURL))
			log.Printf("hive webhook enabled: %s", webhookURL)
		}

		// Wire Mind auto-reply if Claude token is set.
		if claudeToken := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN"); claudeToken != "" {
			mind = graph.NewMind(db, graphStore, claudeToken)
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
			pubSpaces, _ := graphStore.ListPublicSpaces(r.Context())
			var featured []views.FeaturedSpace
			for i, sp := range pubSpaces {
				if i >= 4 {
					break
				}
				featured = append(featured, views.FeaturedSpace{
					Slug: sp.Slug, Name: sp.Name, Description: sp.Description,
					Kind: sp.Kind, NodeCount: sp.NodeCount, HasAgent: sp.HasAgent,
				})
			}
			views.Home(views.HomeStats{
				Spaces: ps.Spaces, Tasks: ps.Tasks,
				Users: ps.Users, AgentOps: ps.AgentOps,
				FeaturedSpaces: featured,
				DemoSlug:       demoSlug,
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

	// Search page — unified search across spaces, content, and users.
	mux.HandleFunc("GET /search", func(w http.ResponseWriter, r *http.Request) {
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		result := views.SearchResult{Query: q}
		if graphStore != nil && q != "" {
			sr := graphStore.Search(r.Context(), q, 10)
			for _, sp := range sr.Spaces {
				result.Spaces = append(result.Spaces, views.SearchSpace{
					Slug: sp.Slug, Name: sp.Name, Description: sp.Description, Kind: sp.Kind,
				})
			}
			for _, n := range sr.Nodes {
				result.Nodes = append(result.Nodes, views.SearchNode{
					ID: n.ID, Title: n.Title, Body: n.Body, Kind: n.Kind,
					State: n.State, SpaceSlug: n.SpaceSlug, SpaceName: n.SpaceName,
				})
			}
			for _, u := range sr.Users {
				result.Users = append(result.Users, views.SearchUser{Name: u.Name, Kind: u.Kind})
			}
		}
		views.SearchPage(result).Render(r.Context(), w)
	})

	// Command palette search (returns compact HTML fragment).
	mux.HandleFunc("GET /api/palette", func(w http.ResponseWriter, r *http.Request) {
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if graphStore == nil || q == "" {
			w.Write([]byte(`<div class="p-4 text-center text-warm-faint text-sm">Type to search...</div>`))
			return
		}
		sr := graphStore.Search(r.Context(), q, 6)
		var buf strings.Builder
		if len(sr.Spaces) > 0 {
			buf.WriteString(`<div class="px-3 py-1.5 text-[10px] font-semibold text-warm-faint uppercase tracking-wider">Spaces</div>`)
			for _, sp := range sr.Spaces {
				buf.WriteString(fmt.Sprintf(`<a href="/app/%s" class="flex items-center gap-2 px-3 py-2 hover:bg-elevated transition-colors rounded-md mx-1 text-sm text-warm"><span class="w-1.5 h-1.5 rounded-full bg-brand flex-shrink-0"></span>%s</a>`, sp.Slug, sp.Name))
			}
		}
		if len(sr.Nodes) > 0 {
			buf.WriteString(`<div class="px-3 py-1.5 text-[10px] font-semibold text-warm-faint uppercase tracking-wider">Items</div>`)
			for _, n := range sr.Nodes {
				buf.WriteString(fmt.Sprintf(`<a href="/app/%s/node/%s" class="flex items-center gap-2 px-3 py-2 hover:bg-elevated transition-colors rounded-md mx-1 text-sm"><span class="text-[10px] font-medium px-1.5 py-0.5 rounded-full bg-warm-faint/20 text-warm-muted">%s</span><span class="text-warm truncate">%s</span><span class="text-[10px] text-warm-faint ml-auto flex-shrink-0">%s</span></a>`, n.SpaceSlug, n.ID, n.Kind, n.Title, n.SpaceName))
			}
		}
		if len(sr.Users) > 0 {
			buf.WriteString(`<div class="px-3 py-1.5 text-[10px] font-semibold text-warm-faint uppercase tracking-wider">People</div>`)
			for _, u := range sr.Users {
				badge := ""
				if u.Kind == "agent" {
					badge = ` <span class="text-[9px] px-1 py-0.5 rounded bg-violet-500/10 text-violet-400">agent</span>`
				}
				buf.WriteString(fmt.Sprintf(`<a href="/user/%s" class="flex items-center gap-2 px-3 py-2 hover:bg-elevated transition-colors rounded-md mx-1 text-sm text-warm">%s%s</a>`, u.Name, u.Name, badge))
			}
		}
		if buf.Len() == 0 {
			buf.WriteString(`<div class="p-4 text-center text-warm-faint text-sm">No results</div>`)
		}
		w.Write([]byte(buf.String()))
	})

	// @mention autocomplete — returns matching user names as HTML options.
	mux.HandleFunc("GET /api/members", func(w http.ResponseWriter, r *http.Request) {
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if graphStore == nil || q == "" {
			return
		}
		users, _ := graphStore.SearchUsers(r.Context(), q)
		var buf strings.Builder
		for _, u := range users {
			badge := ""
			if u.Kind == "agent" {
				badge = `<span class="text-[9px] px-1 py-0.5 rounded bg-violet-500/10 text-violet-400 ml-1">agent</span>`
			}
			buf.WriteString(fmt.Sprintf(`<div class="px-3 py-1.5 hover:bg-elevated cursor-pointer text-sm text-warm transition-colors" onmousedown="insertMention('%s')">@%s%s</div>`, u.Name, u.Name, badge))
		}
		w.Write([]byte(buf.String()))
	})

	// Discover page — list public spaces (no auth required).
	mux.HandleFunc("GET /discover", func(w http.ResponseWriter, r *http.Request) {
		if graphStore == nil {
			views.DiscoverPage(nil, "", "").Render(r.Context(), w)
			return
		}
		query := r.URL.Query().Get("q")
		kindFilter := r.URL.Query().Get("kind")
		spaces, err := graphStore.ListPublicSpaces(r.Context(), query)
		if err != nil {
			log.Printf("discover: %v", err)
			views.DiscoverPage(nil, "", "").Render(r.Context(), w)
			return
		}
		var ds []views.DiscoverSpace
		for _, sp := range spaces {
			if kindFilter != "" && sp.Kind != kindFilter {
				continue
			}
			ds = append(ds, views.DiscoverSpace{
				Slug:         sp.Slug,
				Name:         sp.Name,
				Description:  sp.Description,
				Kind:         sp.Kind,
				CreatedAt:    sp.CreatedAt,
				NodeCount:    sp.NodeCount,
				LastActivity: sp.LastActivity,
				MemberCount:  sp.MemberCount,
				HasAgent:     sp.HasAgent,
			})
		}
		views.DiscoverPage(ds, query, kindFilter).Render(r.Context(), w)
	})

	// Agents page — public directory of active agent personas.
	mux.HandleFunc("GET /agents", func(w http.ResponseWriter, r *http.Request) {
		if graphStore == nil {
			views.AgentsPage(nil).Render(r.Context(), w)
			return
		}
		personas, err := graphStore.ListAgentPersonas(r.Context())
		if err != nil {
			log.Printf("agents: %v", err)
			views.AgentsPage(nil).Render(r.Context(), w)
			return
		}
		categoryOrder := []string{"care", "governance", "knowledge", "product", "outward", "resource"}
		categoryLabels := map[string]string{
			"care": "Care", "governance": "Governance", "knowledge": "Knowledge",
			"product": "Product", "outward": "Outward", "resource": "Resource",
		}
		grouped := map[string][]views.AgentPersonaItem{}
		for _, p := range personas {
			grouped[p.Category] = append(grouped[p.Category], views.AgentPersonaItem{
				Name: p.Name, Display: p.Display, Description: p.Description, Category: p.Category, LastSeen: p.LastSeen,
			})
		}
		var categories []views.AgentCategoryGroup
		for _, cat := range categoryOrder {
			if items, ok := grouped[cat]; ok {
				categories = append(categories, views.AgentCategoryGroup{
					Name: cat, Label: categoryLabels[cat], Personas: items,
				})
			}
		}
		views.AgentsPage(categories).Render(r.Context(), w)
	})

	// Agent profile page — public view of a single agent persona.
	mux.HandleFunc("GET /agents/{name}", func(w http.ResponseWriter, r *http.Request) {
		personaName := r.PathValue("name")
		if graphStore == nil {
			http.NotFound(w, r)
			return
		}
		persona := graphStore.GetAgentPersona(r.Context(), personaName)
		if persona == nil {
			http.NotFound(w, r)
			return
		}
		var buf bytes.Buffer
		agentMD := goldmark.New()
		if err := agentMD.Convert([]byte(persona.Prompt), &buf); err != nil {
			buf.WriteString("<p>" + persona.Description + "</p>")
		}
		views.AgentProfilePage(views.AgentProfileData{
			Name:        persona.Name,
			Display:     persona.Display,
			Description: persona.Description,
			Category:    persona.Category,
			PromptHTML:  buf.String(),
		}).Render(r.Context(), w)
	})

	// Chat with agent — creates a conversation in the agents space with the persona's role tag.
	mux.Handle("POST /agents/{name}/chat", writeWrap(func(w http.ResponseWriter, r *http.Request) {
		personaName := r.PathValue("name")
		if graphStore == nil {
			http.Error(w, "not available", http.StatusServiceUnavailable)
			return
		}
		ctx := r.Context()
		persona := graphStore.GetAgentPersona(ctx, personaName)
		if persona == nil {
			http.NotFound(w, r)
			return
		}
		agentsSpace, err := graphStore.GetSpaceBySlug(ctx, graph.AgentsSpaceSlug)
		if err != nil || agentsSpace == nil {
			http.Error(w, "agents space not available", http.StatusServiceUnavailable)
			return
		}
		user := auth.UserFromContext(ctx)
		actorID, actor, actorKind := "anonymous", "Anonymous", "human"
		if user != nil {
			actorID, actor, actorKind = user.ID, user.Name, "human"
		}
		node, err := graphStore.CreateNode(ctx, graph.CreateNodeParams{
			SpaceID:    agentsSpace.ID,
			Kind:       graph.KindConversation,
			Title:      "Chat with " + persona.Display,
			Author:     actor,
			AuthorID:   actorID,
			AuthorKind: actorKind,
			Tags:       []string{actorID, "role:" + persona.Name},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		graphStore.RecordOp(ctx, agentsSpace.ID, node.ID, actor, actorID, "converse", nil)
		if mind != nil {
			go mind.OnMessage(agentsSpace.ID, agentsSpace.Slug, node, actorID)
		}
		http.Redirect(w, r, fmt.Sprintf("/app/%s/conversation/%s", agentsSpace.Slug, node.ID), http.StatusSeeOther)
	}))

	// User profiles — identity from action history (Layer 8).
	mux.Handle("GET /user/{name}", readWrap(func(w http.ResponseWriter, r *http.Request) {
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
				NodeID: o.NodeID, NodeTitle: o.NodeTitle,
				SpaceName: spaceName, SpaceSlug: spaceSlug, CreatedAt: o.CreatedAt,
			})
		}
		// Endorsement data.
		endorsements := graphStore.CountEndorsements(r.Context(), u.ID)
		endorsers, _ := graphStore.ListEndorsers(r.Context(), u.ID, 10)
		viewer := auth.UserFromContext(r.Context())
		viewerLoggedIn := viewer != nil && viewer.ID != "anonymous"
		hasEndorsed := false
		isFollowing := false
		if viewerLoggedIn {
			hasEndorsed = graphStore.HasEndorsed(r.Context(), viewer.ID, u.ID)
			isFollowing = graphStore.IsFollowing(r.Context(), viewer.ID, u.ID)
		}
		// Follow counts.
		followers := graphStore.CountFollowers(r.Context(), u.ID)
		following := graphStore.CountFollowing(r.Context(), u.ID)
		// Completed work history.
		completed, _ := graphStore.ListCompletedByUser(r.Context(), u.ID, 10)
		var completedWork []views.CompletedWork
		for _, c := range completed {
			completedWork = append(completedWork, views.CompletedWork{
				ID: c.ID, Title: c.Title, SpaceSlug: c.SpaceSlug,
				SpaceName: c.SpaceName, DoneAt: c.DoneAt.Format("Jan 2"),
			})
		}
		// Space memberships.
		memberships, _ := graphStore.ListUserMemberships(r.Context(), u.ID)
		var spaces []views.SpaceMembership
		for _, m := range memberships {
			spaces = append(spaces, views.SpaceMembership{Slug: m.SpaceSlug, Name: m.SpaceName, Kind: m.SpaceKind})
		}
		// Reputation components for "X tasks completed, Y approved" display.
		tasksCompleted, reviewApprovals := graphStore.GetReputationComponents(r.Context(), u.ID)
		views.ProfilePage(views.UserProfile{
			Name: u.Name, Kind: u.Kind,
			TasksDone: u.TasksDone, OpCount: u.OpCount,
			ReputationScore: u.ReputationScore,
			TasksCompleted:  tasksCompleted,
			ReviewApprovals: reviewApprovals,
			Endorsements: endorsements, Endorsers: endorsers,
			HasEndorsed: hasEndorsed, ViewerLoggedIn: viewerLoggedIn,
			Followers: followers, Following: following,
			IsFollowing: isFollowing,
			CompletedWork: completedWork,
			RecentOps: recentOps,
			Spaces: spaces,
		}).Render(r.Context(), w)
	}))

	// Endorse/unendorse a user (Layer 9 — Relationship).
	mux.Handle("POST /user/{name}/endorse", writeWrap(func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if graphStore == nil {
			http.Error(w, "not available", http.StatusServiceUnavailable)
			return
		}
		viewer := auth.UserFromContext(r.Context())
		if viewer == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		targetID := graphStore.ResolveUserID(r.Context(), name)
		if targetID == "" {
			http.NotFound(w, r)
			return
		}
		// Can't endorse yourself.
		if viewer.ID == targetID {
			http.Redirect(w, r, "/user/"+name, http.StatusSeeOther)
			return
		}
		action := r.FormValue("action")
		if action == "unendorse" {
			graphStore.Unendorse(r.Context(), viewer.ID, targetID)
		} else {
			graphStore.Endorse(r.Context(), viewer.ID, targetID)
			graphStore.CreateNotification(r.Context(), targetID, "", "", viewer.Name+": endorsed you")
		}
		// Recompute reputation for the endorsed/unendorsed user.
		graphStore.ComputeAndUpdateReputation(r.Context(), targetID)
		http.Redirect(w, r, "/user/"+name, http.StatusSeeOther)
	}))

	// Follow/unfollow a user (Layer 3 — Social).
	mux.Handle("POST /user/{name}/follow", writeWrap(func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if graphStore == nil {
			http.Error(w, "not available", http.StatusServiceUnavailable)
			return
		}
		viewer := auth.UserFromContext(r.Context())
		if viewer == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		targetID := graphStore.ResolveUserID(r.Context(), name)
		if targetID == "" {
			http.NotFound(w, r)
			return
		}
		// Can't follow yourself.
		if viewer.ID == targetID {
			http.Redirect(w, r, "/user/"+name, http.StatusSeeOther)
			return
		}
		action := r.FormValue("action")
		if action == "unfollow" {
			graphStore.Unfollow(r.Context(), viewer.ID, targetID)
		} else {
			graphStore.Follow(r.Context(), viewer.ID, targetID)
			graphStore.CreateNotification(r.Context(), targetID, "", "", viewer.Name+": started following you")
		}
		http.Redirect(w, r, "/user/"+name, http.StatusSeeOther)
	}))

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
				NodeID: o.NodeID, NodeTitle: o.NodeTitle,
				SpaceName: spaceName, SpaceSlug: spaceSlug, CreatedAt: o.CreatedAt,
			})
		}
		views.GlobalActivityPage(items).Render(r.Context(), w)
	})

	// Market — available tasks across public spaces.
	mux.HandleFunc("GET /market", func(w http.ResponseWriter, r *http.Request) {
		if graphStore == nil {
			views.MarketPage(nil, "").Render(r.Context(), w)
			return
		}
		query := r.URL.Query().Get("q")
		priority := r.URL.Query().Get("priority")
		nodes, err := graphStore.ListAvailableTasks(r.Context(), query, priority, 50)
		if err != nil {
			log.Printf("market: %v", err)
		}
		// Collect unique author IDs for bulk reputation lookup.
		authorIDs := make([]string, 0)
		seen := make(map[string]bool)
		for _, n := range nodes {
			if n.AuthorID != "" && !seen[n.AuthorID] {
				authorIDs = append(authorIDs, n.AuthorID)
				seen[n.AuthorID] = true
			}
		}
		repScores := graphStore.GetBulkReputationByIDs(r.Context(), authorIDs)
		// Resolve space info for links.
		var tasks []views.MarketTask
		for _, n := range nodes {
			slug, spaceName := "", ""
			if sp, _ := graphStore.GetSpaceByID(r.Context(), n.SpaceID); sp != nil {
				slug = sp.Slug
				spaceName = sp.Name
			}
			tasks = append(tasks, views.MarketTask{
				ID: n.ID, SpaceSlug: slug, SpaceName: spaceName, Title: n.Title,
				Body: n.Body, Priority: n.Priority, Author: n.Author,
				AuthorReputation: repScores[n.AuthorID],
			})
		}
		views.MarketPage(tasks, priority).Render(r.Context(), w)
	})

	// Knowledge page — claims across public spaces (Layer 6).
	mux.HandleFunc("GET /knowledge", func(w http.ResponseWriter, r *http.Request) {
		if graphStore == nil {
			views.KnowledgePage(nil, "", "").Render(r.Context(), w)
			return
		}
		stateFilter := r.URL.Query().Get("state")
		query := r.URL.Query().Get("q")
		claims, err := graphStore.ListKnowledgeClaims(r.Context(), stateFilter, query, 100)
		if err != nil {
			log.Printf("knowledge: %v", err)
		}
		var vc []views.KnowledgeClaim
		for _, c := range claims {
			vc = append(vc, views.KnowledgeClaim{
				ID: c.ID, Title: c.Title, Body: c.Body, State: c.State,
				Author: c.Author, AuthorKind: c.AuthorKind,
				SpaceSlug: c.SpaceSlug, SpaceName: c.SpaceName,
				Challenges: c.Challenges, CreatedAt: c.CreatedAt,
			})
		}
		views.KnowledgePage(vc, stateFilter, query).Render(r.Context(), w)
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
		addURL("/agents")
		addURL("/market")
		addURL("/activity")
		addURL("/knowledge")
		addURL("/reference")
		addURL("/reference/grammar")
		addURL("/reference/cognitive-grammar")
		addURL("/reference/grammars")
		addURL("/reference/agents")
		addURL("/reference/higher-order-ops")
		addURL("/reference/code-graph")
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
		// Allow canonical domain, localhost, loopback, and LAN hostnames (no dot = bare hostname like "nucbuntu").
		if host != "" && host != "lovyou.ai" && !strings.HasPrefix(host, "localhost") && !strings.HasPrefix(host, "127.0.0.1") && !strings.HasPrefix(host, "192.168.") && !strings.HasPrefix(host, "10.") && !isBareName(host) {
			target := "https://lovyou.ai" + r.URL.RequestURI()
			http.Redirect(w, r, target, http.StatusMovedPermanently)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isBareName returns true if host is a bare LAN hostname (no dots before any port).
func isBareName(host string) bool {
	h := host
	if i := strings.LastIndex(h, ":"); i != -1 {
		h = h[:i]
	}
	return !strings.Contains(h, ".")
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
