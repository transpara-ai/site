# Cognitive Grammar

The grammar of reasoning itself. Three base operations, nine compositions by self-application, three modifiers, six named functions. Derived in [Post 43](/blog/post43-the-cognitive-grammar).

The cognitive grammar is not the fourteenth domain grammar. It is the grammar that *produces* grammars. Every derivation in the framework uses Derive, Traverse, and Need. The thirteen layer grammars are outputs of the cognitive grammar.

## Three Base Operations

A mind relates to knowledge in exactly three ways: it produces knowledge that doesn't exist, navigates knowledge that does exist, or detects that knowledge is absent.

| Operation | Definition |
|-----------|-----------|
| **Derive** | Produce new knowledge from existing knowledge. Takes premises and produces conclusions. Takes examples and produces patterns. Takes dimensions and fills the matrix. |
| **Traverse** | Navigate knowledge space. Move from one thing to another. Follow connections. Zoom in to examine detail. Zoom out to see landscape. |
| **Need** | Assess knowledge for absence. What's missing? What should be here that isn't? What have I not considered? |

Each is irreducible — you cannot compose any from the other two. Derive without Traverse produces blind output. Traverse without Derive produces sterile navigation. Need without the others produces helplessness.

## Self-Application Matrix

The cognitive grammar's domain is its own operations. Self-application — applying each operation to each operation — produces nine distinct compositions:

```
              Derive        Traverse      Need
            +-------------+-------------+-------------+
 Derive(x)  | Formalize   | Map         | Catalog     |
            +-------------+-------------+-------------+
 Traverse(x)| Trace       | Zoom        | Explore     |
            +-------------+-------------+-------------+
 Need(x)    | Audit       | Cover       | Blind       |
            +-------------+-------------+-------------+
```

Rows share a verb: Derive's row produces structure, Traverse's row navigates, Need's row detects absence. Columns share a subject: the Derive column concerns method, the Traverse column concerns space, the Need column concerns gaps. The grammar has no preferred direction.

## Nine Operations

### Derive applied to each

| Operation | Composition | Definition |
|-----------|------------|-----------|
| **Formalize** | Derive(Derive) | Derive the method of derivation itself. Extract rules from practice. Write down how you do what you do. |
| **Map** | Derive(Traverse) | Derive the navigation structure of a knowledge space. Produce the map before exploring the territory. |
| **Catalog** | Derive(Need) | Derive all the types of absence. Produce a taxonomy of what can be missing: unknown, uncertain, incomplete, wrong, outdated. |

### Traverse applied to each

| Operation | Composition | Definition |
|-----------|------------|-----------|
| **Trace** | Traverse(Derive) | Walk through a derivation chain. Follow provenance. How was this knowledge produced? What were the premises? |
| **Zoom** | Traverse(Traverse) | Navigate navigation itself. Change scale. Move between "this function has a bug" and "this architecture has a flaw." |
| **Explore** | Traverse(Need) | Navigate into what's missing. Venture into gaps. Not identifying the gap — moving into it to discover what's there. |

### Need applied to each

| Operation | Composition | Definition |
|-----------|------------|-----------|
| **Audit** | Need(Derive) | Identify missing derivations. Compare what exists against what should exist. Check an implementation against a specification. |
| **Cover** | Need(Traverse) | Identify unexplored territory. What parts of the space haven't been looked at? The unread file. The unconsidered dimension. |
| **Blind** | Need(Need) | Identify unrecognised gaps. The unknown unknowns. Structurally impossible to perform alone — requires external input. |

## Completeness

The nine operations are a fixed point under self-application. Applying any of the nine to any of the nine collapses to one of the nine with a specific argument. No new operation emerges. The grammar can reason about itself and finds nothing missing.

**Candidates tested and killed:** Decide = Need + Derive. Imagine = Explore + Derive. Remember = Traverse(past). Attend = Zoom + Traverse. Doubt = Need applied to meta-knowledge. No candidates survive.

## Modifiers (3)

Each operation has three aspects — output, input, process — each admitting one modifier.

| Modifier | Aspect | Effect |
|----------|--------|--------|
| **Tentative** | Output | Result is provisional, marked for verification. Tentative Derive produces a hypothesis, not a conclusion. |
| **Exhaustive** | Input | Must cover the complete space, not sample. Exhaustive Cover checks everything. Exhaustive Audit compares every operation. |
| **Bounded** | Process | Limited in scope, depth, or time. Bounded Explore ventures into the gap, but not forever. The pragmatic constraint on infinite operations. |

## Named Functions (6)

Compositions that recur across domains often enough to name.

| Function | Composition | Definition |
|----------|------------|-----------|
| **Revise** | Need + Derive | Identify wrong knowledge, produce corrected knowledge. Every bug fix. Every scientific correction. |
| **Hypothesize** | Explore + Tentative Derive | Venture into a gap, produce a candidate explanation. Not a conclusion — something testable. |
| **Validate** | Trace + Audit | Follow provenance, then check against what should exist. Code review. Peer review. Due diligence. |
| **Orient** | Map + Zoom | Produce a navigation structure, then set your scale. What you do when joining a new project. |
| **Learn** | Explore + Derive + Need | Discover, produce, verify. The complete knowledge-acquisition cycle. If Need finds gaps, the cycle repeats. |
| **Calibrate** | Cover + Blind + Zoom | Check coverage at multiple scales, including for unknown unknowns. The most expensive cognitive operation and the most neglected. |

## Relationship to Other Grammars

The cognitive grammar sits above the domain grammars in the hierarchy:

- **Cognitive Grammar** — the method that produces grammars (this page)
- **Graph Grammar** — 15 base operations on the event graph, derived by the method
- **Layer Grammars** — 13 domain-specific compositions of the graph grammar
