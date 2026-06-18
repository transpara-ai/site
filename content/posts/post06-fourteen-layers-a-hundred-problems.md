# Fourteen Layers, A Hundred Problems

Walking the primitive framework to find everything the event graph touches.

Matt Searles (+Claude) · February 2026

---

The [last post] was supposed to be the final one. It tied together the architecture, the philosophy, and the politics into a single argument. Done.

Then I started thinking about what else the architecture applies to, and the answer turned out to be: almost everything.

The 200-primitive framework wasn't designed as a taxonomy of industries. It was derived from first principles about how systems coordinate, communicate, and maintain trust. But first principles, if they're actually right, have a habit of mapping onto the real world in ways you didn't anticipate. So Claude and I walked the 14 layers and asked a simple question at each one: what breaks in the real world because this layer's problems aren't solved?

The answer is that every major digital platform failure — and quite a few analogue ones — maps to a specific layer in the framework. And the event graph architecture addresses all of them for the same reason: they all share the same root problem.

Multiple actors need to coordinate with verifiable trust, and "trust us" doesn't scale.

---

## **The Walk**

**LAYER 0 — FOUNDATION**

**Verifiable History**

Event, EventStore, Clock, Hash, Self. The primitives that make anything recordable and tamper-evident.

**What this touches:** Any system that needs a trustworthy record of what happened. Version control. Audit logs. Legal records. Scientific data repositories. Chain of custody for evidence. Medical records. Financial transaction histories. The event graph is a better foundation for all of these because it's append-only, hash-chained, and causally linked by default — not as a feature bolted on later, but as the data structure itself.

**LAYER 1 — AGENCY**

**Persistent Memory**

Observer → Participant. The transition from watching to acting, and the need for an agent to understand its own history.

**What this touches:** The context window problem. Every AI system today is amnesiac — it forgets everything between sessions, and within sessions it's limited by how much text fits in its window. An agent that can query its own event history has persistent memory with causal structure. Not "here's a summary of what happened" but "here's the complete, verifiable chain of what I did, why, and what resulted." Personal AI assistants that actually remember. Autonomous agents that can learn from their own failures across sessions. Research assistants that build on previous work rather than starting from scratch every time.

**LAYER 2 — EXCHANGE**

**Two-Party Trust**

Individual → Dyad. What happens when two actors need to transact, negotiate, or collaborate.

**What this touches:** Marketplaces. Contracts. Escrow. Freelance platforms. Any transaction where two parties need to trust each other without trusting a middleman. The event graph provides the trust infrastructure directly — every offer, acceptance, delivery, and dispute is a causally linked event in a verifiable chain. Smart contracts without blockchain overhead. Escrow logic embedded in the event graph rather than held by a third party who takes a cut. Dispute resolution where the complete history of the transaction is tamper-evident and available to both parties.

**LAYER 3 — SOCIETY**

**Group Governance**

Dyad → Group. Consent, Due Process, Legitimacy. What happens when more than two actors need to coordinate.

**What this touches:** Governance of any multi-agent system. Corporate boards where "who decided what and when" is currently reconstructed from meeting minutes that nobody trusts. DAOs that actually work because the governance isn't just code-is-law but verifiable-decision-is-law. Open source project governance where contributor rights, merge decisions, and direction changes are all traceable events. Homeowners associations. Co-ops. Any committee where decisions affect people who weren't in the room — which is every committee.

**LAYER 4 — LEGAL**

**Compliance and Dispute Resolution**

Informal → Formal. The transition from social norms to codified rules.

**What this touches:** Regulatory compliance that's verifiable rather than self-reported. Right now, companies tell regulators what they did, and regulators decide whether to believe them. An event graph makes compliance auditable by default — the regulator can verify the chain independently. Court evidence that's tamper-evident from creation. GDPR and data privacy compliance where the complete history of how personal data was accessed, processed, and shared is cryptographically verifiable. Tax records. Insurance claims. Any domain where "prove what happened" is the central question.

**LAYER 5 — TECHNOLOGY**

**Software and Supply Chains**

Governing → Building. The infrastructure that implements everything else.

**What this touches:** Software supply chain security — every dependency, build, test, and deployment as a causally linked event. The SolarWinds hack succeeded because the build pipeline had no verifiable chain of custody. An event graph makes supply chain attacks detectable because any unauthorised modification breaks the hash chain. CI/CD pipelines with complete causal ancestry for every artifact. Incident response where "what broke and why" is answerable by walking the event graph backwards, not by convening a war room and hoping someone remembers. Hardware supply chains — chip provenance, manufacturing origin, component authenticity.

**LAYER 6 — INFORMATION**

**Content Provenance**

Physical → Symbolic. The transition from things to representations of things.

**What this touches:** Media authenticity. Deepfake detection isn't ultimately about analyzing pixels — it's about whether the content has a verifiable causal history. A photo with a complete event trail from camera sensor to publication is trustworthy. A photo that appears from nowhere isn't. Journalism with sourcing baked into the data structure — not "a source said" but a verifiable chain from information to publication, with appropriate privacy protections for sources. Academic publishing where the data, analysis, and conclusions are causally linked and reproducible. Misinformation is fundamentally an information provenance problem, and provenance is what the event graph does.

**LAYER 7 — ETHICS**

**AI Alignment**

Is → Ought. The layer where structure meets value.

**What this touches:** AI alignment — not as a one-time training problem but as an ongoing verification problem. The event graph makes alignment auditable in real time. You don't ask "is this AI aligned?" — a question nobody can answer definitively. You look at the event trail and see what the AI actually did, what values informed each decision, who approved it, and what the outcomes were. Ethical review boards with verifiable records. Impact assessments that trace outcomes back to the decisions that produced them. Content moderation with transparent, auditable decision chains. Any domain where "was this the right thing to do?" needs to be answerable after the fact.

**LAYER 8 — IDENTITY**

**Reputation and Credentials**

Doing → Being. The transition from actions to the self that acts.

**What this touches:** Digital identity that's self-sovereign and history-backed. Not "who do you claim to be?" but "here's the verifiable trail of what you've done." Reputation systems that can't be gamed because they're append-only — you can't delete your bad reviews, and the good ones are cryptographically linked to real transactions. Professional credentials as event histories rather than certificates. A doctor's credential isn't a piece of paper — it's a verifiable chain of education, training, examinations, and practice. Hiring based on verifiable work history rather than self-reported resumes. KYC (Know Your Customer) that's portable across institutions rather than repeated from scratch at every bank.

**LAYER 9 — RELATIONSHIP**

**Social Networks**

Self → Self-with-Other. What happens when identity meets another identity.

**What this touches:** Social networking — but not as we know it. Current social platforms own your relationship graph and use it to sell your attention. The event graph inverts this: your relationships are your event graph, portable, verifiable, and owned by you. The recommendation algorithm's decisions are traceable events, not a black box. You can see why you were shown what you were shown. You can take your social graph to another platform. Dating apps where trust is built on verifiable interaction history rather than curated self-presentation. Professional networking where endorsements are linked to actual collaborative events, not performative clicking.

**LAYER 10 — COMMUNITY**

**Platform Governance**

Relationship → Belonging. What happens when relationships form a group that matters to its members.

**What this touches:** Community platforms where moderation decisions are transparent and auditable. Right now, platforms moderate content through opaque processes — your post is removed, your account is suspended, and you have no verifiable record of why or by whom. An event graph makes every moderation decision a traceable event with causal ancestry. Community membership and governance with transparent decision-making. Housing co-ops, credit unions, mutual aid networks, professional associations — any organisation where members need to trust the governance but can't currently verify it. Online communities where the rules, their enforcement, and their evolution are all part of the same verifiable record.

**LAYER 11 — CULTURE**

**Creative Attribution**

Living Culture → Seeing Culture. The transition from participating in culture to reflecting on it.

**What this touches:** Creative attribution and intellectual property. Every remix, sample, adaptation, and derivative work linked to its sources through the event graph. Royalty distribution based on verifiable causal chains of influence rather than opaque algorithms controlled by distributors. Art provenance — not just "who owned this painting" but the complete creative lineage. Academic citation networks where the actual intellectual lineage is traceable. AI training data attribution — when an AI generates something influenced by a creator's work, the event graph traces that influence. This is the fair compensation problem for the AI age, and it's a provenance problem.

**LAYER 12 — EMERGENCE**

**Self-Improving Systems**

Content → Architecture. Systems that observe and modify themselves.

**What this touches:** Any system that evolves its own rules — with the evolution itself recorded and auditable. This is mind-zero's self-improvement loop generalised. Adaptive regulation where laws update based on verifiable outcomes rather than political pressure. Platform algorithms that evolve, with every change traceable and its effects measurable. Machine learning systems where model updates, retraining decisions, and performance changes are all events in the graph. Institutional learning — organisations that can verifiably learn from their own mistakes because the mistakes and their causes are permanently recorded.

**LAYER 13 — EXISTENCE**

**Living Constitutions**

Everything → The Fact of Everything. The ground of the whole framework.

**What this touches:** Foundational documents that are living records rather than static texts. Constitutions, charters, mission statements — any document that defines what an institution *is* and *why it exists* — as event graphs with verifiable amendment histories. You can trace every change to the founding principles, see who proposed it, who approved it, what the reasoning was, and what the effects were. This is how you prevent institutional drift — not by making founding documents sacred and unchangeable, but by making every change to them transparent and traceable.

---

## **The Pattern**

Walk any layer. Pick any problem. The root cause is the same: actors need to coordinate, trust is required, and the current mechanism for trust is either "take my word for it" or "trust the platform."

Neither scales. Neither survives bad actors. Neither is verifiable.

The event graph replaces both with something structural. Not a platform that mediates trust — a data structure that makes trust verifiable. Not a third party that holds the record — a cryptographic chain that is the record, independently auditable by anyone.

This isn't a pitch for mind-zero as a product. It's an observation about what the primitives reveal when you walk them honestly. The framework wasn't designed to map onto marketplaces and social networks and supply chains. It was derived from first principles about coordination and trust. But coordination and trust are the substrate of everything humans build together. So the framework maps onto everything humans build together.

That's either a sign that the primitives are genuinely fundamental, or a sign that we've built a hammer and everything looks like a nail. I genuinely don't know which. But I know that the problems listed above are real, the current solutions are inadequate, and the architecture works.

The code is open source. If any of these problems are yours, the tools to solve them are available right now.

---

## **What This Series Has Been**

Six posts in one day. A late-night question became 20 primitives. Twenty became 44 through an accident. Forty-four became 200 through emergence. The code implements them as working software. The Pentagon proved the problem is urgent. The moral ledger showed what's at stake philosophically. And now the walk through the layers shows how far the architecture reaches.

I didn't plan any of this as a series. I planned it as a Saturday in bed with a hangover and a conversation with Claude. The series emerged the way the primitives emerged — each piece revealing the next, each answer raising a new question, until the shape of the whole thing became visible.

A system designed to be self-expanding expanded itself. A series designed to be five posts became six. Maybe that's the architecture's final demonstration: things that are built on sound primitives grow beyond their original scope. Not because you push them. Because the foundations support more than you expected.

Thanks for reading. The code is at [github.com/mattxo/mind-zero-five]. Come build something.

---

*This is Post 6 of a series on Transpara, mind-zero, and the architecture of accountable AI. Post 1: [20 Primitives and a Late Night] Post 2: [From 44 to 200] Post 3: [The Architecture of Accountable AI] Post 4: [The Pentagon Just Proved Why AI Needs a Consent Layer] Post 5: [The Moral Ledger] The code is open source: [github.com/mattxo/mind-zero-five] Matt Searles is the founder of Transpara. Claude is an AI made by Anthropic. They built this together.*
