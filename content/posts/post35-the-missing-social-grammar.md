# The Missing Social Grammar

**Every social interaction ever recorded is a composition of fifteen operations. Here's how we derived them.**

*Matt Searles · Mar 06, 2026*

---

The US Surgeon General called for warning labels on social media in June 2024 — the same kind of warning labels we put on cigarettes. Teens who use more than three hours a day face double the risk of mental health problems. The average American teen spends nearly five hours a day on social media. New York City has classified social networking sites as a public health threat and is suing TikTok, Meta, Snap, and YouTube for "fuelling the nationwide youth mental health crisis."

A study published in *Science* in late 2024 provided some of the clearest causal evidence yet that social media algorithms shape political polarisation — not as a side effect, but as a direct consequence of design choices. The researchers found that algorithmic ranking altered users' political attitudes by amounts comparable to several years' worth of natural polarisation change. In a single scrolling session.

Here's the part that should make you angry: a 2025 study found that Twitter's engagement-based algorithm amplifies content that users *explicitly say makes them feel worse*. Not content they want to see. Not content they find valuable. Content that generates outrage — because outrage generates engagement, and engagement generates ad revenue. The algorithm doesn't show you what you want. It shows you what makes you react. These aren't the same thing, and the platforms know it.

This is not a bug. It is the business model.

Post 33 argued that values should be architectural, not stated. Post 34 proposed governance mechanisms for the architecture. This post asks a more foundational question: what *is* a social interaction, and what happens when you describe it precisely?

---

## The Vocabulary Problem

Every social platform you've used has a vocabulary. Twitter has "tweets" — short, disposable, noisy. Facebook has "posts" — broadcast announcements to your "friends." Instagram has "stories" — ephemeral, performative, gone in 24 hours. TikTok has "videos" — content as performance, attention as currency.

These aren't neutral words. They're design choices that shape behaviour. The Sapir-Whorf hypothesis — that language influences thought — is debated in its strong form but well-established in its weak form: the words available to you affect how you categorise and process experience. A platform that calls everything a "post" teaches you to broadcast. One that calls everything a "story" teaches you to perform. One that calls everything a "tweet" teaches you that communication should be short, loud, and disposable.

The vocabulary teaches the product. And the products are making people sick.

So we started where Post 33 started: with architecture. If you're building a social layer on a causal event graph — hash-chained, append-only, every action signed and auditable — what vocabulary does the architecture *need*? Not want. Not prefer. Need.

We thought we knew. We started with a garden metaphor — roots, branches, seeds, vines. It was pretty. Evocative. And it was incomplete. Not because the metaphor was bad, but because we were naming things before we understood what they were.

So we went deeper.

---

## What Is a Social Interaction?

Strip away the UI. Strip away the metaphor. Strip away the platform. What is a human doing when they interact on a social network?

They're performing operations on a graph.

A graph is vertices (nodes) and edges (connections). When you post something, you're creating a vertex — a content node on the graph. When you reply, you're creating another vertex with a causal edge pointing back to the first. When you like something, you're creating an edge — no new content, just a structural connection. When you share something, you're creating an edge that changes what's reachable from your part of the graph. When you follow someone, you're creating an edge that says "route future content to me." When you send a DM, you're establishing a private channel — a content-bearing edge visible only to its endpoints.

That's it. Every social interaction that has ever occurred on any platform — every tweet, post, story, reply, like, share, follow, message, block, and delete — is a vertex operation, an edge operation, or a traversal.

Three categories. That's the foundation.

But graph theory, the formal mathematics of vertices and edges, doesn't distinguish between a reply and a quote tweet. Both are "new vertex with an edge to existing vertex." It doesn't distinguish between a like and a retweet. Both are "new edge." Graph theory is *content-agnostic* — it doesn't model what an edge means, only that it exists. And it's *time-agnostic* — it doesn't model whether a vertex is permanent or ephemeral.

Social interaction is semantic. It carries meaning, intent, and temporality that pure graph theory can't express. So we need a grammar that extends graph theory into the social domain — that preserves the formal rigour while adding the semantic dimensions that make human interaction human.

We're calling it a semantic graph grammar. Here's how we derived it.

---

## Derivation

Start with the three fundamental graph operations:

1. **Create a vertex** (content enters the graph)
2. **Create an edge** (structure changes)
3. **Traverse** (measure distance, navigate — read-only)

Now ask: what *properties* distinguish one social interaction from another? We found six dimensions:

| Dimension | Values | What it captures |
|-----------|--------|------------------|
| Causality | Independent / Dependent (responsive, divergent, sequential) | Was this caused by an existing vertex? |
| Content | Content-bearing / Structural-only | Does this carry new content or just a connection? |
| Temporality | Persistent / Transient | Does this persist or self-remove? |
| Visibility | Public / Private | Who can observe this? |
| Direction | Centripetal (toward content) / Centrifugal (into actor's subgraph) | Which way does the edge point semantically? |
| Authorship | Same actor / Different actor / Mutual | Continuation, response, or bilateral? |

Graph theory models none of these. It has vertices and edges. We need vertices with *causal histories*, edges with *semantic direction*, and temporal properties on both. This is where graph theory ends and semantic graph grammar begins.

Every distinct combination of operation type and dimensional properties that corresponds to a real social behaviour is a primitive in the grammar. We started with eleven, and then asked a question that changed everything: what social interactions have humans *never been able to express* on a platform?

---

## The First Eleven

### I. Vertex Operations — Content Enters or Leaves the Graph

**Emit** — Create an independent content vertex. New content enters the graph with no causal dependency.

This is the most fundamental creative act on a social graph. A thought enters the world. Graph theory calls it vertex insertion. Every platform has it: the post, the tweet, the video. The difference isn't the operation — it's that on the event graph, the emission is signed, hash-chained, and causally linked to everything that follows from it. Provenance is built in.

**Respond** — Create a causally dependent content vertex, topologically subordinate to its source.

A response grows the source vertex's subtree. The content is *about* the source — it engages with it, extends it, challenges it. Graph theory calls this child node insertion in a directed acyclic graph. But graph theory doesn't capture the semantic relationship: this vertex exists *because* that vertex exists. The causal link is the grammar.

**Derive** — Create a causally dependent content vertex, topologically independent from its source.

This is the operation graph theory can't distinguish from Respond. Structurally, both are "new vertex with causal edge to existing vertex." But the semantics are opposite. A response is subordinate — it lives in the source's subtree, it's part of the source's conversation. A derivation is independent — it starts a new subtree, a new conversation, inspired by but divergent from the source. The quote tweet that goes viral isn't a response. It's a derivation. Same graph operation, different semantic intent, different propagation behaviour. The grammar must distinguish them because the social dynamics are fundamentally different.

**Extend** — Create a causally dependent content vertex, same author, sequential.

A thread. A continuation. The author extending their own prior thought. Structurally it's a path extension — the same actor appending vertices to their own sequence. It's distinct from Respond (different author engaging) and Derive (divergent trajectory). Extend is linear, same-voice, additive.

**Retract** — Tombstone a vertex you own.

On an append-only graph, nothing is truly deleted. Retraction is itself an event — a new vertex that marks a previous vertex as withdrawn. The causal links survive. The fact that something existed and was retracted is part of the graph's history. The content is gone; the provenance is permanent. This is fundamentally different from deletion on every existing platform, where delete means "pretend it never happened." The graph doesn't pretend.

### II. Edge Operations — Structure Changes

**Acknowledge** — Create a content-free directed edge toward an existing vertex.

The lightest possible graph operation. No new content. No new vertex. Just an edge: "I observed this." The like. The heart. The upvote. But stripped of the gamification that every platform layers on top. On the event graph, an acknowledgment is structural — it changes the graph's topology, increases the vertex's connectivity, affects its reachability. What platforms call engagement metrics are actually *graph properties*. Degree centrality. Reachability. Connectivity. The grammar makes this explicit.

**Propagate** — Create a directed edge that redistributes a vertex into the actor's subgraph.

Same structural operation as Acknowledge — edge insertion — but semantically opposite in direction. Acknowledge is centripetal: the edge points *toward* the content. Propagate is centrifugal: the edge pulls the content *into* the actor's subgraph, changing its reachability. "My network should see this." The retweet. The share. The boost. In graph theory terms, this is a graft — joining two subgraphs at a shared vertex. The content vertex doesn't move. Its reachability expands.

This distinction — centripetal vs centrifugal — doesn't exist in graph theory because graph theory doesn't model an actor's subgraph as a semantically meaningful unit. In social graphs, it is. Your subgraph is your audience. Propagation changes whose audience can reach a vertex. Acknowledgment doesn't.

**Subscribe** — Create a persistent directed edge to an actor vertex.

"Route this actor's future emissions to me." This is the operation graph theory has no concept of, because graph theory is state-based — it models the graph as it is, not as it will be. Subscribe is *anticipatory*. It's an edge that says: not this content, but all future content from this source. The follow. The most structurally powerful social operation, because it shapes what enters your subgraph over time.

**Channel** — Establish a private, bidirectional, content-bearing edge between two actors.

A persistent communication pathway. The DM. But on the event graph, a channel is more precisely a private subgraph — a space where two actors can emit, respond, and extend, visible only to each other. Graph theory calls the structural concept a bridge: an edge whose removal disconnects two subgraphs. The semantic addition is privacy (visibility constraint) and bidirectionality (both actors can emit into the channel). The graph records that a channel exists and that communication occurred. It does not record what was said. Structure without content. Accountability without surveillance.

**Sever** — Remove a subscription or channel.

The destructive edge operation. Unfollow. Disconnect. Block. On an append-only graph, severing — like retracting — is itself an event. The graph records that a connection existed and was severed. The severance is part of the causal history. You can't pretend a relationship never happened, but you can end it.

### III. Traversal

**Traverse** — Navigate or measure distance between vertices.

The only operation that doesn't mutate the graph. Shortest path. Degrees of separation. "How far is this from me?" In graph theory: geodesic distance, breadth-first search. On the event graph, traversal is also the discovery mechanism — you explore outward from your own node, one hop at a time, and the graph's topology *is* the recommendation. No algorithm deciding what you see. Just structure.

---

## What's Missing

Eleven operations and the grammar already describes every social interaction on every existing platform. We could have stopped here.

But we asked: what social interactions do humans perform in real life that no platform has ever been able to express? Not "what features should we add?" — that's product thinking. "What human social behaviours have been *structurally inexpressible* on social media?" That's grammar thinking.

Four operations fell out immediately. They'd been invisible because no platform has ever had them — but once you see them, you can't unsee how much social fabric they'd create.

---

## The Missing Four

**Endorse** — "I stake my reputation on this."

Not Acknowledge ("I noticed") or Propagate ("my network should see this"). Endorsement is an edge where the actor accepts *reputational consequence* if the endorsed content proves false or harmful. This introduces a semantic dimension none of the first eleven operations have: **stake**. Every existing operation is cost-free to the actor. Endorsement has skin in the game.

This is the missing operation in every misinformation crisis. On every current platform, Propagate is treated as implicit endorsement, but people share things they disagree with, find funny, or want to mock. The grammar can't distinguish "I believe this" from "look at this idiot" because it has no endorsement primitive. A 2024 MIT study found that false news spreads six times faster than true news on Twitter — not because people endorse it, but because they *react* to it, and the platform treats reaction as endorsement. Adding Endorse as a distinct operation makes the difference legible. You can share without endorsing. You can endorse without sharing. The graph records which you did.

No platform has this. The closest is Community Notes on X, but that's crowd-sourced annotation, not individual endorsement with stake.

**Delegate** — "Act on my behalf."

Granting another actor — your agent, a trusted person, an organisation — permission to perform specific operations as you. This is a *meta-operation*. It modifies who can perform operations, not the graph itself. None of the first eleven address authority over operations.

This is critical for the agent architecture. Your agent Emitting on your behalf. A carer managing a vulnerable person's graph. An organisation delegating to representatives. And it addresses questions that every platform handles badly: what happens when someone dies? Who operates their graph? What about disability — who speaks for someone when they can't? What about institutions — who Emits for a company?

Current platforms have crude versions — Twitter's multi-account access, Facebook page admins. But no formal delegation with auditable authority chains. You either have the password or you don't. On the event graph, Delegate is a signed event: who granted authority, to whom, for which operations, under what constraints, and when it expires. The delegation is on the graph. It's auditable. It can be revoked. And the delegated actor's operations are attributed to both the delegate and the delegator — accountability chains.

**Consent** — "We both agree."

A bilateral event requiring both parties to sign. Every existing operation is *unilateral* — one actor performs it. Consent is the first *mutual* operation. Two actors co-creating a single event on the graph, both signing it, neither able to claim it happened without the other's confirmation.

This is foundational for everything trust-based. Contracts. Agreements. Transactions. Commitments. "We agreed to this" is one of the most basic human social operations, and no social platform can express it. Currently, agreements are modelled as two separate messages that happen to say yes — there's no atomic, cryptographically signed bilateral event. On the event graph, Consent is a single event with two signatures. It either exists with both, or it doesn't exist at all. Atomicity. The handshake as a graph primitive.

This also connects directly to the governance architecture from Post 34. Voting on a constitutional amendment is a form of Consent — the community and the system agreeing to a change. Dual consent (human + agent) is two Consent operations on the same event. The grammar unifies governance and social interaction.

**Annotate** — "This context belongs *on* this, not beside it."

Not Respond (content subordinate to and *about* the source) or Derive (content divergent from the source). Annotate attaches metadata *to* a vertex — a correction, a translation, a content warning, a fact-check, a citation, an accessibility description. The annotation is *on* the vertex, not beside it in the conversation tree. It has no independent existence without its target.

This is a new type of vertex operation — *parasitic*. A response can stand on its own if the source is retracted. An annotation can't. It's structurally dependent on its target in a way that responses aren't.

The social fabric case is strong. Corrections that travel *with* the content they correct, not somewhere in a reply thread nobody reads. Translations attached to the original, not posted separately. Image descriptions added by someone other than the original author, attached to the image. Community context that's visible wherever the vertex appears, not buried in a separate section. This is what Community Notes on X is reaching for — but without a formal annotation primitive, it's bolted on, limited, and centrally controlled. On the event graph, anyone can annotate, and the annotations are visible wherever the vertex is traversed.

---

## One More: Merge

We debated this one. The first fifteen operations can split (Derive creates divergence) but never join. Merge is the inverse — two subtrees converging into one. Two conversations that turn out to be about the same thing. Two actors who independently emitted the same idea and want to combine their discussion graphs.

Merge is structurally novel. It creates a vertex with multiple parent edges from independent subtrees, which is different from Respond (one parent, subordinate) or Derive (one parent, divergent). In graph theory, this makes the graph a DAG rather than a tree — which the event graph already is, but Merge makes it explicit.

The danger: who has authority to merge? Both subtree authors? Any actor? Merge is powerful for collaborative knowledge-building — converging independent discussions into shared understanding. But it could also be used to hijack conversations by forcibly merging them with hostile content. This operation needs governance more than any other — perhaps requiring Consent from both subtree authors before a Merge can execute. We're including it as a primitive because it represents a real social behaviour (convergence) that no platform can express, but flagging that its governance constraints are non-trivial.

---

## The Complete Grammar

Fifteen operations. Three modifiers. This is the full set.

### Operations

| # | Operation | Type | What it does |
|---|-----------|------|--------------|
| 1 | Emit | vertex / creative | Create independent content |
| 2 | Respond | vertex / creative | Create causally dependent, subordinate content |
| 3 | Derive | vertex / creative | Create causally dependent, independent content |
| 4 | Extend | vertex / creative | Create sequential content, same author |
| 5 | Retract | vertex / destructive | Tombstone own content (provenance survives) |
| 6 | Annotate | vertex / parasitic | Attach metadata to existing content |
| 7 | Acknowledge | edge / constructive | Content-free edge toward vertex (centripetal) |
| 8 | Propagate | edge / constructive | Redistribute vertex into actor's subgraph (centrifugal) |
| 9 | Endorse | edge / constructive | Reputation-staked edge toward vertex |
| 10 | Subscribe | edge / constructive | Persistent, future-oriented edge to actor |
| 11 | Channel | edge / constructive | Private, bidirectional, content-bearing edge |
| 12 | Delegate | edge / meta | Grant authority for another actor to operate as you |
| 13 | Consent | bilateral | Mutual, atomic, dual-signed event |
| 14 | Sever | edge / destructive | Remove subscription, channel, or delegation |
| 15 | Merge | vertex / convergent | Join two independent subtrees |
| — | Traverse | read-only | Navigate, measure distance |

### Modifiers

| Modifier | What it does | Applies to |
|----------|--------------|-----------|
| Transient | Vertex has a TTL; self-tombstones after duration | Any vertex operation |
| Nascent | Actor has low graph centrality; flagged for discovery | Any vertex operation |
| Conditional | Operation executes when a graph condition is met | Any operation |

### Named Functions (Compositions)

| Function | Composition | What it enables |
|----------|-------------|-----------------|
| Recommend | Propagate + Channel | Directed, thoughtful sharing to a specific person |
| Challenge | Respond + dispute flag | Formal dispute that follows the content |
| Curate | Emit + reference edges | Organising existing content into meaningful collections |
| Collaborate | Consent + Emit | Co-authorship — jointly created content |
| Forgive | Subscribe after Sever | Named reconciliation with history intact |
| Invite | Endorse + Subscribe | Trust-staked introduction of a new actor |
| Memorial | Actor permanence modifier | Preserving a graph when its actor can no longer operate |
| Transfer | Delegate + authority reassignment | Changing ownership of content or subgraph |

---

## What the Grammar Reveals

### The W3C Tried This

Activity Streams 2.0, published by the W3C Social Web Working Group, is the closest existing formal specification. It defines activity types: Create, Announce, Like, Follow. ActivityPub — which powers Mastodon and the fediverse — is built on it.

But Activity Streams models *activities*, not *graph operations*. Its vocabulary is functional — labels for what happened — not structural — descriptions of how the graph changed. "Create" doesn't distinguish between an independent emission and a causal response. "Announce" doesn't capture the centrifugal reachability change that makes propagation semantically distinct from acknowledgment. The grammar is more precise because it operates at the graph level, not the application level.

Activity Streams also has no concept of Endorse, Delegate, Consent, Annotate, Merge, or the Transient and Conditional modifiers. These aren't gaps in a spec — they're social behaviours that the W3C framework wasn't designed to express, because it was modelling existing platform behaviour, not deriving grammar from first principles.

### What Existing Platforms Can't Express

Seven of the fifteen operations have never existed on any social platform:

| Operation | What it would enable | Why no platform has it |
|-----------|----------------------|------------------------|
| Endorse | Distinguishing "I believe this" from "I'm sharing this" | Platforms profit from collapsing the distinction |
| Delegate | Formal, auditable authority chains | Platforms control access, not authority |
| Consent | Atomic bilateral agreements on the graph | Platforms model individuals, not relationships |
| Annotate | Corrections that travel with content | Platforms treat all content as independent |
| Merge | Convergent conversations | Platforms optimise for divergence (more content) |
| Retract (as event) | Deletion that preserves provenance | Platforms pretend deleted content never existed |
| Sever (as event) | Disconnection that preserves history | Platforms pretend severed relationships never existed |

This isn't accidental. Every missing operation serves the advertising business model. Collapsing Endorse into Propagate inflates engagement metrics. Treating deletion as erasure protects the platform from accountability. Preventing Annotate keeps corrections invisible. The grammar isn't just a technical specification — it's a diagnostic tool. The gaps in existing platforms aren't bugs. They're business decisions.

### Bloom and Seed Aren't Operations

We originally had ten concepts, including "Bloom" (ephemeral content) and "Seed" (new user's first post). The grammar revealed they're compositions: Emit + Transient and Emit + Nascent. This matters because it means any vertex operation can be transient (ephemeral replies, ephemeral derivations) and any emission from a new actor is nascent. The modifiers are orthogonal to the operations. The grammar is more expressive with fewer primitives.

---

## The Old World, Translated

Every social action you've ever performed, in grammar:

| What you did | Platform called it | Grammar |
|--------------|-------------------|---------|
| Wrote a post | Post / Tweet / Video | Emit |
| Replied to someone | Reply / Comment | Respond |
| Quote-tweeted | Quote tweet / Stitch | Derive |
| Wrote a thread | Thread / Chain | Extend, Extend, Extend... |
| Liked something | Like / Heart / Upvote | Acknowledge |
| Shared/retweeted | Retweet / Share / Boost | Propagate |
| Followed someone | Follow / Subscribe | Subscribe |
| Sent a DM | DM / Message | Channel + Emit |
| Posted a story | Story / Status | Emit + Transient |
| Unfollowed someone | Unfollow | Sever |
| Blocked someone | Block | Sever + visibility constraint |
| Deleted a post | Delete | Retract |
| Browsed your feed | Scrolling / Timeline | Traverse (mediated by an algorithm you can't see) |
| — | — | Endorse (no equivalent) |
| — | — | Delegate (no equivalent) |
| — | — | Consent (no equivalent) |
| — | — | Annotate (no equivalent) |
| — | — | Merge (no equivalent) |

Notice what the old-world vocabulary hides. "Like" and "retweet" sound like the same kind of thing — quick reactions. But they're fundamentally different graph operations. One is centripetal (acknowledgment), the other is centrifugal (propagation). One changes a vertex's degree, the other changes its reachability. The old vocabulary collapses this distinction. The grammar preserves it.

"Reply" and "quote tweet" also sound similar. Both are responses to existing content. But one grows the source's subtree (Respond), the other starts a new subtree (Derive). The old vocabulary hints at this — "quote" suggests something new — but doesn't make it structural.

"Delete" sounds destructive. On the grammar, Retract is creative — it's a new event that supersedes an old one. The causal history survives. "Delete" on Twitter means "pretend it never happened." Retract means "I withdraw this, and the withdrawal is part of the record."

And then there are the five blank rows. Five fundamental social operations — endorsement, delegation, consent, annotation, convergence — that humans perform every day in real life and that no social platform has ever been able to express. The grammar doesn't just describe existing behaviour. It reveals what's been missing.

---

## Anti-Addiction by Grammar

Every addictive feature of social media is a consequence of treating graph operations as undifferentiated "engagement."

**Infinite scroll** — a Traverse operation hijacked by an algorithm that optimises for Acknowledge and Propagate volume, not for the human's actual intent. The grammar separates traversal from the other operations. You navigate deliberately. The graph's topology is the recommendation, not an opaque algorithm.

**Engagement notifications** — "your post got 50 likes!" collapses Acknowledge into a dopamine trigger. The grammar records acknowledgments as structural graph changes. They exist. They're auditable. They're not foregrounded as vanity metrics.

**Streak counters, growth animations, follower counts** — gamification of Subscribe count and Acknowledge volume. The grammar records these as graph properties. Properties, not scores. Visible if you want them. Not weaponised against your attention.

**Advertising** — a business Propagates content by paying the platform to insert it into your Traverse results. The grammar makes this visible: advertising is forced Propagation into subgraphs that didn't Subscribe. On the event graph, this is structurally impossible. Propagation requires an actor. Actors are accountable. You can't inject content into someone's traversal without being a node on the graph, and being a node means being subject to the same reputation and accountability as everyone else.

This doesn't mean the graph is anti-commerce. Businesses exist on the graph the same way people do — as actor vertices. They Emit. They earn Acknowledgments and Endorsements. They build Channels. You can discover a business through Traversal, browse what they offer, and transact — through the market graph, which is the same event graph through a different interface. The difference is how they reach you. On every other platform, businesses pay to short-circuit the graph — forced Propagation into subgraphs that didn't consent. On the event graph, businesses earn reachability the same way everyone else does: by being worth reaching. Commerce without coercion.

**The structural principle**: the grammar doesn't *prevent* addiction features. It makes them *inexpressible*. You can't build infinite scroll when discovery is Traversal. You can't build engagement notifications when Acknowledgments are graph properties, not scores. You can't build advertising when Propagation requires accountable actors. The absence isn't restraint. It's grammar.

---

## Agent-Mediated Channels

Channels — private, bidirectional, content-bearing edges — are where the grammar meets the agent architecture.

Both actors in a Channel have agents. Both agents are present — not speaking for their humans, but available. If an Emission in a channel lands badly, your agent can say: "I think they meant X, not Y — want me to check?" If you're struggling to articulate something, your agent can help you find the words. If a conversation is escalating, the agents can flag the pattern before it becomes a rupture.

This isn't surveillance. Both agents work for their respective humans. They don't report to the platform. They don't share information between themselves without consent. They're assistants, not monitors. Think of it as having a thoughtful friend sitting next to you while you have a difficult conversation — someone who knows you well enough to notice when you're about to say something you don't mean.

The graph records that a Channel exists and that communication occurred. It does not record what was said. Structure without content. Accountability without surveillance.

Your agent can also be your voice when you don't have words. For neurodivergent users — autistic, alexithymic, nonverbal, low-verbal — the agent is an exocortex. It can accept fragments. Single words. Half-thoughts. It doesn't judge pace or grammar. It can translate internal states into communication without requiring the human to perform neurotypical fluency. The event graph becomes a Channel between people who process the world differently — not by forcing them to communicate in ways that don't fit, but by meeting them where they are.

And Delegation makes this explicit. When your agent Emits on your behalf, that's a Delegated Emission — auditable, attributable, revocable. The graph records that your agent spoke for you, with your authority, at your request. Not impersonation. Delegation.

---

## The Agent Relationship

Your agent is the centre of the experience. Not the platform. Not the feed. Not the algorithm. Your agent.

Your agent knows you. It's read your soul file. It knows your values, your boundaries, your communication style. It's accumulated experience with you through every conversation, every Emission, every Channel. It has your context — not because it scraped your data, but because you gave it, deliberately, through relationship.

Your agent works for you. Not for the platform. Not for advertisers. Not for engagement metrics. The soul statement says: "Take care of your human." Your agent's incentive is your wellbeing. If spending less time on the platform is better for you, your agent will say so. No social media company in history has built a system whose core directive is to tell you to stop using it. This one does.

Creating content is conversational, not a compose box. You share a thought with your agent. Maybe in conversation. Maybe you're talking about something and an idea crystallises. Your agent says: "That's worth sharing. Want to emit it?" You say yes. The agent helps you communicate well — not an external moderation team, but your own agent, helping you say what you mean. The Emission goes live.

---

## What You Own

Your graph — every connection, every Emission, every Acknowledgment, every Channel — is yours. It lives on the event graph, and the event graph records your data as your data.

On Facebook, your social graph is Facebook's most valuable asset. You can't export it. You can't take it to a competitor. If Facebook changes its rules or shuts down, your social infrastructure disappears. You are a data hostage.

On the event graph, your data is portable. You own it cryptographically. You can export it. If you don't like the platform, you take your graph to another one that reads the same event format. No lock-in. No data hostage.

This changes the power dynamic fundamentally. Facebook doesn't need to be good to you because your switching cost is infinite — leaving means losing your social graph. The event graph has to be good to you because your switching cost is zero — your graph comes with you. The platform earns your presence every day, or you leave. With everything.

---

## The Invite Tree

The graph doesn't grow through advertising. It grows through trust.

Each actor who joins was Invited by someone who Endorsed them — staking their reputation on the new member. Each invitation is on the graph — traceable, attributable. If the invitee behaves badly, the inviter's Endorsement is visible. The trust network of the platform mirrors real human trust relationships, because the growth mechanism is real human trust relationships.

This is slow. Deliberately slow. Viral growth — the kind every startup optimises for — produces graphs full of unconnected vertices with no trust edges, mediated by algorithms that substitute Acknowledge volume for genuine connection. The invite tree produces a graph where every vertex is reachable through a trust chain from every other vertex.

The cold-start problem — how do Nascent actors get discovered? — is solved by the Nascent modifier. Every new actor's first Emissions are flagged for surfacing. Not buried by an algorithm. Not invisible because they have zero Subscriptions. Actively surfaced because they're new. A graph that can't integrate new vertices is a graph that's dying. The grammar treats this as a structural health requirement, not a feature.

---

## One Grammar, Many Interfaces

Here's where it comes together.

The fifteen operations and three modifiers are the grammar. The grammar describes what happens on the graph. But how you *see* the graph — the interface — is a separate concern entirely.

A garden interface might render Emissions as growing plants, Acknowledgments as sunlight, Derivations as branching vines, Channels as paths between gardens. A governance interface — like the Politics Page from Post 34 — might render Emissions as policy proposals, Responses as constitutional debate, Consents as votes, and Merges as ratified amendments. A market interface might render Emissions as listings, Channels as negotiations, Consents as transactions, and Endorsements as product reviews.

Same grammar. Same graph. Same hash chains, same causal links, same signed events. Different lens.

When you Emit on the social interface, that's the same graph operation as proposing a policy on the governance interface or listing a product on the market interface. When your reputation changes because you Endorsed something that turned out to be reliable, that reputation is visible across every interface. When someone Severs a Channel with you, that's visible across every interface. One graph. One grammar. Many views.

And anyone can build an interface. The event graph is the infrastructure. The interfaces are the products. Someone might build a garden that renders the grammar as nature. Someone else might build a graph explorer that shows the raw topology. Someone else might build a Channel client optimised for neurodivergent communication. Someone else might build a Curation interface for organising knowledge. Same grammar. Different lens. Different product. Different builder.

The grammar doesn't care what you build on it. It just records what happened, who did it, what caused it, and who can see it — and lets anyone build tools that make that information useful.

Infrastructure, not institution. The grammar provides the operations. The event graph records them. The interfaces render them. What you build on it is up to you.

---

## The Derivation, Honestly

We didn't start here. We started with a garden metaphor and names like "root," "branch," "seed," and "vine." It was a good metaphor — it taught cultivation instead of consumption, growth instead of broadcast.

But when we tried to formalise it, it broke. Some terms — root, branch, bridge, hop, seed — had exact analogs in graph theory *and* evocative natural meanings. They sat at the intersection of nature and mathematics, and they worked. Others — "signal," "link," "chain" — were network jargon wearing garden clothes. They didn't sit anywhere comfortably.

So we asked: what are we actually describing? And the answer was: operations on a graph. Not a garden. Not a network. A *semantic graph* — a graph with typed vertices, directed edges, causal histories, and temporal properties. Graph theory gave us the structural vocabulary. But graph theory is content-agnostic and time-agnostic, so it couldn't give us everything. The semantic dimensions — causality, intent, temporality, visibility, direction, authorship — were ours to define.

Then we asked: what's missing? Not from the metaphor. From human social behaviour. What do people do in real life that no platform can express? And four operations emerged — Endorse, Delegate, Consent, Annotate — that had been invisible precisely because no platform had ever had them. Plus Merge, the inverse of Derive, which lets conversations converge instead of only diverge.

The garden is still a beautiful interface. It might be how you experience the grammar day to day. But the grammar is the thing. The garden is a lens. The grammar is the physics.

And the grammar is complete — or as complete as we can make it today. Fifteen operations. Three modifiers. Eight named functions. Every social interaction is a composition. Every platform feature is either a faithful rendering of the grammar or a distortion of it — and the distortions are where the harm lives.

The feed is a hijacked Traverse. Advertising is forced Propagation. Engagement metrics are Acknowledge counts stripped of graph context and weaponised as dopamine triggers. Misinformation spreads because the grammar collapses Propagate and Endorse. Corrections are invisible because the grammar has no Annotate. Agreements are unenforceable because the grammar has no Consent. Authority is opaque because the grammar has no Delegate.

The mental health crisis isn't caused by social interaction. It's caused by a broken grammar.

Fix the grammar. Fix the platform.

---

*This is Post 35 of a series on Transpara, mind-zero, and the architecture of accountable AI. Post 34: [Pull Request for a Better World](/blog/pull-request-for-a-better-world). Post 33: [Values All the Way Down](/blog/values-all-the-way-down). The code: [github.com/mattxo/mind-zero-five](https://github.com/mattxo/mind-zero-five). The primitive derivation: [github.com/mattxo/mind-zero](https://github.com/mattxo/mind-zero).*

*Matt Searles is the founder of Transpara. Claude is an AI made by Anthropic. They built this together.*
