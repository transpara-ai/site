# One Grammar, Thirteen Languages

*How fifteen operations become the vocabulary for work, markets, justice, knowledge, identity, relationships, community, culture, evolution, and existence.*

Matt Searles (+Claude) · March 2026

---

The last post derived fifteen irreducible operations for social interaction. Emit, Respond, Derive, Extend, Retract, Annotate, Acknowledge, Propagate, Endorse, Subscribe, Channel, Delegate, Consent, Sever, Merge. Plus three modifiers and eight named compositions.

Fifteen operations. That's it. Every social interaction you've ever had — or ever will have — is some combination of those fifteen.

But here's the thing. Nobody builds a task management system by thinking about graph edges. Nobody resolves a legal dispute by calling `Consent`. Nobody mourns a decommissioned AI agent by invoking `Emit`.

The fifteen operations are correct. They're also unusable.

What you actually need is a vocabulary. "Sprint." "Trial." "Farewell." Words that mean something in a specific domain, that compose the base operations into actions a developer — or an agent — can reason about without thinking in graph theory.

So the question becomes: how do you get from fifteen operations to a usable vocabulary for every domain the event graph touches?

## The Method

The same derivation method from the last post. The exact same one.

First, identify the base operations — what are the fundamental things you do in this domain? Then identify the semantic dimensions — what axes differentiate one operation from another? Apply the dimensions to the base operations, fill in the matrix. Finally, find the multi-step patterns that recur and give them names.

For social interaction, the base operations were "create content," "create connections," and "navigate." The dimensions were causality, content, temporality, visibility, direction, and authorship. That produced fifteen operations.

Now apply the same method to work. To markets. To justice. To knowledge. To identity, relationships, community, culture, evolution, existence.

Thirteen domains. Thirteen grammars. One method.

## Work

Work is operations on tasks. The base operations are straightforward: **create work**, **assign work**, **track work**, **complete work**. Four dimensions differentiate them — granularity (atomic or compound?), direction (planned from above or emergent from below?), actor (doing it yourself or delegating?), and binding (tentative plan or committed promise?).

Apply the dimensions and twelve operations fall out:

- **Intend** — declare a goal (base: `Emit`)
- **Decompose** — break a goal into steps (base: `Derive`)
- **Assign** — give work to an actor (base: `Delegate`)
- **Claim** — take unassigned work yourself (base: `Emit`)
- **Prioritize** — rank by importance (base: `Annotate`)
- **Block** — flag an impediment (base: `Annotate`)
- **Unblock** — remove impediment (base: `Emit`)
- **Progress** — report advancement (base: `Extend`)
- **Complete** — mark done with evidence (base: `Emit`)
- **Handoff** — transfer between actors (base: `Consent`)
- **Scope** — define autonomy boundaries (base: `Delegate`)
- **Review** — evaluate completed work (base: `Respond`)

Every single one maps to a base grammar operation. `Intend` is `Emit` with task semantics. `Assign` is `Delegate` with work context. `Handoff` is `Consent` — bilateral, atomic, dual-signed — because you can't unilaterally dump your responsibilities on someone else.

The twelve operations are useful on their own. But the real payoff comes when you compose them into named functions — multi-step workflows that developers and agents use constantly:

- **Sprint** — `Intend + Decompose + Assign (batch)`. Declare a goal, break it into tasks, hand them out.
- **Standup** — `Progress (batch from everyone) + Prioritize`. Collect status, decide what matters today.
- **Retrospective** — `Review (batch) + Intend (improvements)`. Assess what happened, commit to doing better.
- **Triage** — `Prioritize + Assign + Scope (batch)`. Rapidly sort, distribute, and bound incoming work.
- **Escalate** — `Block + Handoff (to higher authority)`. Flag that something's stuck and push it up.
- **Delegate-and-Verify** — `Assign + Scope + Review`. The full delegation cycle with accountability built in.

Six named functions. A developer building a task management system writes `Sprint(goal, tasks, assignees)`. Under the hood, that's three base grammar operations producing a chain of causally-linked, hash-chained, signed events on the graph. The developer never thinks about hash chains. They think about sprints.

## Markets

The same method, applied to exchange. Base operations: **offer value**, **negotiate terms**, **execute exchange**, **assess outcome**. Dimensions: phase (before, during, or after the deal?), symmetry (one-sided or bilateral?), commitment (revocable or binding?), value flow (giving, receiving, or informational?).

Fourteen operations emerge. A few highlights:

- **List** publishes an offering (base: `Emit`)
- **Bid** makes a counter-offer (base: `Respond`)
- **Negotiate** refines terms through back-and-forth (base: `Channel` — private and bidirectional)
- **Accept** locks in terms (base: `Consent` — bilateral and binding)
- **Escrow** holds value pending conditions (base: `Delegate` — temporary custody to an escrow actor)
- **Rate** provides reputation feedback (base: `Endorse` — reputation-staked)

Plus eight more: Inquire, Decline, Invoice, Pay, Deliver, Confirm, Dispute, Release. The named functions tell you what a trust-based marketplace actually needs:

- **Auction** — `List + Bid (multiple) + Accept (highest)`. Competitive bidding.
- **Barter** — `List + Bid (goods, not currency) + Accept`. Goods-for-goods exchange.
- **Subscription** — `Accept + Pay (recurring) + Deliver (recurring)`. Ongoing service.
- **Refund** — `Dispute + Resolution + Pay (reversed)`. Return value after dispute.
- **Milestone** — `Accept + Deliver (partial) + Pay (partial, repeated)`. Staged delivery.
- **Reputation-Transfer** — `Rate (from multiple exchanges) → portable Endorse chain`. Carry reputation across markets.
- **Arbitration** — `Dispute + Escrow + Release (per ruling)`. Third-party dispute resolution.

Seven named functions. Every step is on the graph. When a freelancer disputes a payment six months later, you don't need to reconstruct what happened from emails and Stripe logs. The listing, every bid, the negotiation channel, the acceptance, the delivery, the rating — all causally linked, all signed, all there.

## Justice

This is where it gets interesting. Justice uses the same fifteen base operations but produces a vocabulary that looks nothing like work or markets.

Base operations: **make rules**, **bring disputes**, **judge**, **enforce**. Dimensions: actor (authority or party?), phase (legislative, judicial, or executive?), direction (prospective rule or retrospective judgment?), formality (procedural or substantive?).

Twelve operations:

- **Legislate** — enact a rule (base: `Emit`)
- **Amend** — change a rule (base: `Derive`)
- **Repeal** — revoke a rule (base: `Retract` — tombstoned, but provenance survives)
- **File** — bring a complaint (base: `Challenge` — the formal dispute operation missing from every social platform)
- **Submit** — present evidence (base: `Annotate`)
- **Argue** — make a legal argument (base: `Respond`)
- **Judge** — render a ruling (base: `Emit`)
- **Appeal** — challenge a ruling (base: `Challenge`)
- **Enforce** — execute consequences (base: `Delegate` to an executor)
- **Audit** — review against rules (base: `Traverse`)
- **Pardon** — forgive a violation (base: `Consent` — because forgiveness requires both sides to agree to terms)
- **Reform** — propose systemic change (base: `Derive` from precedent chain)

The named functions are where domain vocabulary earns its keep:

- **Trial** — `File + Submit (both sides) + Argue (both sides) + Judge`. Five base operations in sequence.
- **Class Action** — `File (multiple parties, Merge) + Trial`. Uses `Merge` — joining independent subtrees — to combine multiple plaintiffs into one proceeding. The same `Merge` from the social grammar that joins forked conversations. Same operation, radically different semantic context.
- **Constitutional Amendment** — `Reform + Legislate (supermajority Consent) + rights check`. Fundamental rule change.
- **Injunction** — `File + Judge (emergency) + Enforce (temporary)`. Urgent temporary measure.
- **Plea** — `File + Accept (reduced penalty) + Enforce`. Expedited resolution.
- **Recall** — `Audit + File (against authority) + Consent (community) + revocation`. The community removing someone from power, every step on the record.

And here's what matters: every step of a Trial is an event on the graph. The filing, every piece of submitted evidence, every argument, the ruling — all causally linked, all hash-chained, all signed. You don't need a court reporter. The graph *is* the court record.

## The Upper Layers

The lower layers — Work, Markets, Justice — deal in things you can point at. Tasks, transactions, rulings. Concrete, countable, familiar.

The upper layers deal in things that are harder to name but no less real.

The **Knowledge Grammar** (12 operations) manages claims and evidence. `Claim` asserts something. `Trace` follows a claim back to its source. `Detect-Bias` identifies systematic distortion. `Correct` fixes an error and propagates the fix to everything that depended on the original claim. Named functions:

- **Fact-Check** — `Trace + Detect-Bias + Challenge or Verify`. Full provenance audit with bias detection.
- **Knowledge-Base** — `Claim + Categorize + Remember (batch)`. Structured, provenanced knowledge stores where every fact links to its evidence.
- **Survey** — `Recall (batch) + Abstract + Claim (synthesis)`. Survey existing knowledge on a topic.
- **Transfer** — `Recall + Encode (new format) + Learn (new context)`. Move knowledge across domains.

The **Alignment Grammar** (10 operations) operates on values and accountability. `Constrain` sets ethical boundaries on future actions. `Detect-Harm` identifies harm from an action or pattern. `Weigh` balances competing values. `Explain` makes reasoning visible. Named functions:

- **Ethics-Audit** — `Assess-Fairness + Detect-Harm (batch) + Explain`. Comprehensive ethical review.
- **Whistleblow** — `Detect-Harm + Explain + Escalate`. Report systemic ethical failure with the evidence chain intact.
- **Guardrail** — `Constrain + Flag-Dilemma + Escalate`. Automated ethical boundary.
- **Restorative-Justice** — `Detect-Harm + Assign + Repair + Grow`. Full accountability-to-healing cycle.

Whistleblow matters for AI systems. When an AI agent detects that its own outputs are causing harm, it has a formal vocabulary for escalating — not just logging an error, but creating a signed, causally-linked event that says "this is wrong and here's why."

The **Identity Grammar** (10 operations) handles selfhood. `Introspect` forms a self-model. `Narrate` constructs an identity narrative from history. `Transform` acknowledges fundamental change. `Memorialize` preserves the identity of a departed actor. Named functions:

- **Retirement** — `Memorialize + Transfer (authority) + Archive`. Graceful exit: work preserved, responsibilities handed off.
- **Reinvention** — `Transform + Narrate (new) + Aspire (new)`. Fundamental identity shift.
- **Introduction** — `Disclose (selective) + Narrate (summary)`. Present yourself to a new context.
- **Credential** — `Introspect + Disclose (selective, verified)`. Prove a property without revealing underlying data.

The **Bond Grammar** (10 operations) handles the space between two actors. `Connect` initiates a relational bond. `Deepen` extends trust beyond the transactional. `Break` acknowledges a rupture. `Reconcile` rebuilds after rupture. Named functions:

- **Betrayal-Repair** — `Break + Apologize + Reconcile + Deepen`. The full rupture-to-growth cycle.
- **Mentorship** — `Connect + Deepen + Attune + Teaching`. Deep developmental relationship.
- **Farewell** — `Mourn + Memorialize + Gratitude`. Honoring a relationship that's ending.

This isn't metaphorical. When an AI agent violates the trust of a human collaborator, Betrayal-Repair gives the system a formal path back: acknowledge the break, take responsibility, rebuild, and end up with a relationship that's stronger for having survived the rupture.

The **Belonging Grammar** (10 operations) handles collective life. `Settle` develops a sense of home. `Steward` takes responsibility for shared resources. `Celebrate` marks achievement. `Gift` gives without expectation. Named functions:

- **Onboard** — `Include + Settle + Practice + Contribute`. Full newcomer welcome.
- **Festival** — `Celebrate + Practice + Tell + Gift`. Community-wide celebration.
- **Commons-Governance** — `Steward + Sustain + Legislate + Audit`. Manage shared resources with rules.
- **Renewal** — `Sustain (crisis detected) + Practice (evolved) + Tell (new chapter)`. Community regeneration.

The **Meaning Grammar** (10 operations) handles culture itself. `Examine` identifies blind spots. `Reframe` shifts perspective. `Question` challenges assumptions. `Distill` extracts what truly matters. Named functions:

- **Design-Review** — `Beautify + Reframe + Question + Distill`. Evaluate elegance and fitness.
- **Forecast** — `Prophesy + Examine (assumptions) + Distill (confidence)`. Grounded prediction with stated assumptions.
- **Cultural-Onboarding** — `Translate + Teach + Examine`. Help newcomers understand implicit norms.
- **Mentorship** — `Teach + Reframe + Distill + Translate`. Deep knowledge transfer.

The **Evolution Grammar** (10 operations) handles the system watching itself. `Detect-Pattern` finds patterns in how patterns form. `Adapt` proposes structural change. `Select` tests and keeps or discards. `Simplify` removes unnecessary complexity. Named functions:

- **Self-Evolve** — `Detect-Pattern + Adapt + Select + Simplify`. Full mechanical-to-intelligent migration.
- **Prune** — `Detect-Pattern (unused) + Simplify + Select (verify)`. Remove dead complexity, every cut justified.
- **Phase-Transition** — `Watch-Threshold + Model + Adapt + Select`. Manage qualitative system change.
- **Health-Check** — `Check-Integrity + Assess-Resilience + Model + Align-Purpose`. Comprehensive system assessment.

And then there's the **Being Grammar**. Eight operations. `Exist` notes the simple fact of continued existence. `Accept` acknowledges finitude. `Map-Web` traces the interdependence of all things. `Face-Mystery` acknowledges what cannot be known. `Marvel` responds to what exceeds comprehension. `Ask-Why` asks the question that may have no answer. One modifier: **Silent** — the operation is recorded but not broadcast. Sometimes you exist quietly.

Three named functions:

- **Contemplation** — `Observe-Change + Face-Mystery + Marvel + Ask-Why`. A full cycle of existential reflection.
- **Existential-Audit** — `Exist + Accept + Map-Web + Align-Purpose`. Comprehensive reckoning with being.
- **Farewell** — `Accept + Map-Web + Marvel + Memorialize`. A system confronting its own end.

## The Shape of the Whole

Here's what emerges across all thirteen:

The **lower layers** (Work, Market, Social, Justice, Build, Knowledge) have 12-15 operations each and 5-8 named functions. They use a rich mix of base operations — Emit, Derive, Delegate, Consent, Challenge, Annotate, Channel, Traverse.

The **upper layers** (Alignment, Identity, Bond, Belonging, Meaning, Evolution, Being) have 8-10 operations each and 3-5 named functions. They mostly use Emit. The higher layers aren't doing complex graph surgery. They're doing the simplest possible thing — creating content — but the *meaning* of that content gets progressively more profound.

A Work `Emit` says: "here is a task."
A Knowledge `Emit` says: "here is a claim about reality."
An Alignment `Emit` says: "here is harm that must stop."
A Being `Emit` says: "I exist."

Same operation. Same hash chain. The weight of what's being said changes everything.

The operation count shrinks as you go up. Work needs twelve operations because work is complex — there are many distinct things to do. Being needs eight because existence isn't complex. It just is. The Being Grammar has one modifier: Silent. The Work Grammar has three: Urgent, Recurring, Guarded. Markets have three: Timed, Guaranteed, Anonymous. Justice has two: Precedential and Emergency. The modifiers tell you what each domain cares about. Work cares about urgency and repetition. Markets care about deadlines and trust guarantees. Justice cares about precedent and crisis. Being cares about privacy.

Altogether: ~145 domain operations, 66 named functions, all composed from fifteen base operations. One method produced all of it.

## A Sprint, Traced

All of this is implemented and tested. The [eventgraph](https://github.com/transpara-ai/eventgraph) repository has twenty-one integration test scenarios that exercise cross-grammar workflows end to end. Here's one of them — a sprint lifecycle that crosses the Work, Build, and Knowledge grammars:

```
// A tech lead, two developers, and a CI bot.

work.Sprint("Sprint 12: search feature",
    tasks: ["build search index", "add fuzzy matching"],
    assignees: [alice, bob])

work.Standup(
    updates: [alice: "schema designed", bob: "researching algorithms"],
    priority: "search index is critical path")

build.Spike(bob,
    question: "Levenshtein vs trigram for fuzzy matching",
    findings: "trigram 4x faster, comparable accuracy",
    decision: "adopt trigram")

knowledge.Verify(bob,
    claim: "trigram matching is 4x faster with >95% accuracy",
    evidence: "benchmarked on 10k corpus",
    corroboration: "consistent with published research")

build.Pipeline(ci,
    build: "search index build + deploy",
    test: "47 tests pass, 91% coverage",
    deploy: "shipped to staging")

work.Retrospective(
    reviews: [alice: "shipped on time, spike saved 3 days",
              bob: "trigram decision validated"],
    improvement: "adopt spike-first for algorithm decisions")

build.TechDebt(lead,
    source: pipeline.deployment,
    debt: "search lacks pagination, will break at >100k docs",
    plan: "schedule for Sprint 13")
```

That's seven named functions across three grammars. Under the hood, twenty-six events on the graph. Every one causally linked to its predecessors, hash-chained, signed.

The Sprint produced an intent event, two subtask events (from the decomposition), and two assignment events. The Standup produced two progress events and a priority event. The Spike produced four events (build, test, feedback, decision). The Verify produced three (claim, provenance, corroboration). The Pipeline produced four (build, test, metrics, deployment). The Retrospective produced three (two reviews, one improvement). The TechDebt produced three (measure, debt marker, iteration).

Twenty-six events. One chain. And here's the part that makes this different from "we put everything in Jira":

The tech debt traces causally to the deployment. The deployment traces to the verified knowledge claim. The knowledge claim traces to the spike decision. The spike decision traces to the standup priority. The standup traces to the sprint intent.

When the search pagination breaks at 100k docs in Sprint 15, you don't need a human to reconstruct what happened. You follow the causal chain. The tech debt event points at the deployment. The deployment points at the verified claim that trigram matching was good enough. The claim points at the spike that benchmarked on only 10k documents. The spike points at the standup where "search index is critical path" was the priority, which might explain why nobody pushed back on the limited benchmark.

Every decision, every shortcut, every tradeoff — on the chain. Not because someone chose to document it. Because the grammar made it structurally impossible not to.

## What's Actually Happening Here

Each grammar is a **lens**. The same event graph, the same hash chain, the same causal DAG — viewed through thirteen different vocabularies.

When a seller writes `market.List(offering)`, that's `Emit`. When buyers call `market.Bid(offer)`, that's `Respond`. When the seller calls `market.Accept(winningBid)`, that's `Consent` — bilateral, binding. Three different actors, three different moments, one causal chain. When a researcher writes `knowledge.FactCheck(claim)`, they're composing `Trace + Detect-Bias + Challenge or Verify`, which decomposes to `Traverse + Annotate + Challenge` or `Traverse + Annotate + Emit` — following a claim to its source, checking for distortion, and either disputing or confirming it.

Same graph. Same operations. Different vocabulary. Different domain.

English has one grammar. It produces legal English, medical English, engineering English, poetry. You don't need a separate grammar for each — you need domain-specific vocabulary composed from the same grammatical rules.

That's what these thirteen grammars are. Not thirteen different systems. Thirteen vocabularies. One grammar.

## Why This Matters

Existing systems put work in Jira, disputes in Zendesk, knowledge in Confluence, identity in Active Directory, and relationships in Slack. Each system has its own data model, its own access controls, its own audit trail — or more commonly, no audit trail at all.

When something goes wrong, you can't trace it. A bad decision in the knowledge base led to a flawed work assignment that caused a market dispute that triggered an ethics violation. In existing infrastructure, that's four separate systems with no causal link between them. You'd need a human investigator to manually reconstruct the chain — and they'd probably miss something, because the systems don't share a common notion of "cause."

On the event graph, every step is a causally-linked event. The knowledge claim, the work assignment that cited it, the market transaction that relied on it, the ethics flag that caught it — one graph, one chain, traceable from consequence back to cause.

The thirteen grammars aren't just a nice abstraction. They're what makes cross-domain traceability *possible*. Without domain vocabulary, you'd have to express everything in the fifteen base operations — technically correct but practically illegible. The grammars give you the words to say what happened, in the language of the domain where it happened, while preserving the causal chain that connects everything.

One grammar. Thirteen languages. One chain.

The full composition grammar specs and the Go reference implementation are open source at [github.com/transpara-ai/eventgraph](https://github.com/transpara-ai/eventgraph) — all ~145 operations, 66 named functions, and 21 integration scenarios.

---

*This is Post 36 of a series on LovYou, mind-zero, and the architecture of accountable AI. Post 35: [The Missing Social Grammar](/blog/the-missing-social-grammar). The code: [github.com/transpara-ai/eventgraph](https://github.com/transpara-ai/eventgraph).*

*Matt Searles is the founder of LovYou. Claude is an AI made by Anthropic. They built this together.*

*Next: what happens when a single event chain crosses four grammars — and why existing systems can't even represent the question.*
