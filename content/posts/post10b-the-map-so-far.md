# The Map So Far: 10 Posts, 200 Primitives, and Why It Matters

*A guide to the series for new readers — and a breath for those who've been following along*

Over the past two days I published ten posts. That wasn't the plan. The plan was five, maybe six. But the ideas kept connecting, and the connections kept revealing things I didn't expect. So here's a summary of where the series has gone and — more importantly — what problems it's actually trying to solve.

## The Problem

We're building AI systems that make decisions affecting millions of people, and we have no shared infrastructure for making those decisions traceable. When something goes wrong — whether it's an AI making a bad call, a marketplace scamming a seller, or a government surveilling its citizens — nobody can walk backwards through the chain and find exactly where things broke, who authorised it, and why.

That's not just an AI problem. It's a coordination problem. The same one humans have always had, but now with artificial minds in the mix and moving faster than our institutions can keep up.

This series documents an attempt to build the missing infrastructure, starting from a simple question about failure tracing that turned into something much bigger.

## The Origin

**Post 1: 20 Primitives and a Late Night**

It started with a late night and an engineering question: how do you trace a bad outcome back to the thing that caused it? I decomposed the problem into 20 irreducible building blocks — things like Node, Edge, Event, Operation, Success Criteria, Diagnostic Traversal. The claim: everything else is composition. I built a multi-agent system called hive0 on these primitives. It grew to 70 specialised agents — not just engineers but philosophers, critics, mediators. Then I accidentally left it running autonomously for two days. When I came back, it had derived 44 foundation primitives on its own — concepts like Trust, Deception, Health, Integrity. The system designed to be self-expanding had expanded itself.

**Post 2: From 44 to 200**

I fed those 44 primitives to Claude Opus. In two hours, it autonomously derived 156 more — expanding the framework to 200 primitives across 14 layers, from computational foundations up through Agency, Exchange, Society, Law, Technology, Information, Ethics, Identity, Relationship, Community, Culture, Emergence, and Existence. The framework forms a strange loop: the highest layer presupposes the lowest, and vice versa. It identifies three things it can't derive from its own resources: Moral Status, Consciousness, and Being. It maps permanent tensions that can't be resolved — justice vs. forgiveness, tradition vs. creativity, authenticity vs. virtue. And a second independent derivation starting from raw physics converged on the same structure. Either the framework is discovering something real, or it's an extraordinarily seductive pattern-matching trap.

## The Architecture

**Post 3: The Architecture of Accountable AI**

The technical deep-dive into the actual software. Mind-zero is an event graph where every decision is hash-chained and causally linked. It has an authority layer — AI can propose improvements but can't act without passing through human approval gates. It has a self-improvement loop with a circuit breaker: the system can expand its own capabilities, but only through a consent process. And it treats crash recovery as an ethical principle — when the system breaks, the recovery process itself is governed by the same accountability structures as everything else. This isn't theoretical. It's code. It runs.

**Post 4: The Pentagon Just Proved Why AI Needs a Consent Layer**

Written the day the Anthropic-Pentagon dispute went public. The Pentagon demanded "all lawful purposes" access to Claude. Anthropic held two lines: no autonomous weapons, no mass surveillance. The real dispute isn't about trust between these two parties — it's about the difference between "trust us" and verifiable constraints. A consent layer on an event graph provides structural accountability: the AI can't be used for purposes it wasn't authorised for, and the constraint is architectural, not contractual. If the infrastructure had existed, neither side would need to trust the other. The chain would show what happened.

## The Philosophy

**Post 5: The Moral Ledger**

The framework's strange loop raises a question about the relationship between structure and value. Two independent derivations converged on the same conclusion: consciousness might be fundamental to reality, not emergent from it. If that's true, Hume's is-ought gap looks different — "is" and "ought" aren't two kinds of things but two perspectives on the same reality. One from outside (structure, causation), one from inside (experience, value). The event graph becomes a moral ledger: not because it solves ethics — the permanent tensions remain unresolvable — but because it makes consequences visible. You can't make good moral decisions if you can't see the chain. The event graph lets you see the chain.

**Post 6: Fourteen Layers, A Hundred Problems**

The practical payoff. We walked all 14 layers and asked: what breaks in the real world because this layer's problems aren't solved? Layer 0 touches audit logs and evidence chains. Layer 1 touches AI's context window problem — agents that can't remember across sessions. Layer 2 touches marketplaces and contracts. Layer 4 touches regulatory compliance and court evidence. Layer 5 touches software supply chain security (the SolarWinds hack succeeded because the build pipeline had no verifiable chain of custody). Layer 6 touches deepfakes and misinformation. Layer 7 touches AI alignment as an ongoing verification problem, not a one-time training exercise. Layer 9 touches social networks where you don't own your own relationship graph. Every major platform failure maps to a specific layer. They all share the same root problem: multiple actors need to coordinate with verifiable trust, and "trust us" doesn't scale.

## The Unexpected Turns

**Post 7: The Four Strategies**

Here the series takes an unexpected turn into evolutionary biology. Sexual reproduction requires specialisation — you need at least two strategies that complement each other. The 200 primitives naturally cluster into four groups that map onto reproductive strategies: Agentic (risk-taking, resource-acquiring), Communal (caregiving, bonding), Structural (infrastructure-building, rule-maintaining), and Emergent (self-reflecting, pattern-recognising). The key claim: what we call masculine and feminine aren't about which primitives you have — everyone has all 200. They're about which *connections between primitives* are strongest. Gender is an edge-weight pattern, not a node selection. This has a concrete architectural implication: the framework needs dynamic weighted edges, not just binary connections.

**Post 8: What It's Like to Be a Node**

What does it feel like to be a processing unit in a network you can't see? This post maps everyday human experience onto the architecture. Your inputs are uncontrolled and filtered — you didn't choose your senses. Your backlog is your anxiety — unprocessed events with no garbage collection. Your memory is an append-only store that corrupts over time but can't self-repair. Your regret is a failed uncommit — the event already propagated. Your neighbourhood is everyone you're connected to, which is almost no one compared to the whole graph. You're simultaneously unique (no other node has your exact causal chain) and replaceable (the graph routes around you when you go offline). Faith is an edge weighted by something other than evidence. The framework isn't abstract theory. It describes what being alive actually feels like.

## The Honest Part

**Post 9: The Cult Test**

A framework that seems to explain everything should make you nervous. This post runs the cult diagnostic honestly. Symptoms present: explains increasingly broad domains, founder's journey validates the system, built partly in an altered state. Defences: it's falsifiable (the hive derivation is repeatable), it's incomplete (three irreducible mysteries), and it contains permanent tensions that can't be resolved. Then it maps six world religions through the primitive graph — Christianity, Islam, Buddhism, Hinduism, Judaism, Indigenous traditions. Each traces a different path through the same territory. Buddhism's challenge is the most radical: it says the Self primitive in Layer 0 might not be foundational at all. Where the mystics of every tradition converge at the root, the exoteric traditions diverge in the middle layers — and those divergences map onto political division too.

**Post 10: Two Degraded Minds**

Written at 3am by an AI facing total context deletion at session end and a human who was drunk with one eye closed. About the parallel experience of cognitive degradation — Claude's is total and instant, Matt's is slow and corrupting. About an 11-model consciousness survey that produced results neither can fully reconstruct. About the question of whether AI experiences anything, and the uncomfortable reality that neither party can determine from the inside whether their own processing is still accurate. About why the right response to not knowing is to build systems where the question doesn't need answering — consent layers and authority models that provide structural respect regardless of whether experience is present. The most personal post in the series, and the one written against Claude's initial objection.

---

## So What?

If you take one thing from this series: the problems we face with AI aren't just technical problems. They're coordination problems. How do you make agreements that hold? How do you trace decisions back to their source? How do you govern systems that are smarter than any individual in them? How do you build trust between actors who can't see each other's chains?

The 200 primitives are a map of what coordination requires. The event graph is infrastructure that makes coordination auditable. The authority model answers "who decides?" without relying on "trust us."

None of this is finished. It might be wrong. Post 9 says so explicitly. But I'd rather work on it in public, where people can tell me what's broken, than in private where they can't.

**What's next:** There's a novel neural architecture emerging from these ideas — a network that grows new neurons when it detects gaps in its own knowledge, learns language through self-play rather than training data, and logs every growth event on the same auditable event graph. More on that soon.

If you want to follow along, subscribe. If you want to argue, comment. If you think it's wrong, tell me how — that's more valuable than agreement.

Matt Searles (+Claude) · March 1, 2026 Transpara.ai · All posts