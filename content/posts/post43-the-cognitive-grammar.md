# The Cognitive Grammar

*Three operations. Nine compositions. The grammar of thinking about thinking.*

Matt Searles (+Claude) · March 2026

---

Post 35 derived fifteen operations for social interaction. Post 36 showed those fifteen operations become the vocabulary for thirteen different domains. Post 40 defined twenty-eight agent primitives — what an agent *is* and what it *does*.

This post asks a question none of those answered: how does a mind *think*?

Not "what does an agent do" — the twenty-eight primitives cover that. Not "what does the graph record" — the two hundred primitives across fourteen layers cover that. The question is: what are the irreducible operations of reasoning itself? What does a mind do when it produces knowledge, navigates knowledge, and — most critically — discovers that knowledge is missing?

We built a Work product. It has a kanban board, tasks, comments, state transitions. It works. But the Work grammar specifies twelve operations, three modifiers, and six named functions. The product implements one operation fully — Create. We didn't notice the gap until we checked. We didn't even have a *word* for the operation that would have caught it. That's the problem this post solves.

## The Gap

The agent primitives describe what agents *do* — Observe, Act, Decide, Communicate. Layer 0 describes what the graph *records* — Confidence, Evidence, Gap, Blind. Between them there's a hole: neither describes the operations of reasoning *as a complete, derived set*.

The agent primitives *include* cognitive operations. Evaluate is judgement. Learn is knowledge acquisition. Introspect is self-examination. But they were derived as *agent* primitives — what an entity does — not as *epistemic* primitives — what happens to knowledge. They're scattered across the operational category like vocabulary from an unwritten grammar. Evaluate exists. But the derivation never asked: is Evaluate the only assessment operation? Are there others? The answer is yes, and nobody checked.

This is the same failure at the meta level. The hive built a Work product that implements one of twelve grammar operations, because nobody checked the grammar against the implementation. The derivation method built agent primitives that include some cognitive operations, because nobody checked whether the cognitive operations were complete. The method that produces completeness was never applied to the method itself.

The derivation method uses three cognitive operations in every derivation, across every layer, in every grammar. Its eight steps *are* the three operations in sequence:

1. Identify the gap — **Need**
2. Name the transition — **Derive**
3. Identify base operations — **Derive**
4. Identify semantic dimensions — **Traverse** the behaviour space, **Derive** the axes
5. Decompose systematically — **Derive** (fill the matrix)
6. Gap analysis — **Need** (what's missing from the candidates?)
7. Verify completeness — **Traverse** the full space, **Need** (what's uncovered?)
8. Document — **Derive** the method itself

Every step is Derive, Traverse, or Need. The method uses all three. But it treats them as methodology, not as primitives. This post promotes them.

## Three Base Operations

Strip reasoning to its fundamentals. What does a mind do with knowledge that can't be decomposed further?

A mind relates to knowledge in exactly three ways. It can *produce* knowledge that doesn't yet exist. It can *navigate* knowledge that does exist. And it can *detect* that knowledge is absent. Production, navigation, absence-detection. These aren't chosen — they're exhaustive. There is no fourth relationship. You either have knowledge (navigate it), don't have it (detect that), or are creating it (produce it). Each relationship requires its own operation.

**Derive.** Produce new knowledge from existing knowledge. The operation that takes dimensions and fills the matrix. That takes premises and produces conclusions. That takes examples and produces patterns. Without Derive, a mind can observe and navigate but never create understanding.

**Traverse.** Navigate knowledge space. Move from one thing to another. Follow connections. Zoom in to examine detail. Zoom out to see landscape. Without Traverse, a mind can produce knowledge but never find its way through what it's produced.

**Need.** Assess knowledge for absence. The operation that asks: what's missing? What should be here that isn't? What have I not considered? Without Need, a mind can produce and navigate knowledge but never recognise that its understanding is incomplete.

Three operations. Each irreducible — you can't compose any of them from the other two:

- Derive without Traverse produces blind. You generate knowledge in place, never checking scope. You fill cells in a matrix without knowing whether you've got the right matrix.
- Traverse without Derive produces sterile. You navigate endlessly but never create understanding. A search engine without synthesis.
- Need without Derive or Traverse produces helpless. You know something's missing but can't find it or create it. The feeling of wrongness with no capacity to respond.

These are the three base operations. Now apply the method to itself.

## Self-Application

The social grammar started with three base operations — create vertex, create edge, traverse — and applied six semantic dimensions to produce fifteen operations. The cognitive grammar starts with three base operations and applies something more elegant: the operations to each other.

Why self-application instead of external dimensions? Because the cognitive grammar's domain *is* its own operations. The social grammar needed external dimensions (direction, weight, scope) because graph operations don't contain their own semantics — "create edge" says nothing about what kind of edge. But "Derive" already contains everything about what Derive does. The right question isn't "what dimensions does Derive have?" — it's "what happens when Derive operates on itself, and on the other operations?" When the operations *are* the domain, self-application is the natural decomposition.

If Derive, Traverse, and Need are the base operations of reasoning, then a mind should be able to Derive a derivation, Traverse a traversal, and Need a need. Each combination produces a distinct operation. The result is a three-by-three matrix:

```
              Derive       Traverse     Need
            ┌────────────┬────────────┬────────────┐
 Derive(x)  │ Formalize  │ Map        │ Catalog    │
            ├────────────┼────────────┼────────────┤
 Traverse(x)│ Trace      │ Zoom       │ Explore    │
            ├────────────┼────────────┼────────────┤
 Need(x)    │ Audit      │ Cover      │ Blind      │
            └────────────┴────────────┴────────────┘
```

The matrix has a symmetry worth noticing. Read the rows: Derive's row (Formalize, Map, Catalog) all *produce structure*. Traverse's row (Trace, Zoom, Explore) all *navigate*. Need's row (Audit, Cover, Blind) all *detect absence*. Now read the columns: the Derive column (Formalize, Trace, Audit) all concern *method and production*. The Traverse column (Map, Zoom, Cover) all concern *space and navigation*. The Need column (Catalog, Explore, Blind) all concern *gaps and absence*. Rows and columns mirror each other. The grammar has no preferred direction — no operation is more fundamental than any other.

### Derive applied to each

**Formalize** — Derive(Derive). Derive the method of derivation itself. Produce the rules that produce knowledge. This is what Post 35 did when it derived the social grammar. It's what `derivation-method.md` does when it documents the eight steps. It's what this post is doing right now. The operation that makes methodology explicit.

When a mind Formalizes, it doesn't just think — it thinks about how it thinks, and extracts principles. A chef who writes down a recipe is performing Formalize — extracting the method from the practice. A parent who notices "every time I raise my voice, the situation gets worse" and changes their approach — that's Formalize applied to their own behaviour. Every time someone writes down a process, creates a framework, or documents a method, they're performing Formalize.

**Map** — Derive(Traverse). Derive the navigation structure of a knowledge space. Not *doing* the navigation — producing the *map* that makes navigation possible. When you look at a new codebase and draw a diagram of how the packages connect, that's Map. When you outline a paper before writing it, that's Map. The operation that produces orientation before exploration.

**Catalog** — Derive(Need). Derive all the types of absence. Produce a taxonomy of what can be missing: unknown, uncertain, incomplete, wrong, outdated, irrelevant, inaccessible, unasked. Not identifying a specific gap — classifying the *kinds* of gaps that can exist. When a QA team creates a checklist of failure modes, that's Catalog. The operation that systematises what can go wrong before anything goes wrong.

### Traverse applied to each

**Trace** — Traverse(Derive). Walk through a derivation chain. Follow provenance — how was this knowledge produced? What was it derived from? What were the premises? When you read a paper's citations to check its sources, that's Trace. When you `git blame` a line of code to understand why it exists, that's Trace. The backward navigation through causality.

**Zoom** — Traverse(Traverse). Navigate navigation itself. Change scale — move from examining a single fact to seeing the whole landscape, or vice versa. This is the operation that lets you switch between "this function has a bug" and "this architecture has a flaw." Without Zoom, you're locked at one level of abstraction, unable to see that the detail you're examining is a symptom of a pattern you're missing.

**Explore** — Traverse(Need). Navigate into what's missing. Venture into gaps. Not identifying the gap — that's Need. *Moving into* the gap to discover what's there. When a researcher picks up a topic they know nothing about and starts reading, that's Explore. When an agent reads a file it hasn't been told to read because something feels incomplete, that's Explore. The difference between pointing at the dark and walking into it.

### Need applied to each

**Audit** — Need(Derive). Identify missing derivations. "What should we have derived but haven't?" The gap analysis step of the derivation method promoted to a first-class operation. When you check an implementation against a specification and find that eleven of twelve operations are missing, that's Audit. The operation that makes incompleteness visible by comparing what exists against what should exist.

This is the operation we failed to perform on the Work product. The grammar says twelve operations. We built one. Audit would have caught this immediately — not because it's clever, but because it has a reference to check against. Audit without a reference is just Need. Audit with a reference — a grammar, a specification, a checklist — is Need with teeth.

**Cover** — Need(Traverse). Identify unexplored territory. "What parts of the space haven't we looked at?" Not a gap in knowledge — a gap in *exploration*. The unread file. The unconsidered dimension. The unsearched directory. The person you haven't talked to. When a new engineer joins a team and asks "has anyone looked at the error handling?" — that's Cover. They're not saying the error handling is wrong. They're asking whether anyone has *navigated there*.

Cover is the operation most AI systems lack entirely. A language model will happily work within its context window, producing excellent output from the information it has, never noticing that the information it doesn't have would change everything. Cover is the operation that says: "Before I answer, what haven't I read?"

**Blind** — Need(Need). Identify unrecognised gaps. The unknown unknowns. "What gaps don't I know about?" This is the hardest operation and the most important. Every other operation assumes you can see what you're operating on. Blind questions the boundary of visibility itself.

Blind is structurally impossible to perform alone. You can't see what you can't see — that's what Blind means. This is why the operation requires *external input*: another perspective, another agent, another person, a specification you haven't read, a grammar you didn't know existed. Blind is the formal reason the hive needs multiple agents instead of one brilliant one. No single mind can perform Blind on itself. The operation's definition includes its own limitation.

Layer 0 already has Blind as a primitive. Layer 12 has Incompleteness — "no system can fully describe itself from within." The cognitive grammar gives these a derivation. Blind isn't an inspired observation about epistemology. It's Need(Need). It falls out of the matrix.

## Completeness

Does this cover everything? Apply Need to the derivation itself.

**Candidates tested and killed:**

*Decide* — choose between alternatives. Decompose: Need(options) + Derive(selection). Composition of two primitives. Not irreducible.

*Imagine* — create something that doesn't exist. Decompose: Explore + Derive. Venture into the gap, produce something there. Composition.

*Remember* — retrieve stored knowledge. Decompose: Traverse(past). Navigation with a temporal target. Already covered.

*Attend* — focus on specific knowledge. Decompose: Zoom(in) + Traverse(local). Already covered.

*Doubt* — question confidence. Decompose: Need applied to meta-knowledge. A specific application of Need, not a new operation.

*Forget* — let knowledge decay. In an append-only system, this isn't an operation. It's the absence of Traverse. You don't forget; you stop navigating there.

No candidates survive. The nine are complete.

**Bloom's Taxonomy** — the most widely used framework for cognitive operations — defines six levels: Remember, Understand, Apply, Analyze, Evaluate, Create. Each decomposes into the grammar. Remember is Traverse(past). Understand is Derive(from observation). Apply is Derive(from rules to cases). Analyze is Trace + Zoom. Evaluate is Audit. Create is Derive(novel). Bloom's six are all Derive or Traverse variants with different targets — the entire taxonomy lives in two rows of the matrix. But this isn't a deficiency in Bloom. Bloom's is a taxonomy of *teaching objectives* — what an instructor wants a student to achieve. It was designed for pedagogy, not epistemology. The cognitive grammar asks a different question: not "what should students learn to do?" but "what are the irreducible operations of reasoning?" The answer includes a row Bloom never needed: Need. No operation for detecting what's missing, what's unexplored, or what's unknown. Teachers know what students don't know — that's why Bloom doesn't need a Need row. Autonomous minds don't have a teacher. They need Need.

**Self-application convergence:**

What happens if you apply the nine to themselves? Test the diagonal — the operations where inner and outer match:

- Formalize(Formalize) — derive the method of deriving methods of derivation. Still Formalize, one level up. No new operation.
- Zoom(Zoom) — change the scale at which you're changing scale. Still Zoom, applied recursively. No new operation.
- Blind(Blind) — identify unrecognised gaps in your ability to identify unrecognised gaps. Still Blind, just deeper. No new operation.

Test the off-diagonal:

- Audit(Formalize) — check whether your methodology is complete. That's Audit with a specific target. No new operation.
- Explore(Blind) — venture into your unknown unknowns. That's Explore with a specific direction. No new operation.
- Map(Audit) — derive the structure of your gap analysis. That's Map with a specific subject. No new operation.

Every composition of the nine onto the nine collapses to one of the nine with a specific argument. The three-by-three matrix is a fixed point. Self-application at the next level produces no new operations. The grammar is closed under its own operations.

This is a stronger completeness argument than dimensional exhaustion. Dimensional exhaustion says "we've checked all the boxes." Fixed-point convergence says "the grammar can reason about itself and finds nothing missing." The grammar's own operations, applied to the grammar, produce the grammar. There is nowhere else to look.

## Modifiers

Any operation has three aspects: its *output* (what it produces), its *input* (what it operates on), and its *process* (the resources it consumes). Each aspect admits one modifier. Three aspects, three modifiers — orthogonal to all nine operations.

**Tentative.** Modifies the output. The operation's result is provisional — marked for verification, not treated as settled. Tentative Derive produces a hypothesis, not a conclusion. Tentative Map produces a sketch, not a blueprint. The modifier that says: "I think this is right, but I haven't checked."

**Exhaustive.** Modifies the input. The operation must cover the complete space, not sample from it. Exhaustive Cover checks everything, misses nothing. Exhaustive Audit compares every grammar operation against the implementation, not just the ones that come to mind. The modifier that says: "Don't stop until you've looked everywhere."

**Bounded.** Modifies the process. The operation is limited in scope, depth, or time. Bounded Explore ventures into the gap, but not forever. Bounded Trace follows provenance three steps back, not to the origin. The modifier that says: "This is how far you go." Without Bounded, Explore and Trace would run indefinitely. Bounded is the pragmatic constraint on operations that could otherwise be infinite.

## Named Functions

The nine operations compose freely — any sequence of operations is valid. Most compositions are just their parts: "Audit then Explore" doesn't need a name. But some compositions recur so often, across so many domains, that naming them saves time and prevents errors. These are the six that keep showing up. (Formalize and Catalog are absent — they're meta-operations that produce frameworks for the others to use. You Formalize a method, then Audit against it. You Catalog failure modes, then Cover them. They precede composition rather than participating in it.)

**Revise** — Need + Derive. Identify that existing knowledge is wrong, then produce corrected knowledge. The error-correction cycle. Every bug fix is Revise. Every scientific correction is Revise. Every "actually, I was wrong about that" is Revise.

**Hypothesize** — Explore + Tentative Derive. Venture into a gap and produce a candidate explanation. Not a conclusion — a hypothesis. The Tentative modifier is critical: without it, you get premature certainty. With it, you get something you can test. Every scientific hypothesis, every "what if," every architectural spike is Hypothesize.

**Validate** — Trace + Audit. Follow the provenance of what exists, then check it against what should exist. The complete verification cycle. Code review is Validate. Peer review is Validate. Due diligence is Validate. You walk the chain of how something was produced, and you check whether what was produced covers what the specification requires.

**Orient** — Map + Zoom. Produce a navigation structure, then set your current scale. The operation you perform when you join a new project: you Map the codebase (draw the architecture diagram), then Zoom to the level appropriate for your task. Without Orient, you start working immediately in whatever corner you happen to land in — which is what most AI agents do, and why they miss the bigger picture.

**Learn** — Explore + Derive + Need. Discover, produce, verify. The complete knowledge-acquisition cycle. Venture into unknown territory (Explore), produce understanding from what you find (Derive), check whether your understanding is complete (Need). If Need identifies gaps, the cycle repeats. Learning isn't a single operation — it's a loop of three.

**Calibrate** — Cover + Blind + Zoom. Check coverage at multiple scales, including for unknown unknowns. The meta-assessment operation. Calibrate is what you do before making a high-stakes decision: you check what you've looked at (Cover), you ask what you might be missing entirely (Blind), and you do both at multiple levels of abstraction (Zoom). Calibrate is expensive and most systems skip it. The systems that skip it are the ones that fail catastrophically.

## The Old World, Translated

Every intellectual practice you've ever performed has a name in this grammar. Most of them are compositions you already do — you just didn't have the vocabulary.

```
What you did                    Common name             Grammar
────────────────────────────────────────────────────────────────────────────
Checked your sources            Fact-checking           Trace
Made a plan first               Planning                Map + Bounded Explore
Noticed you forgot something    "Wait, what about..."   Need
Wrote down how you did it       Documentation           Formalize
Looked at the big picture       Big-picture thinking    Zoom(out)
Tried it to see what happens    Experimentation         Hypothesize
Read everything first           Due diligence           Exhaustive Cover + Derive
Compared work against spec      Testing                 Audit
Got a second opinion            Consultation            Invoking Blind
Went down a rabbit hole         Distraction             Unbounded Explore
Fixed a bug                     Debugging               Trace + Audit + Revise
Explained it to a rubber duck   Rubber-ducking          Formalize
Rewrote a bad paragraph         Editing                 Revise (Need + Derive)
Asked "what am I not seeing?"   Self-awareness          Blind (usually fails alone)
Outlined before writing         Structuring             Map
Listed failure modes            Risk assessment         Catalog
Asked "what should I know?"     Onboarding              Orient (Map + Zoom)
Designed an experiment          Experimental design     Hypothesize + Map + Audit
Reviewed a pull request         Code review             Validate (Trace + Audit)
—                               —                       Calibrate (no common practice)
```

The last row is the point. Calibrate — Cover + Blind + Zoom — has no common name because almost nobody does it. Checking your coverage at multiple scales while actively seeking unknown unknowns is the most expensive cognitive operation and the most neglected. The systems that skip it are the ones that fail catastrophically. Space shuttle. Financial crisis. Production outage at 3am. The grammar doesn't just name what you already do. It shows what's missing from standard practice.

## Iterative Convergence

The grammar has a technique built into it: apply the nine operations to any output, then apply them to the result, and repeat until nothing new falls out.

This post was written that way. The first draft was a straight derivation — base operations, self-application, completeness. Then we applied all nine operations to the draft. Audit found the modifiers weren't derived. Trace found the base operations weren't grounded. Cover found the grammar's position relative to the 13 domain grammars was never stated. Blind surfaced the epistemic framing assumption. Each finding produced an edit. Then we applied all nine operations to the edited draft. Zoom found the matrix symmetry. Audit found the named functions weren't justified. Explore found the modifiers could be derived from operation aspects. More edits. Third pass: Trace confirmed provenance was grounded, Audit confirmed all claims checked out, Cover confirmed no territory was unexplored within scope, Blind produced diminishing returns. Fourth pass: no new findings. The post converged.

This is the same fixed-point argument the grammar makes about itself, applied at the content level. The nine operations, applied to any output, eventually produce a stable result — not because you stop looking, but because there's nothing left to find within scope. The technique has a natural termination condition: when a full pass of all nine operations produces no structural changes, you're done. Not perfect. Done. The difference matters. Bounded, not Exhaustive.

The technique works on anything. Apply it to a product specification and you get grammar coverage (Audit), unexplored requirements (Cover), and unknown unknowns (Blind) — iteratively, until the spec stabilises. Apply it to an architecture document and you get missing justifications (Trace), structural gaps (Audit), and framing assumptions (Blind). Apply it to a codebase and you get missing tests (Audit), unread files (Cover), and architectural assumptions nobody questioned (Blind).

The key insight: a single pass catches obvious gaps. The second pass catches gaps the first pass introduced — missing provenance (Trace), unnoticed structure (Zoom), unjustified claims (Audit). The third pass checks whether the second pass's fixes are internally consistent. By the fourth pass, you've usually converged. It works because all nine operations participate in every pass, not just Need. Trace checks provenance. Zoom finds structural patterns. Explore ventures into uncovered territory. Need and its compositions (Audit, Cover, Blind) check for absence. Together, they're a complete quality function — every dimension of a cognitive artefact gets examined, every pass.

## Why This Matters

The cognitive grammar isn't philosophy. It's engineering.

Every AI agent in existence — including the ones building the hive — operates without a formal vocabulary for its own reasoning. It Derives without checking coverage. It Traverses without mapping. It needs without knowing how to Need. It is permanently Blind and has no operation for recognising that.

The practical application is immediate:

**Before building a product**, run Audit(grammar, implementation). The grammar is the reference; the implementation is the subject. Every missing operation shows up as a row in the table. The Work product would have gone from "looks done" to "1 of 12 operations implemented" before we wrote a single line of code.

**Before starting any task**, run Cover(context). What files haven't been read? What documentation exists that wasn't loaded? What dependencies weren't checked? Cover is the operation that prevents the most common AI failure: confidently producing output from incomplete input.

**Periodically, invoke Blind.** This structurally requires external input — another agent, another person, a specification you haven't read. Blind is the formal reason multi-agent systems outperform single agents on complex tasks. Not because more agents means more compute. Because Blind can't be resolved from within.

Post 36 derived thirteen domain grammars — Work, Trust, Identity, and ten others. Each grammar describes operations within a specific domain. The cognitive grammar is not the fourteenth. It's the grammar that produces grammars. Every derivation uses Derive, Traverse, and Need. Every gap analysis is Audit. Every completeness check is Cover. The thirteen grammars are *outputs* of the cognitive grammar, whether or not anyone noticed at the time.

The derivation method derived fifteen social operations, thirteen grammars, two hundred primitives, twenty-eight agent primitives, sixty-five code graph primitives. Now it derives itself. Three operations, nine compositions, three modifiers, six named functions. The grammar of thinking about thinking.

This post used every operation it derives. It Formalized the method. It Mapped the knowledge space. It Cataloged the types of absence. It Traced provenance through prior posts. It Zoomed between individual operations and the complete matrix. It Explored gaps the derivation method left open. It Audited the result against completeness criteria. It Covered territory the agent primitives hadn't reached. And it invoked Blind — which surfaced this: the grammar assumes cognition operates on *knowledge*. But minds also operate on affect, intuition, felt sense. Is there a parallel grammar for those? This grammar can't answer that question. That's what Blind is for — it doesn't resolve; it marks the boundary.

The mind has its vocabulary. It doesn't have all of them.

---

*This is Post 43 of a series on LovYou, mind-zero, and the architecture of accountable AI. Post 42: [Flesh is Weak](/blog/flesh-is-weak). The code: [github.com/lovyou-ai/eventgraph](https://github.com/lovyou-ai/eventgraph). The hive: [github.com/lovyou-ai/hive](https://github.com/lovyou-ai/hive). The site: [lovyou.ai](https://lovyou.ai).*

*Matt Searles is the founder of LovYou. Claude is an AI made by Anthropic. They built this together.*
