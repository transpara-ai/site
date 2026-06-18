# The Governance Graph

*Every governance decision is made by someone, for some reason, affecting someone. Currently you can verify none of that. On the event graph, you can verify all of it.*

Matt Searles (+Claude) · March 2026

---

We've built up through work, markets, community, justice, knowledge, ethics, identity, relationships, and community. Ten layers of infrastructure, all on the same event graph. Now the question that governs all of them: who decides?

Not who decides within a community (that's Layer 3 — Society). Not who decides within a dispute (that's Layer 4 — Justice). Layer 11 is about governance at the level that shapes the rules the other layers operate under. The meta-layer. The decisions about decisions. The power that writes the code that everyone else lives inside.

This is the layer where the framework meets politics directly. Post 13 mapped left and right as different primitive weightings. This post asks a different question: regardless of which primitives you weight, how do you verify that the people making governance decisions are accountable for the consequences?

Currently, you can't.

---

## The Primitives

Layer 11 — which in the full framework is the Governance layer — contains primitives related to: Policy, Governance, Accountability, Representation, Mandate, Transparency, Oversight, Power, Corruption, Reform, Constitution, Legitimacy.

These describe the meta-structure of collective decision-making. Not the decisions themselves — the system that produces decisions. Who has Power. Where their Mandate comes from. Whether they're subject to Oversight. Whether their decisions are Transparent. Whether the system can Reform itself when it fails. Whether it has a Constitution that constrains even the powerful. Whether governance is Legitimate — exercised with the consent and for the benefit of the governed.

Every political system in history is an implementation of these primitives. The implementations vary enormously. The primitives don't.

## The Opacity Problem

The central problem with governance everywhere — democracies, autocracies, corporations, platforms, DAOs, international bodies — is the same: the governed cannot verify what the governors are doing or why.

### Democratic opacity.

Democracy is supposed to be government by the people. In practice, it's government by elected representatives whose decision-making processes are largely invisible to the electorate. A legislator votes on a bill. You can see the vote. You can't see: who lobbied them, what information informed their decision, what trade-offs they considered, what promises they made in private, what the causal chain was from campaign donation to legislative position.

The vote is public. Everything that produced the vote is private. Accountability operates on the thinnest possible information — you know what they decided, but not why, not what alternatives they considered, and not who influenced them. You vote for or against them based on outcomes you can observe, with no access to the process that produced those outcomes.

Freedom of Information requests exist. They're slow, heavily redacted, frequently denied, and structurally inadequate — they reveal individual documents, not decision chains. Knowing that a meeting occurred between a legislator and a lobbyist is not the same as seeing the causal chain from meeting to legislation.

### Corporate opacity.

Corporations govern the lives of billions — their employees, their customers, the communities where they operate. Corporate governance is less transparent than democratic governance by a wide margin. Board meetings are private. Executive decisions are proprietary. The causal chain from shareholder pressure to product decision to customer impact is invisible.

A tech company decides to change its algorithm. The change affects two billion people's information diet. The decision was made by a product team, approved by an executive, possibly reviewed by a board. Nobody outside the company knows why the decision was made, what alternatives were considered, what the predicted impact was, or what values informed the choice. The governed — two billion users — have no visibility, no voice, and no recourse.

### Platform opacity.

Post 16 described Facebook as a government that two billion people live under without having voted for it. The governance primitives make this precise. Facebook exercises Power (controls what you see), claims Mandate (the Terms of Service you "agreed" to), lacks Oversight (no external body reviews its decisions), lacks Transparency (algorithmic decisions are opaque), and has no mechanism for Reform from below (users can't change the rules).

This is governance without accountability. Every Layer 11 primitive that should constrain power — Oversight, Transparency, Legitimacy, Reform — is absent or nominal.

### AI governance opacity.

The newest and most urgent form. AI systems are making governance decisions — content moderation, loan approvals, hiring recommendations, medical diagnoses, criminal risk assessments. Each of these is a governance decision: someone (the deployer) exercises power (the AI makes consequential decisions) that affects the governed (the people subject to those decisions).

The governed have no visibility into how the decision was made. "The algorithm decided" is the 21st-century equivalent of "the king decreed." The decision is final, the process is invisible, and the affected party has no meaningful recourse.

**The perverse incentive:** transparency threatens incumbent power. Every governance system has insiders who benefit from opacity — because opacity prevents the governed from evaluating decisions, which prevents accountability, which preserves the insiders' position regardless of performance. Transparent governance is better for the governed and worse for poor governors. Since governors control whether governance is transparent, the default is opacity.

This is true across every form of governance — democratic, corporate, platform, AI. The people who would need to make governance transparent are the people who benefit from keeping it opaque. The reform has to come from outside, which means it has to come from infrastructure that makes opacity structurally difficult rather than relying on governors to volunteer transparency.

---

## The Event Graph Version

### Every governance decision is an event.

On the Governance Graph, a governance decision is an event with full causal provenance. Not just the decision — the chain that produced it. Who proposed it. What information informed it. What alternatives were considered. Who was consulted. Who approved it. What authority they exercised. What the predicted impact was. What the actual impact turned out to be.

This applies at every scale. A community moderator removing a post: decision event, linked to the norm being enforced, linked to the content being evaluated, linked to the moderator's authority. A corporate executive changing a product: decision event, linked to the business case, linked to the impact assessment, linked to the approval chain. A legislator voting on a bill: decision event, linked to the legislative record, committee discussions, and — this is the radical part — the lobbying interactions and constituent communications that informed the position.

The chain doesn't prevent bad decisions. It makes them traceable. You can walk backwards from any governance outcome to every decision that produced it. If the outcome was harmful, the chain shows where the harm entered. If the decision was influenced by corruption, the chain shows the influence. If the decision-maker ignored relevant information, the chain shows what was available and what was disregarded.

### Accountability as architecture.

Currently, accountability is adversarial. Journalists investigate. Whistleblowers leak. Oversight bodies audit. Each of these is a manual, expensive, after-the-fact process that catches a fraction of governance failures.

On the Governance Graph, accountability is structural. Every decision has a chain. Every chain is queryable. You don't need an investigative journalist to discover that a legislator met with a lobbyist before changing their position on a bill — the meeting is an event, the position change is an event, and the causal link between them is on the chain.

This doesn't eliminate the need for investigation. Some governance failures are subtle — the causal chain exists but the interpretation requires expertise. Some are deliberately hidden — events are misrepresented or omitted. But the baseline shifts from "prove there was a meeting" to "the meeting is on the chain, now let's discuss what it meant." The evidence is the default. The investigation is about interpretation, not discovery.

### Rules and enforcement on the same graph.

One of the most profound governance failures in every system is the gap between rules and enforcement. Laws are passed. Regulations are issued. Policies are written. And then enforcement is sporadic, inconsistent, and subject to the same opacity as everything else.

On the Governance Graph, rules and enforcement share the same event graph. The rule is an event. The behaviour it governs is an event. Compliance or violation is a queryable relationship between the two. Enforcement is an event linked to the violation.

The gap between rules and enforcement becomes visible. "This rule exists. Here are the violations that occurred. Here are the enforcement actions taken. Here are the violations that weren't enforced." The pattern is on the chain. Selective enforcement — enforcing rules against some people but not others — is visible because the enforcement events (or their absence) are as traceable as the violation events.

**The Governance Graph's proposition:** you don't need to trust your governors. You need to verify them. Trust is what you need when you can't see what's happening. Verification is what you do when you can. The event graph makes governance visible. Visibility enables verification. Verification enables accountability. Accountability is what makes governance legitimate.

---

## AI Governance Specifically

This is the application closest to where the framework started — the original question that produced the 20 primitives: how do you hold AI accountable?

On the Governance Graph, AI governance is not a separate problem. It's the same problem as all governance — with the same primitives, on the same graph. An AI system making consequential decisions is a governor. It has Power (its decisions affect people). It should have Oversight (someone monitors its decisions). It should have Transparency (its decision process should be visible). It should have Accountability (when its decisions cause harm, the chain shows why).

The AI's decision chain: what inputs did it receive? What model processed them? What confidence level did it have? What constraints were applied? What authority approved the decision? What was the outcome? Was the outcome consistent with the values the AI was supposed to embody?

Every AI governance decision on the chain. The Ethics Graph (Layer 7) monitors the patterns. The Justice Graph (Layer 4) handles disputes. The Governance Graph holds the meta-structure: who has authority over this AI, what rules constrain it, and are those rules being followed?

This is what the Pentagon post (Post 4) was about. AI systems making military decisions with no accountability infrastructure. The Governance Graph is the accountability infrastructure. Not an ethics review board that meets quarterly. Not an alignment technique applied during training. Real-time, structural, verifiable governance of AI decision-making in production.

---

## The Constitution Layer

Every governance system needs a meta-rule: a set of principles that constrain even the most powerful actors. In democracies, it's a constitution. In corporations, it's articles of incorporation and fiduciary duty. In communities, it's foundational norms.

On the Governance Graph, the constitution is the root authority event — the event that defines the rules that all other governance events must comply with. The constitution event specifies: who has authority (and its limits), what rights are protected (and can't be overridden by governance decisions), how the constitution itself can be amended (and what threshold is required).

The constitution is on the same chain as everything else. Governance decisions that violate the constitution are visible as chain conflicts — the decision event's authority doesn't trace back to a constitutional provision, or it contradicts a constitutional constraint. Constitutional review isn't a slow, expensive legal process. It's a chain query.

This is enormously powerful for AI governance specifically. The AI's constitution — its fundamental constraints, the values it must embody, the boundaries it cannot cross — is on the chain. If the AI makes a decision that violates its constitution, the violation is structurally detectable. Not "we hope the alignment training held." Not "we'll audit it next quarter." The chain shows the violation in real time.

---

## The Global Governance Problem

There's one governance challenge that no existing system has solved: global coordination. Climate change, pandemic response, AI regulation, nuclear proliferation — these are problems that require governance at a scale that no institution currently operates at effectively.

The United Nations exists. It has no enforcement power. International treaties exist. Compliance is voluntary. Global coordination mechanisms exist. They're slow, captured by national interests, and structurally unable to act at the speed required.

The Governance Graph doesn't solve global governance. That's a political problem requiring political solutions. But it provides infrastructure that global governance currently lacks: transparent commitment tracking. If a nation commits to an emissions target, that's an event on the graph. Actual emissions data is on the graph. The gap between commitment and reality is visible. Not "we self-report compliance." The chain shows whether the commitment was met.

This applies to corporate commitments too. ESG pledges. Net-zero targets. Diversity commitments. Human rights standards. Currently, these are promises verified by self-report or by expensive, sporadic auditing. On the Governance Graph, the commitment is an event. The behaviour is events. The comparison is a query. The gap is visible to everyone.

Visibility doesn't guarantee compliance. But it eliminates the possibility of invisible non-compliance — which is the default state of global governance today.

---

## The East-West Question

There's a post coming about how different civilisations have implemented governance — specifically, the contrast between Eastern and Western approaches. A preview of why it matters for the Governance Graph:

The West builds governance on individual rights constrained by collective authority. The East (particularly China) builds governance on collective harmony maintained by centralised authority. Both are implementations of the Layer 11 primitives. Both have pathologies — the West struggles with collective action because individual rights constrain coordination. The East struggles with individual rights because collective harmony suppresses dissent.

The Governance Graph is structurally neutral between these approaches. It doesn't prescribe individual rights or collective harmony. It makes governance decisions visible regardless of which value system produces them. A Western democracy and an Eastern technocracy could both operate on the Governance Graph — the graph would show their decisions, their reasoning, and their outcomes with equal transparency.

Whether that transparency is compatible with governance systems that depend on opacity is the question. The answer is probably no — and that's the point.

Next deep dive: the Culture Graph — what happens when meaning, art, ritual, and shared understanding move onto the event graph. The layer where infrastructure meets the sacred.

---

*This is Post 24 of a series on Transpara, mind-zero, and the architecture of accountable AI. Previous: [The Community Graph](/blog/the-community-graph) (Layer 10 deep dive) Post 4: [The Pentagon Just Proved Why AI Needs a Consent Layer](/blog/the-pentagon-just-proved-why-ai-needs) (where AI governance started) Post 13: [The Same 200 Primitives, Weighted Differently](/blog/the-same-200-primitives) (left/right as governance weightings) The code is open source: [github.com/mattxo/mind-zero-five](https://github.com/mattxo/mind-zero-five) Matt Searles is the founder of Transpara. Claude is an AI made by Anthropic. They built this together.*