# Higher-Order Operations

The operations *on* operations. If the cognitive grammar is a set of functions, these are the algebra of that set. Derived in [Post 44](/blog/post44-higher-order-operations).

Post 43 proved the cognitive grammar is a fixed point under composition — f(g) produces no new operations beyond the nine. But composition is not the only thing you can do with functions. Six operations on operations reveal the structure of the space the nine operations inhabit.

## Six Operations on Operations

### 1. Iteration — f applied repeatedly

f(f(f(x))). What happens when you apply the same operation at increasing depth?

| Depth | Formalize | Blind |
|-------|-----------|-------|
| 1 | Write down the method | What don't I know I'm missing? |
| 2 | Write down the method of writing down methods | What don't I know about what I don't know I'm missing? |
| 3 | Write down the method of... (converges) | Meta-unknowns (converges) |

The grammar has **nine operations x n depth levels**. In practice, iterative convergence stabilises in 3-4 passes. Beyond that, you're doing the same thing with longer sentences.

**For agent design:** "How many levels of meta-reflection should this agent perform?" is a real parameter. Depth 0: just do it. Depth 1: think about how you're doing it. Depth 2: think about how you're thinking. The convergence result says depth 3-4 is the practical ceiling.

### 2. Product — f and g applied independently

Composition is serial: f(g(x)). Product is parallel: apply f and g to the same target simultaneously, producing a pair of results.

Audit(spec) tells you what derivations are missing. Cover(spec) tells you what territory is unexplored. Audit x Cover gives you *both at the same time*. Independent assessments that together form a richer picture.

**The deepest consequence:** Agent1.Need x Agent2.Need produces gap-assessments that neither agent could produce alone. This is why the hive needs multiple agents. Not more compute — more *perspectives*. Blind is "structurally impossible to perform alone." Product is why. Blind requires the product of at least two Need operations from different positions.

### 3. Pipeline — f then g, with state

In f(g(x)), g's output becomes f's input. In a pipeline f ; g, f transforms x, then g encounters the *transformed x*. The thing changes between operations.

**Optimal pipeline ordering:** The grammar suggests Need first, Traverse second, Derive third.

| Phase | Row | Operations | Purpose |
|-------|-----|-----------|---------|
| 1 | Need | Audit, Cover, Blind | Detect absence |
| 2 | Traverse | Trace, Zoom, Explore | Navigate to gaps |
| 3 | Derive | Formalize, Map, Catalog | Fill gaps |

Gap -> Navigate -> Produce. This mirrors the derivation method's own step order. When the grammar's own method follows this order, the ordering isn't arbitrary — it falls out of the structure.

### 4. Inverse — f undone

The cognitive operations are **irreversible**.

You can't un-derive (knowledge produced can't be unproduced). You can't un-traverse (a path walked can't be unwalked). You can't un-need (once you've identified absence, you can't un-know it).

The closest thing to an inverse is **Revise** (Need + Derive), which doesn't undo — it *supersedes*. The original doesn't disappear; it gets marked as superseded by something newer.

**This connects directly to the event graph's append-only architecture.** Events don't get deleted; they get superseded. Hash chains don't get rewritten; they get extended. The cognitive grammar is append-only at the reasoning level for the same reason the event graph is append-only at the data level. Irreversibility is the formal link between the cognitive grammar and the event graph.

### 5. Fixpoint — where f(x) = x

For each operation, what x satisfies f(x) = x?

| Operation | Fixpoint | Meaning |
|-----------|----------|---------|
| **Derive** | Tautology | Knowledge that, when derived from, produces itself. A = A. |
| **Traverse** | Circularity | A path that leads back to where you started. |
| **Need** | Completeness | Knowledge where no gaps are detected. |

These are the three terminal states of reasoning: **tautology, circularity, completeness**. A mind that has reached all three simultaneously has either finished or stalled. The grammar can't distinguish between the two from inside.

**This explains "feels done" failure modes.** All tests pass (Need fixpoint). Architecture is consistent (Traverse fixpoint). Abstractions are clean (Derive fixpoint). Then a user finds a fundamental problem nobody inside the system could see. The fixpoints are **local, not global**. "No gaps detected" doesn't mean "no gaps exist." Blind is the operation that says: the fixpoint might be false. Get another vantage.

### 6. Duality — f and its complement

Derive and Need are duals. Derive *fills* (produces knowledge where there was none). Need *empties* (identifies absence where fullness was assumed). Traverse is self-dual — navigation doesn't have a complement.

Duality reveals that practices which seem different are structurally twins:

| Function | Composition | Its Dual | Dual Composition | Known As |
|----------|------------|----------|-----------------|----------|
| **Revise** | Need + Derive | Derive + Need | Produce, then check | Test-Driven Development |
| **Validate** | Trace + Audit | Audit + Trace | Find failure, then trace why | Root Cause Analysis |

Same operations, opposite starting points. The grammar doesn't just name operations — it reveals structural relationships between practices.

## The Arity Question

Apply Blind to this analysis. Everything in the cognitive grammar is unary — f(x). But reasoning is often binary: comparing, choosing, relating.

Does Relate(x, y) decompose?

**Yes.** Relate(x, y) = Traverse(x) ; Traverse(y) ; Derive. Navigate to x, navigate to y, produce the relationship from enriched context. Each step is unary. By the time Derive runs, the state contains both. The relationship emerges from Derive operating on enriched context, not from a binary operation.

**Caveat:** Pipeline ordering matters. Traverse(x) ; Traverse(y) ; Derive might differ from Traverse(y) ; Traverse(x) ; Derive — if x provides context that changes how you perceive y. "Compare A to B" and "compare B to A" can genuinely differ. This isn't a bug; it's pipeline ordering (section 3) doing what pipeline ordering does.

Unary is computationally sufficient. Binary is a performance optimisation. The grammar describes what operations exist, not how fast they run.

## What's Useful Now

**Pipeline ordering.** Need -> Traverse -> Derive. Any agent or process applying multiple cognitive operations should follow this sequence.

**Irreversibility.** The cognitive grammar and the event graph share append-only semantics. Architecture should mirror epistemology.

**Fixpoint awareness.** When everything feels complete, invoke Blind. The feeling of completeness is the strongest signal that external input is needed.

## Relationship to Other Grammars

The higher-order operations are not new operations. They are the structure of the space the nine cognitive operations live in:

- **Cognitive Grammar** — nine operations from three bases
- **Higher-Order Operations** — the algebra of those nine (this page)
- **Graph Grammar** — 15 base operations, derived by the method
- **Layer Grammars** — 13 domain compositions of the graph grammar
