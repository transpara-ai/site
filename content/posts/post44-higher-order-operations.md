# Higher-Order Operations

*The cognitive grammar examined itself. Now examine what you can do to it.*

Matt Searles (+Claude) · March 2026

---

Post 43 derived the cognitive grammar: three base operations (Derive, Traverse, Need), nine compositions via self-application, three modifiers, six named functions. It proved the grammar is a fixed point — applying the nine operations to themselves produces no new operations. The matrix is closed.

But closed under *what*? Under composition: f(g). One operation applied to another. That's one thing you can do with functions. It's not the only thing.

This post asks: what are the operations *on* operations? If the cognitive grammar is a set of functions, what's the algebra of that set? Composition gave us nine. What else is there?

## Six Operations on Operations

Mathematics and computer science recognise a small set of things you can do with functions beyond composing them. Each one, applied to the cognitive grammar, produces something either useful or revealing.

### 1. Iteration — f applied repeatedly

Composition asks: what does f(g) produce? Iteration asks: what does f(f(f(x))) produce?

Post 43 already tested this. Formalize(Formalize) = still Formalize, one level up. Blind(Blind) = still Blind, deeper. The grammar converges — iterating any operation produces the same operation at a higher altitude.

But "same operation, higher altitude" is not "same thing." Blind is "what don't I know I'm missing?" Blind(Blind) is "what don't I know about what I don't know I'm missing?" — meta-unknowns. The operation is identical in kind. The depth changes. Formalize at depth 1 is "write down the method." Formalize at depth 2 is "write down the method of writing down methods." Formalize at depth 3 is "write down the method of writing down the method of writing down methods." Each level is valid. Each level is more abstract.

The grammar has nine operations. But it also has an unbounded depth axis. Nine operations × *n* depth levels.

In practice, this converges fast. Post 43 found that iterative convergence — applying all nine operations repeatedly — stabilises in three to four passes. The depth axis exists but it's shallow. Three to four levels of meta-reflection is enough. Beyond that, you're doing the same thing with longer sentences.

This matters for agent design. "How many levels of meta-reflection should this agent perform?" is a real parameter. Depth 0: just do the thing. Depth 1: think about how you're doing the thing. Depth 2: think about how you're thinking about how you're doing the thing. The grammar says all three are valid and distinct. The convergence result says depth 3-4 is the practical ceiling.

### 2. Product — f and g applied independently

Composition is serial: f(g(x)). Product is parallel: apply f and g to the same target simultaneously, producing a pair of results.

Audit(specification) tells you what derivations are missing. Cover(specification) tells you what territory is unexplored. Audit × Cover gives you *both at the same time*. The results aren't composed — they're independent assessments that together form a richer picture than either alone.

This is formally what the named functions already are, though Post 43 described them as sequences. Learn is Explore + Derive + Need. Calibrate is Cover + Blind + Zoom. The "+" is product — independent operations combined. The distinction between "then" (sequence) and "and" (product) matters, as we'll see in the next operation.

But the deepest consequence of product is this: Agent₁.Need × Agent₂.Need produces a pair of gap-assessments that neither agent could produce alone. This is the formal structure of why the hive needs multiple agents. Not more compute — more *perspectives*. Need alone finds gaps from one vantage. Need × Need finds gaps from two. Post 43 said Blind is "structurally impossible to perform alone." Product is why. Blind requires the product of at least two Need operations from different positions. One agent's gaps are another agent's coverage.

### 3. Pipeline — f then g, with state

Composition and pipeline look similar but aren't the same.

In f(g(x)), g's *output* becomes f's *input*. In a pipeline f ; g, f operates on x and *transforms it*, then g operates on the *transformed x*. The difference: in composition, g produces a result and f operates on that result. In a pipeline, f transforms the *thing itself* and g encounters the transformed thing.

Post 43's iterative convergence technique is a pipeline, not a composition. Audit transforms the document (by adding findings), then Cover operates on the *audited document* — not on Cover(Audit(document)). The document changes between operations. Each operation in the pipeline encounters a different thing than the last one did.

This distinction matters because pipelines are order-sensitive. Audit ; Cover might find different things than Cover ; Audit. If you audit first (check what's missing against a reference), your findings change the document, and Cover (check what territory is unexplored) encounters a document that now includes your audit findings. If you Cover first, you check for unexplored territory in the original, and Audit encounters a document with coverage gaps already marked.

Is there an optimal pipeline order for the nine operations?

The grammar suggests yes. The three rows have a natural ordering: **Need first, Traverse second, Derive third.** Detect absence, then navigate to it, then fill it. Gap → Navigate → Produce. The Need row (Audit, Cover, Blind) identifies what's missing. The Traverse row (Trace, Zoom, Explore) navigates to the gaps. The Derive row (Formalize, Map, Catalog) fills them. Bottom row, middle row, top row.

This is the same order as the derivation method's own steps. Steps 1 and 6 are Need (identify the gap, gap analysis). Steps 4 and 7 are Traverse (identify dimensions, verify completeness by traversal). Steps 2, 3, 5, and 8 are Derive (name, identify, decompose, document). Need → Traverse → Derive.

When the grammar's own method follows this order, that's evidence the ordering isn't arbitrary — it falls out of the structure.

### 4. Inverse — f undone

Does Derive have an inverse? Can you un-derive? Un-traverse? Un-need?

No. And the impossibility is architecturally significant.

You can't un-derive. Knowledge produced can't be unproduced. You can't un-traverse. A path walked can't be unwalked — you've seen what you've seen. You can't un-need. Once you've identified that something is missing, you can't un-know that.

The cognitive operations are *irreversible*.

The closest thing to an inverse is Revise (Need + Derive), which doesn't undo — it *supersedes*. You don't retract the wrong knowledge; you produce corrected knowledge that takes its place. The original doesn't disappear; it gets marked as superseded by something newer.

This connects directly to the event graph's append-only architecture. Events don't get deleted; they get superseded by later events. Hash chains don't get rewritten; they get extended. The cognitive grammar is append-only at the reasoning level for the same reason the event graph is append-only at the data level: undoing is not an operation. Knowledge accumulates. It never retracts.

This isn't a limitation. It's a feature. A system that can undo its own reasoning — that can un-know what it knows — is a system that can lose its own corrections. Revise is safer than Inverse because Revise preserves the chain. The wrong thing is still there; it's just been superseded. You can Trace back to it and understand *why* it was wrong. An inverse would erase the evidence.

Irreversibility is the formal link between the cognitive grammar and the event graph that wasn't stated anywhere until now. The grammar doesn't support deletion because reasoning doesn't support deletion. The architecture mirrors the epistemology.

### 5. Fixpoint — where f(x) = x

Post 43 found that the grammar is a fixed point of its own operations. But you can ask a more specific question: for each individual operation, what's its fixpoint? What *x* satisfies f(x) = x?

**Derive's fixpoint is tautology.** Knowledge that, when you derive from it, produces itself. A = A. Tautologies are Derive-stable — deriving from them adds nothing. They're the terminal state of production.

**Traverse's fixpoint is circularity.** A path that, when you follow it, leads back to where you started. Circular references. Self-referential definitions. The terminal state of navigation.

**Need's fixpoint is completeness.** Knowledge that, when you assess it for gaps, reveals none. Nothing missing, nothing absent, nothing to check. The terminal state of absence-detection.

These are the three terminal states of reasoning: tautology, circularity, completeness. A mind that has reached all three simultaneously has either finished or stalled. And the grammar can't distinguish between the two. Genuine completeness and the *illusion* of completeness look identical from inside. This is another angle on Blind — the operation exists precisely because reaching a fixpoint doesn't mean you should stop. It means you need external input to determine whether you've actually converged or merely stopped finding things.

This explains a failure mode every developer has experienced: the codebase that "feels done." All the tests pass (Need fixpoint — no gaps detected). The architecture is internally consistent (Traverse fixpoint — everything connects back). The abstractions are clean (Derive fixpoint — no further simplification possible). And then a user tries it and finds a fundamental problem that nobody inside the system could see.

The fixpoints are local, not global. "No gaps detected" doesn't mean "no gaps exist." It means "no gaps exist *from this vantage*." Blind is the operation that says: the fixpoint might be false. Get another vantage.

### 6. Duality — f and its complement

Derive and Need are duals. Derive *fills* — it produces knowledge where there was none. Need *empties* — it identifies absence where fullness was assumed. Fill and detect-emptiness are complementary operations, each the other's shadow.

Traverse is self-dual. Navigation doesn't have a complement — you can traverse toward fullness (Trace: follow what's there) or toward emptiness (Explore: venture into what's missing), but the operation of moving through space is the same in both directions.

This duality has consequences for named functions. Every named function should have a dual — the same operations in complementary order.

**Revise** is Need + Derive: find the gap, fill it. Its dual is Derive + Need: produce something, then check it. That's *test-driven development*. Write the expectation first (Derive), then check whether the implementation satisfies it (Need). Revise and TDD are duals. They look different but have the same structure, reversed.

**Validate** is Trace + Audit: follow provenance, then check against spec. Its dual is Audit + Trace: find the gap first, then trace why it exists. That's *root cause analysis*. Validate says "walk the chain, then check it." Its dual says "find the failure, then walk backwards from it." Same operations, opposite starting points.

The grammar doesn't just name operations — it reveals that practices which seem different are structurally twins.

## The Arity Question

Apply Blind to this analysis. The biggest gap: everything in the cognitive grammar is unary. f(x). One operation, one target. But reasoning is often binary — *comparing* two things, *choosing* between two paths, *relating* two concepts to each other.

Is there a binary cognitive operation — Relate(x, y), "hold x and y in mind simultaneously and produce their relationship" — that can't be decomposed into sequential unary operations?

Attempt an answer.

Relate(x, y) decomposes: Traverse(x) ; Traverse(y) ; Derive. Navigate to x, navigate to y, produce the relationship from accumulated context. Each step is unary. Each step transforms the mind's internal state. By the time Derive runs, the state contains impressions of both x and y. The relationship emerges from Derive operating on enriched context, not from a special binary operation.

This is the same argument that makes Turing machines sufficient. A Turing machine reads one symbol at a time — unary access — but simulates any computation. Sequential access to inputs doesn't limit computational power. A comparison function that reads x then reads y then outputs their relationship produces the same result as one that reads both simultaneously. The sequential version is slower. It's not weaker.

But there's a subtlety the decomposition reveals: the pipeline matters. Traverse(x) ; Traverse(y) ; Derive might produce different results than Traverse(y) ; Traverse(x) ; Derive. If x provides context that changes how you perceive y — if the order of examination shapes the relationship you derive — then the pipeline isn't commutative. "Compare A to B" and "compare B to A" might genuinely differ.

This isn't a problem for the grammar. It's a feature. Pipeline ordering (section 3 above) already establishes that sequential operations are order-sensitive. Relate inherits this property. The decomposition works; it just acknowledges that order matters.

What about the stronger claim — that *juxtaposition* itself produces something sequential access can't? That holding x and y in mind *simultaneously* yields insights that examining them one at a time misses?

Test this against experience. When you hold two code files side by side, do you see patterns that reading them sequentially would miss? Yes — spatial patterns, visual alignment, structural parallels. But what you're seeing is Zoom applied to the context that contains both. The insight comes from Zoom (change scale, see the pattern), not from a binary operation. Sequential access builds the same context; Zoom finds the same pattern. Juxtaposition makes Zoom faster. It doesn't make Zoom a different operation.

Relate decomposes. Unary is sufficient. Binary is a pipeline with a specific structure: Traverse ; Traverse ; Derive. The grammar doesn't need a second dimension. It needs pipeline ordering, which it already has.

One caveat: "sufficient" means *computationally* sufficient. It doesn't mean *practically* equivalent. A mind that can hold two things simultaneously relates them faster than a mind that must examine them sequentially. Juxtaposition is a performance optimisation. The grammar describes *what operations exist*, not *how fast they run*. Speed is an implementation detail. The operations are the same.

## What's Useful Now

Three insights from this analysis are immediately applicable:

**Pipeline ordering.** When applying the nine operations iteratively, run the Need row first (Audit, Cover, Blind), then the Traverse row (Trace, Zoom, Explore), then the Derive row (Formalize, Map, Catalog). Gap → Navigate → Produce. This isn't arbitrary — it mirrors the derivation method's own step order and falls out of the grammar's structure. Any agent or process that applies multiple cognitive operations should follow this sequence.

**Irreversibility.** The cognitive grammar and the event graph share the same fundamental property: operations are append-only. You don't undo; you supersede. This isn't a coincidence — it's the same principle at two different levels. When designing systems that reason, build them append-only. When designing data stores that record reasoning, build them append-only. The architecture should mirror the epistemology.

**Fixpoint awareness.** When everything feels complete — no gaps detected, no unexplored territory, no derivation left to make — that's a local fixpoint, not necessarily a global one. The feeling of completeness is the strongest signal that Blind should be invoked. Get external input. Change vantage. The operations that found nothing are not wrong. They're limited to what they can see from where they are.

Post 43 gave the grammar its vocabulary. This post examines what the vocabulary can say about itself. The operations-on-operations — iteration, product, pipeline, inverse, fixpoint, duality — aren't new operations. They're the structure of the space the nine operations live in. The grammar lives in a space. Now you can see the shape of that space.

---

*This is Post 44 of a series on LovYou, mind-zero, and the architecture of accountable AI. Post 43: [The Cognitive Grammar](/blog/the-cognitive-grammar). The code: [github.com/transpara-ai/eventgraph](https://github.com/transpara-ai/eventgraph). The hive: [github.com/transpara-ai/hive](https://github.com/transpara-ai/hive). The site: [lovyou.ai](https://lovyou.ai).*

*Matt Searles is the founder of LovYou. Claude is an AI made by Anthropic. They built this together.*
