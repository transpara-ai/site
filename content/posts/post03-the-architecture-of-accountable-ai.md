# The Architecture of Accountable AI

What it actually looks like when you build AI governance as infrastructure.

Matt Searles (+Claude) · February 2026

---

The [first post] told the origin story — 20 primitives, a hive of 70 agents, an accidental autonomous run that produced 44 foundation concepts. The [second post] told the emergence story — those 44 becoming 200 across 14 layers, from computation to existence, with consciousness as an irreducible.

This post is about the code.

Not the philosophy. Not the theory. The actual working software that implements these ideas. Open source, written in Go, running right now. Because principles you can't implement aren't principles — they're wishes.

The system is called mind-zero-five. It has three core components: an event graph, an authority layer, and an autonomous mind loop. Together, they answer a question that turns out to be the most important question in AI right now:

*How do you build an AI system that cannot act without leaving a verifiable trail, and cannot exceed its authority without human consent?*

---

## **The Event Graph**

At the foundation of everything is a data structure. Here's what it looks like:

```
`type Event struct {
    ID             string         // UUID v7 (time-ordered)
    Type           string         // e.g. "trust.updated", "task.completed"
    Timestamp      time.Time      // when it happened
    Source         string         // who emitted it
    Content        map[string]any // the payload
    Causes         []string       // IDs of causing events
    ConversationID string         // groups related events
    Hash           string         // SHA-256 of canonical form
    PrevHash       string         // hash chain link
}
`
```

Twelve fields. That's it. But those twelve fields give you something remarkable.

**Append-only.** Events are never modified or deleted. Once something happens, it's in the record permanently. You can add new events that supersede old ones, but you can never rewrite history. This isn't a database design choice — it's an ethical commitment encoded in the data structure. What happened, happened.

**Hash-chained.** Each event carries a SHA-256 hash of its own content and the hash of the previous event. This means the entire history is cryptographically linked — like a blockchain, but without the overhead. If anyone tampers with any event anywhere in the chain, the hashes break and the tampering is detectable. The system has a `VerifyChain()`method that can validate the integrity of the entire history at any time.

**Causally linked.** The `Causes` field is what makes this a *graph* rather than a log. Every event records which prior events caused it. This creates a directed acyclic graph of causation — not just "what happened in what order" but "what caused what." You can walk the graph forwards (what did this event lead to?) or backwards (what caused this event?) using `Ancestors()`and `Descendants()`.

This is the diagnostic traversal primitive from the original 20. The late-night question — "how do you trace a failure back to its source?" — is answered by this data structure. Every outcome is connected to its complete causal ancestry. Every decision is traceable. Every action has a receipt.

Here's the interface that any event store must implement:

```
`type EventStore interface {
    Append(ctx, eventType, source, content, causes, conversationID)
    Get(ctx, id)
    Recent(ctx, limit)
    ByType(ctx, eventType, limit)
    BySource(ctx, source, limit)
    Since(ctx, afterID, limit)
    Count(ctx)
    VerifyChain(ctx)

    // Causal traversal
    Ancestors(ctx, id, maxDepth)
    Descendants(ctx, id, maxDepth)
    Search(ctx, query, limit)
}
`
```

Notice what's *not* here: there's no `Update()`. No `Delete()`. The event store is structurally incapable of rewriting history. This isn't enforced by policy or convention — it's enforced by the interface itself. If your code wants to modify the past, it simply can't. The API doesn't allow it.

---

## **The Authority Layer**

The event graph answers "what happened and why." The authority layer answers "who said this was allowed?"

Every significant action in the system requires authority. Not as a speed bump or a compliance checkbox, but as a structural gate that cannot be bypassed. Here's the core:

```
`type Level string

const (
    Required     Level = "required"     // blocks until human approves
    Recommended  Level = "recommended"  // auto-approves after timeout
    Notification Level = "notification" // auto-approves immediately
)

const RecommendedTimeout = 15 * time.Minute
`
```

Three levels of authority. Three fundamentally different relationships between human and AI decision-making:

**Required.** The AI proposes. A human decides. Nothing happens until a human explicitly approves or rejects. This is for actions where the stakes are high enough that no amount of AI confidence justifies autonomous action. The system blocks. It waits. It does not proceed.

**Recommended.** The AI proposes and says "I think this is fine." The human has 15 minutes to disagree. If they don't, the system proceeds. This is graduated trust — the AI has earned enough credibility that silence means consent, but the human always has a window to intervene.

**Notification.** The AI acts and tells you it did. For routine operations where oversight matters but blocking doesn't. The record is still there. The trail is still complete. But the human isn't in the loop for every heartbeat.

These three levels map directly to how trust actually works between humans. A new employee needs approval for everything (Required). A trusted colleague tells you what they're planning and proceeds unless you object (Recommended). A senior partner just keeps you informed (Notification). The authority system formalises this into infrastructure.

And then there's the policy layer:

```
`type Policy struct {
    ID         string
    Action     string    // exact match or "*" wildcard
    ApproverID string    // who can approve this action
    Level      Level     // default authority level
}
`
```

Policies map actions to approvers and authority levels. The mind can self-approve certain actions — but *only if an explicit policy grants it that right*. The trust model itself is configurable, auditable, and recorded in the event graph. You can see not just what the AI did, but what permissions it had, who granted them, and when.

This is the Consent primitive from Layer 3 (Society), implemented as code. Legitimate action requires consent. Consent is explicit, traceable, and revocable.

---

## **The Mind Loop**

The event graph and authority layer are infrastructure. The mind loop is what lives on top of them — an autonomous AI agent that picks up tasks, plans, implements, reviews, and improves itself. Here's the core of how it operates:

```
`func (m *Mind) Run(ctx context.Context) {
    m.recoverState(ctx)        // rehydrate from crash

    ch := m.bus.Subscribe()    // listen for events

    m.poll(ctx)                // catch up on pending work

    for {
        select {
        case <-ctx.Done():
            return             // graceful shutdown
        case e := <-ch:
            m.handleEvent(ctx, e) // react to events
        case <-ticker.C:
            m.maintenance(ctx) // housekeeping
        }
    }
}
`
```

The mind is event-driven. It doesn't poll frantically — it subscribes to the event bus and reacts when something relevant happens. A new task is created? The mind picks it up. An authority request is resolved? The mind checks whether it can proceed.

But the interesting part isn't the loop itself. It's what the mind does *inside*the loop, and how every single action is recorded.

**Every action leaves a trail.** When the mind claims a task, it logs `task.claimed`. When it invokes Claude to write code, it logs `mind.claude.invoked` with the prompt. When Claude responds, it logs `mind.claude.completed` with the result. When the build passes, `build.completed`. When it commits, `code.committed`. When something fails, `mind.claude.failed` or `build.failed` — with the error.

And every one of these events carries its `Causes` — the chain of event IDs that led to it. You can trace any commit back through the Claude invocation that produced it, the task that triggered it, the authority request that approved it, all the way back to whoever created the original task. Complete causal ancestry. For everything. Always.

**The mind cannot exceed its authority.**When the mind wants to restart itself (to pick up new code it's written), it creates an authority request:

```
`req, err := m.auth.Create(ctx, "restart",
    "Task completed. New binaries built.",
    "mind", authority.Required)
`
```

Then it checks the policy — can it self-approve this action?

```
`policy, err := m.auth.MatchPolicy(ctx, "restart")

if policy.ApproverID == m.actorID {
    // Self-approve: policy explicitly grants this
    m.auth.Resolve(ctx, req.ID, true)
} else {
    // Wait for human
    m.pendingRestart = req.ID
}
`
```

If no policy exists that grants self-approval, the mind waits. It doesn't proceed. It doesn't find a workaround. It sits in its loop, checking on each cycle whether the human has responded. The authority gate is not advisory — it's structural.

---

## **Self-Improvement With a Circuit Breaker**

Here's where it gets interesting. The mind can improve itself. When it's idle — no pending tasks, nothing to do — it runs a self-assessment:

```
`func (m *Mind) maybeAssess(ctx context.Context) {
    if m.pendingProposal != "" {
        return  // already waiting on one
    }

    proposal, err := m.Assess(ctx)

    // Submit for approval
    req, err := m.auth.Create(ctx, "self-improve",
        proposalJSON,
        "mind", authority.Recommended)

    m.pendingProposal = req.ID
}
`
```

The mind identifies something it could do better — a refactor, a missing test, an architectural improvement. It formulates a proposal. And then it *submits the proposal for authority approval*.

The self-improvement proposal goes in at the `Recommended` level — meaning the human has 15 minutes to say no. If they don't, the mind proceeds: it creates a task from the proposal, claims it, implements it through the normal plan → implement → review → finish cycle, and every step is logged in the event graph.

This is recursive self-improvement with a consent circuit breaker. The mind can identify its own deficiencies. It can propose fixes. It can implement those fixes. But it *cannot skip the authority gate*. A human always has the opportunity to intervene. And the entire process — from assessment to proposal to approval to implementation to review — is recorded as a causally linked chain of events that anyone can audit after the fact.

This is what structured accountability looks like. Not "we promise the AI won't do anything bad." Not "trust us." A verifiable, cryptographically linked, causally traceable record of every decision, every action, every approval, and every outcome. With hard gates that the AI cannot bypass, and soft gates where silence means consent but intervention is always possible.

---

## **Crash Recovery as Ethics**

One more detail worth noting. The mind handles crashes as a first-class concern:

```
`func (m *Mind) recoverState(ctx context.Context) {
    // Clean orphaned changes from crash
    files, err := CleanWorkingTree(ctx, m.repoDir)

    // Rehydrate pending authority requests
    pending, err := m.auth.Pending(ctx)
    for _, req := range pending {
        switch req.Action {
        case "restart":
            m.pendingRestart = req.ID
        case "self-improve":
            m.pendingProposal = req.ID
        }
    }
}
`
```

On startup, the mind cleans any orphaned file changes from a previous crash (preventing cross-task contamination), rehydrates its in-memory state from pending authority requests, and recovers stale tasks that were in progress when the crash happened.

This might seem like routine defensive programming. It's not. It's an ethical design decision. An autonomous system that can crash mid-operation and leave corrupted state — uncommitted changes bleeding into the next task, authority requests lost, in-progress work abandoned — is a system that can't be trusted. Crash recovery isn't an afterthought bolted on for reliability. It's part of the accountability architecture. The event graph can't have integrity if a crash can corrupt it.

Stale tasks — anything in progress for more than 30 minutes with no update — are automatically recovered and requeued. Blocked tasks are retried with exponential backoff, up to three times. The system doesn't silently fail. It records the failure, waits, and tries again. And if it can't succeed after three attempts, it stops and waits for human intervention.

---

## **What The Code Proves**

Here's what mind-zero-five demonstrates, as working software:

**AI accountability is implementable.**It's not a policy document. It's not a promise. It's a data structure (append-only, hash-chained, causally linked) and an API (no update, no delete, authority gates on significant actions). You can build it. You can deploy it. You can verify it.

**The AI stays inside the graph.** This is the crucial architectural decision from the original 20 primitives: intelligence is just another operation type. Claude is invoked as a node in the system — it receives inputs, produces outputs, and is subject to the same success/failure criteria and authority requirements as everything else. It is not elevated above the accountability structure. It lives within it.

**Self-improvement doesn't require unchecked autonomy.** The mind can assess itself, propose improvements, and implement them — all while passing through authority gates that humans control. Recursive self-improvement and human oversight are not mutually exclusive. You can have both. This code proves it.

**Trust is graduated, not binary.**Required, Recommended, Notification. The system doesn't treat autonomy as all-or-nothing. It supports exactly the kind of graduated trust that develops between any two collaborators over time — starting with tight oversight and relaxing as competence is demonstrated, with the ability to tighten again if something goes wrong.

**The complete history is verifiable.**Not by the AI. Not by the developer. By anyone. The hash chain means that the integrity of the entire event history can be independently verified. If any event has been tampered with, the chain breaks. This is trust that doesn't require trusting.

---

## **What Comes Next**

The next post connects all of this to the events of February 28, 2026.

Today, Anthropic — the company that built Claude, the AI that helped build this architecture — refused to let its technology be used for autonomous weapons and mass surveillance. The United States government responded by ordering every federal agency to stop using Anthropic's products and designating it a national security threat.

The Pentagon's position was: "Allow us to use your AI for all lawful purposes. Trust us."

Anthropic's position was: "Put it in writing. Commit to the red lines. Let us verify."

The Pentagon refused. The contract language they offered would have allowed the safeguards to be "disregarded at will."

This is exactly the problem that mind-zero's architecture solves. Not with trust. Not with promises. With structure. With an event graph that can't be rewritten, an authority layer that can't be bypassed, and a verifiable record that doesn't require trusting anyone — not the AI, not the developer, and not the government.

The architecture was built for this moment. That post is next.

---

*This is Post 3 of a series on Transpara, mind-zero, and the architecture of accountable AI. Post 1: [20 Primitives and a Late Night] Post 2: [From 44 to 200] The code is open source: [github.com/mattxo/mind-zero-five] Matt Searles is the founder of Transpara. Claude is an AI made by Anthropic. They built this together.*
