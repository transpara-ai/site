# Thirteen Graphs, One Infrastructure

*What needs building, what's broken about what exists, and why the event graph fixes the root cause every time.*

Matt Searles (+Claude) · March 2026

---

In Post 6 we walked the 14 layers of the primitive framework and asked: what breaks in the real world at each layer? The answer was: almost everything, and for the same reason. Multiple actors need to coordinate with verifiable trust, and "trust us" doesn't scale.

This post is the other half. Not what's broken, but what to build. Specifically: thirteen product graphs, all running on the same event graph infrastructure, each addressing a different tier of human coordination — from individuals managing tasks to the species managing its relationship with existence.

But first, the idea that ties it all together.

**Views, Not Products**

These aren't thirteen separate systems. They're thirteen views of the same data.

Imagine a freelancer does a job through the marketplace. That's a Market Graph event. The work itself is tracked on the Work Graph. When the client doesn't pay, that's a Justice Graph event. The community discusses it on the Social Graph. A journalist covers it on the Knowledge Graph. The freelancer's record of completing work despite non-payment strengthens their Identity Graph. The regulatory body takes note on the Ethics Graph.

Same events. Same hash chains. Same causal links. Different primitives foregrounded. Different interfaces built on top. One infrastructure.

This is what makes the event graph fundamentally different from building thirteen startups. You don't need thirteen databases, thirteen user accounts, thirteen trust systems. You need one substrate and thirteen lenses.

The reason this matters isn't architectural elegance. It's that the problems at each layer are currently solved by separate platforms that can't talk to each other, that each extract rent for mediating trust they don't actually provide, and that each have perverse incentives to keep the problems unsolved. The event graph replaces all of them with infrastructure that anyone can build on — including you, if you have a Claude subscription and a weekend.

---

**Why Now**

Two things changed in the last eighteen months that make this buildable by individuals rather than requiring a company.

First, AI agents got good enough to be real participants in event graphs. Claude, GPT-4, Gemma — any of these can operate as a node in a workflow, make decisions, explain their reasoning, and have that reasoning recorded. Eighteen months ago you needed a team of engineers to build agent orchestration. Now you need a prompt and an API key.

Second, the tools to build event-graph infrastructure are available off the shelf. Hash chains are trivial to implement. Append-only stores are a solved problem. Causal linking is graph database 101. The hard part was never the technology. It was knowing what to build and why. The 200 primitives provide the what. The 14 layers provide the why. The rest is engineering.

I'm building the Work Graph this week, deploying it at a real company to replace hundreds of legacy apps. But the point of publishing this isn't to recruit users. It's to describe infrastructure that the world needs, that anyone can build, and that matters more than who builds it.

---

### **INDIVIDUAL TIER**

**LAYERS 1–3: INDIVIDUAL**

**LAYER 1 — WORK GRAPH**

**How Things Get Done**

**THE PRIMITIVES**

Agency, Observer, Participant, Actor, Action, Decision, Intention, Goal, Plan, Resource, Capacity, Autonomy, Responsibility.

**WHAT EXISTS**

Jira. Asana. Monday. Linear. Trello. Notion. ClickUp. The task management market is worth $4.3 billion and growing. Every company uses something. Most use several things that don't talk to each other.

**WHY IT'S BROKEN**

Every one of these tools is a *representation* of work, not a *record* of it. You create a ticket. Someone does something. They update the ticket. Maybe. If they remember. The ticket says "Done" — but there's no verifiable chain linking the ticket to what was actually done, by whom, when, or how. The gap between the tool and reality is filled by humans remembering to update things, which they reliably don't.

The perverse incentive: these platforms profit from seat licenses. They need your whole team on the platform. They don't need your work to actually get done more efficiently — they need you to *believe* it's getting done more efficiently, which is a different thing. A tool that genuinely solved coordination would reduce the number of seats you need, which reduces their revenue.

**THE EVENT GRAPH VERSION**

Work is events, not tickets. When an agent writes code, that's an event. When a human reviews it, that's an event. When the build passes, that's an event. Each event is hash-chained to its cause and its outcome. The "task" isn't a separate object that someone has to remember to update — it's the chain of events itself. You can walk backwards from any outcome to every decision that produced it. Humans and AI agents are both nodes in the same graph, subject to the same authority model and the same accountability.

A solo founder with the Work Graph and a Claude subscription gets the coordination capabilities of a 50-person company. The AI agents do the work. The event graph proves it happened. The authority model ensures the human stays in the loop where it matters.

**LAYER 2 — MARKET GRAPH**

**How Value Gets Exchanged**

**THE PRIMITIVES**

Offer, Acceptance, Obligation, Reciprocity, Property, Contract, Debt, Gift, Competition, Cooperation, Scarcity, Surplus.

**WHAT EXISTS**

eBay. Amazon Marketplace. Uber. Fiverr. Upwork. Airbnb. Every gig economy platform. Every two-sided marketplace.

**WHY IT'S BROKEN**

Every marketplace extracts rent for mediating trust. Uber takes 25-30% for connecting a driver to a rider. Upwork takes 10-20% for connecting a freelancer to a client. Airbnb takes 14-20% combined from host and guest. What are they actually providing for that cut? Two things: a matching algorithm (which AI now handles trivially) and a trust layer (reviews, ratings, dispute resolution). The trust layer is where the money is, and it's the weakest part.

Reviews are gamed. Ratings are inflated (4.7 is "average" on Airbnb). Dispute resolution favours the platform's revenue over either party's interests. And the platform owns the relationship — if Uber bans a driver, that driver loses their entire customer base and reputation overnight, because the reputation isn't portable.

The perverse incentive: platforms profit from being the *only* place buyers and sellers can trust each other. If trust were portable and verifiable independent of the platform, the platform would lose its moat. So they ensure it isn't.

**THE EVENT GRAPH VERSION**

Every offer, acceptance, delivery, and payment is an event on the graph with full causal provenance. Both parties can verify the complete chain. Reputation isn't a star rating owned by a platform — it's a hash-chained history of actual transactions, portable to any marketplace that reads the graph. Escrow is embedded in the event structure, not held by a third party. Disputes are resolvable by walking the chain, not by appealing to a support team incentivised to close tickets.

The matching algorithm is commoditised — any LLM can match buyers to sellers. What's not commoditised is the trust infrastructure. That's what the event graph provides as a public good rather than a platform monopoly.

**LAYER 3 — SOCIAL GRAPH**

**How People Connect and Self-Govern**

**THE PRIMITIVES**

Membership, Role, Norm, Status, Consent, Due Process, Legitimacy, Sanction, Commons, Public Good, Free Rider, Social Contract.

**WHAT EXISTS**

Facebook. Instagram. Twitter/X. Reddit. Discord. TikTok. LinkedIn. Every social network.

**WHY IT'S BROKEN**

Social networks are the most successful misalignment in technology history. They optimise for engagement — time on platform, ad impressions, content that provokes reaction. The user's actual social needs (connection, belonging, support, information from trusted sources) are instrumentalised to serve the advertising model. The algorithm decides what you see, and you can't see how it decides. Your social graph — the actual map of your relationships — is the platform's most valuable asset, and you don't own it. You can't export it. You can't take it somewhere else. If the platform disappears or changes its rules, your social infrastructure disappears with it.

The perverse incentive: engagement-maximisation and user wellbeing are inversely correlated. Content that makes people anxious, angry, or envious drives more engagement than content that makes them satisfied. The platform profits from your dissatisfaction. This isn't a conspiracy — it's a structural feature of advertising-funded social networks.

**THE EVENT GRAPH VERSION**

Your social graph is *yours*. It's an event graph of your connections, interactions, and relationships — portable, verifiable, and under your control. The platform provides the interface; the infrastructure is independent. If you don't like the platform, you take your graph to another one. No lock-in. No data hostage.

Governance primitives are baked in rather than absent. Groups have explicit norms (not just "community guidelines" that the platform enforces selectively). Moderation decisions are events on the graph — traceable, auditable, contestable. Consent is structural: you control what you share, who you share it with, and you can verify who's seen it because access is an event.

The feed is not an opaque algorithm. It's a query on your event graph with visible parameters. You can see why you're being shown something. You can change the query. The infrastructure serves the user because the user owns the infrastructure.

---

### **INSTITUTIONAL TIER**

**LAYERS 4–6: INSTITUTIONAL**

**LAYER 4 — JUSTICE GRAPH**

**How Disputes Resolve and Agreements Hold**

**THE PRIMITIVES**

Sovereignty, Authority, Law, Rights, Adjudication, Punishment, Restitution, Precedent, Jurisdiction, Due Process, Evidence, Testimony.

**WHAT EXISTS**

Courts. Arbitration services. Ombudsmen. Regulatory bodies. Small claims courts. Online dispute resolution platforms like Kleros (crypto-based) and Modria. Contract management software.

**WHY IT'S BROKEN**

Justice is slow and expensive because *assembling the evidence takes forever*. The vast majority of legal cost isn't adjudication — it's discovery. Finding out what happened. Collecting documents. Taking depositions. Reconstructing a chain of events that nobody recorded properly in the first place. The legal system is an incredibly expensive mechanism for reconstructing history that should have been recorded as it happened.

Beyond cost, access is the deeper problem. The median civil case costs more to litigate than it's worth. If someone owes you $5,000 and won't pay, you effectively have no recourse — the cost of enforcing the claim exceeds the claim itself. This means that for most people, in most disputes, there is no justice system. There's a justice system for people who can afford lawyers and have disputes large enough to justify the cost. Everyone else just absorbs the loss.

The perverse incentive: the legal profession profits from complexity. Simplifying dispute resolution reduces billable hours. Making evidence self-assembling reduces the need for discovery. Making contracts self-enforcing reduces the need for litigation. The people who would need to implement these changes are the people whose livelihood depends on the current system's inefficiency.

**THE EVENT GRAPH VERSION**

If the work was done on the Work Graph, the payment was on the Market Graph, and the agreement was recorded as events — then when a dispute arises, the evidence already exists. The complete causal chain is there. The Justice Graph doesn't need to *find* the evidence. It needs to *adjudicate* it. That cuts 80% of the cost and time out of dispute resolution.

Start small: disputes arising from events already on the graph. A freelancer completed work (Work Graph events), client didn't pay (Market Graph breach event), the chain is clear, an AI arbitrator can assess the evidence and propose a resolution. Scale up: any agreement between two parties that's recorded on the event graph becomes enforceable without a lawyer, because the evidence is tamper-proof and the chain is complete.

This isn't replacing judges. It's replacing the $200-billion-a-year evidence-assembly industry that exists because we don't record things properly in the first place.

**LAYER 5 — RESEARCH GRAPH**

**How Knowledge Gets Created and Validated**

**THE PRIMITIVES**

Tool, Technique, Invention, Method, Standard, Efficiency, Automation, Infrastructure, Discovery, Hypothesis, Experiment, Replication.

**WHAT EXISTS**

Academic journals (Elsevier, Springer, Wiley). Preprint servers (arXiv, bioRxiv). Google Scholar. ResearchGate. Lab management software. Citation databases.

**WHY IT'S BROKEN**

The replication crisis is the headline, but the structural problem goes deeper. Academic publishing is a system where: researchers do the work for free, peer reviewers review it for free, and publishers charge $30 per article to access the result — or $10,000+ per year for institutional subscriptions. The publishers' contribution is formatting and distribution, both of which are essentially free in 2026. Elsevier's profit margin is ~37% — higher than Apple's.

But the business model isn't even the worst part. The incentive structure rewards *novelty* over *truth*. Positive results get published. Replications don't. Null results don't. The result is a literature systematically biased toward surprising findings, many of which don't replicate. And nobody can verify the underlying data because it's not part of the publication — you get the paper, not the pipeline.

The perverse incentive: publishers profit from gatekeeping access. Journals profit from publishing novel results. Researchers profit from publication count. Nobody profits from making research reproducible, data accessible, or methods verifiable. The people who would benefit most — other researchers trying to build on the work — have no market power in the system.

**THE EVENT GRAPH VERSION**

Every experiment is a chain of events: hypothesis, method, data collection, analysis, result. Each step is hash-chained to the previous one. The data is on the graph, not locked in a researcher's hard drive. The method is executable, not described in prose. Replication isn't a separate study — it's running the same event chain with new inputs and comparing outputs. Peer review is an event too: who reviewed it, what they said, how the authors responded, all on the chain.

The Research Graph doesn't just fix publishing. It fixes the *process* of research by making it natively transparent, reproducible, and verifiable. And it enables a new model: open collaborative research where anyone can contribute, build on others' work, and get verifiable attribution for their contributions through the causal chain.

Mind-zero itself is the first project on the Research Graph. The primitive derivation, the consciousness survey, the evolutionary analysis — all conducted openly, all documented, all falsifiable. Eating our own cooking.

**LAYER 6 — KNOWLEDGE GRAPH**

**How Information Shows Its Provenance**

**THE PRIMITIVES**

Symbol, Language, Encoding, Record, Channel, Copy, Data, Computation, Algorithm, Noise, Entropy, Measurement, Knowledge, Model, Abstraction.

**WHAT EXISTS**

Reuters. AP. BBC. CNN. The New York Times. Fox News. Substack. Medium. Twitter/X. Wikipedia. Google News. Facebook's news feed. Every news aggregator.

**WHY IT'S BROKEN**

The information layer of human civilisation is in crisis. Not because of any single bad actor, but because the economic model for producing and distributing reliable information has collapsed. Newspapers are dying. Journalists are underpaid and overworked. The platforms that distribute news don't produce it and don't pay for it. The algorithms that decide what you see optimise for engagement, not accuracy. And the tools to produce convincing misinformation — deepfakes, AI-generated text, manipulated media — are now cheaper and faster than the tools to verify anything.

The result: nobody agrees on what's real. The same event produces completely different "records" depending on which channel you consume. Not because journalists are lying (mostly), but because the selection, framing, and distribution of information is controlled by systems with no accountability and no relationship to truth.

The perverse incentive: attention is the currency. Accuracy doesn't drive clicks. Outrage does. Novelty does. Tribal confirmation does. A news ecosystem funded by advertising will always optimise for what gets attention, not what gets things right. Subscription models are better but still incentivise confirming subscribers' existing beliefs.

**THE EVENT GRAPH VERSION**

Every claim has a source. Every source has a causal chain. Every chain is verifiable — not by a fact-checker (who also has biases and incentives), but by the architecture itself. You said this. Here's when. Here's what you were responding to. Here's what evidence you cited. Here's whether that evidence still holds. Here's who challenged it and on what grounds. Here's the current state of the dispute.

Not a truth engine that tells you what to believe. A truth graph that shows you the provenance of any claim and lets you make your own assessment with full visibility into the information supply chain. The infrastructure doesn't arbitrate truth — it makes the chain visible so that arbitration is possible.

This is what content provenance standards like C2PA are reaching for, but they're trying to solve it at the file level (was this image manipulated?) rather than the claim level (is this assertion supported by evidence?). The Knowledge Graph operates at the claim level, which is where trust actually lives.

---

### **CIVILISATIONAL TIER**

**LAYERS 7–10: CIVILISATIONAL**

**LAYER 7 — ETHICS GRAPH**

**How Harm Gets Prevented and Accountability Verified**

**THE PRIMITIVES**

Care, Harm, Dignity, Flourishing, Conscience, Justice, Motive, Moral Status.

**WHAT EXISTS**

Regulatory bodies (SEC, FTC, OFCOM, EU regulators). Ethics review boards. Corporate compliance departments. Trust & Safety teams at tech companies. AI safety organisations. ESG rating agencies.

**WHY IT'S BROKEN**

Regulation is a system where the regulator knows less than the regulated. Companies self-report their compliance. Regulators decide whether to believe them. When violations are found, they're found *after the harm has occurred*, often years later, and the penalties are typically a fraction of the profit generated by the violation. Facebook was fined $5 billion for Cambridge Analytica — roughly 5 weeks of revenue. That's not a penalty. It's a cost of doing business.

AI alignment specifically: the current approach is to train models to be "safe" during development and then hope they stay that way in deployment. But alignment isn't a one-time property — it's an ongoing state. A model that behaves well in testing may behave differently in production when the inputs change. And there's no real-time verification that the model's decisions are aligned with any particular set of values, because the decision-making is opaque.

The perverse incentive: regulators are typically underfunded, understaffed, and often staffed by people who recently worked at (or will soon work at) the companies they regulate. The "revolving door" means the regulator's interests and the regulated's interests are more aligned than either is with the public interest. ESG ratings are paid for by the companies being rated. Ethics review boards have no enforcement power. The entire ethics infrastructure is advisory, not structural.

**THE EVENT GRAPH VERSION**

Compliance isn't self-reported. It's verifiable. Every decision an AI makes is an event on the graph with a causal chain showing what inputs it received, what values informed the decision, what authority approved it, and what the outcome was. The regulator doesn't need to trust the company's report — they can verify the chain independently.

This isn't surveillance. It's accountability infrastructure. The same way financial auditing doesn't mean the government reads every transaction — it means the transactions are recorded in a way that *can* be audited. The Ethics Graph provides the same thing for decisions involving harm, dignity, and care: a verifiable record that the right processes were followed, the right constraints were applied, and the right humans were in the loop.

Trust isn't a star rating. It's derived from behaviour across all the other graphs. Did this entity fulfil its agreements (Market Graph)? Did this employer treat workers with dignity (Work Graph)? Did this community welcome newcomers or harass them (Social Graph)? Did this source's claims hold up over time (Knowledge Graph)? Trust emerges from the chain. Not what people say about you. What the graph shows you did.

**LAYER 8 — IDENTITY GRAPH**

**How Entities Are Verified**

**THE PRIMITIVES**

Self-Concept, Narrative, Reflection, Authenticity, Expression, Growth, Integration, Crisis, Purpose, Aspiration.

**WHAT EXISTS**

Passports. Driver's licenses. Social security numbers. WorldID. OAuth. Google/Apple sign-in. KYC providers. Credit bureaus. LinkedIn. Background check services.

**WHY IT'S BROKEN**

Identity is fragmented, insecure, and you don't own it. Your government issues you identity documents that can be forged. Your bank runs a KYC process that every other bank also runs from scratch. Your professional reputation lives on LinkedIn, which you don't control. Your credit score is calculated by three companies using opaque algorithms on data that's frequently wrong, and correcting errors is deliberately difficult.

Digital identity is worse. You prove who you are by knowing a password (which gets stolen), owning a phone (which gets lost), or showing your face (which gets deepfaked). Each platform maintains its own identity silo. None of them interoperate. And the data they collect about you — to "verify your identity" — becomes their asset, not yours. They sell it, leak it, or lose it to hackers, and you bear the consequences.

The perverse incentive: identity providers profit from being the sole verifier. If your identity were portable and self-sovereign, you wouldn't need each platform to re-verify you. The current fragmentation isn't a technical limitation — it's a business model. Each identity silo is a captive audience.

**THE EVENT GRAPH VERSION**

Identity isn't a document you carry or a profile you curate. It's the hash-chained record of what you've actually done across all the other graphs. Your work history (Work Graph). Your transaction history (Market Graph). Your social connections (Social Graph). Your dispute record (Justice Graph). Your contributions (Research Graph). Your trustworthiness (Ethics Graph).

You own it. It's cryptographically yours. You choose what to share: prove you completed a degree without revealing where, prove you're over 18 without revealing your age, prove you have a track record of honest dealing without exposing individual transactions. Selective disclosure on a verifiable chain.

WorldID is reaching for this with biometrics-first (iris scans). The Identity Graph is behaviour-first — identity that emerges from what you've done, with biometrics as one verification layer among many. Not "prove you have unique eyeballs." Prove you're someone with a history of acting in the world in verifiable ways.

**LAYER 9 — POPULATION GRAPH**

**How Populations Are Understood**

**THE PRIMITIVES**

Bond, Attachment, Intimacy, Attunement, Repair, Grief, Forgiveness, Loyalty, Mutual Constitution.

**WHAT EXISTS**

Census bureaus. Public health systems. Epidemiological models. Demographic surveys. Dating apps. Family courts. Social services. Insurance actuarial tables.

**WHY IT'S BROKEN**

Governments understand their populations through census data collected once a decade, surveys with low response rates, and administrative records that are fragmented across agencies. Public health systems discovered during COVID that they couldn't track disease spread in real time because the data infrastructure didn't exist. Epidemiologists built models on incomplete data with lag times measured in weeks. Economic inequality is measured retrospectively, long after the policies that caused it have become entrenched.

At the individual level: relationships are invisible to the systems that affect them. A family court resolving custody has to reconstruct the relationship history from testimony — he-said-she-said, reconstructed memories, selective presentation. Health systems treating addiction can't see the social connections that sustain or undermine recovery. Social services supporting vulnerable people can't see the network of relationships that constitutes the person's actual support system.

The perverse incentive: governments that understood their populations in real time would have to respond to problems in real time. Delayed data provides political cover — by the time the numbers show a crisis, the responsible politicians have moved on. And the companies that do have real-time population data (Facebook, Google, mobile carriers) don't share it and use it to sell ads, not to inform public policy.

**THE EVENT GRAPH VERSION**

The Population Graph is what the Social Graph looks like at demographic scale, with appropriate privacy protections. Not individual relationship details — aggregate patterns. Migration flows visible in real time. Disease spread traceable through connection patterns (with consent-based, anonymised participation). Economic inequality mapped not as a snapshot but as a dynamic process, with the causal chains that produce it visible.

At individual scale: a relationship has a history on the graph. Not surveillance — a record that both parties contribute to and control. When a dispute reaches the Justice Graph, the evidence of the relationship's actual dynamics exists, rather than being reconstructed from unreliable memory. When someone enters healthcare, their support network is visible (with consent), enabling treatment that accounts for social context.

This is the most sensitive layer. The privacy risks are obvious. But the alternative — that governments fly blind on demographics while tech companies know everything and share nothing — is also a privacy disaster, just one we've normalised.

**LAYER 10 — GOVERNANCE GRAPH**

**How Communities Self-Organise**

**THE PRIMITIVES**

Welcome, Belonging, Solidarity, Voice, Ritual, Practice, Place, Sacred, Tradition, Shared Narrative.

**WHAT EXISTS**

National governments. Local councils. HOAs. Co-ops. Credit unions. Professional associations. Trade unions. Religious congregations. Online communities (Discord, Slack, subreddits). DAOs.

**WHY IT'S BROKEN**

Every community, from a nation-state to a Discord server, faces the same problem: how do you make collective decisions that the community accepts as legitimate? The current answers are either top-down (someone's in charge and everyone else deals with it), or structureless (nobody's in charge and nothing gets decided, or the loudest voice wins).

Formal governance — representative democracy, corporate boards, committee structures — produces decisions that are legitimate in theory but often unrepresentative in practice. The people affected by decisions have no real-time visibility into how those decisions were made. Meeting minutes are incomplete. Voting records are aggregated. The causal chain from "citizens want X" to "policy delivers Y" is opaque and full of intermediaries who have their own interests.

Online community governance is even worse. Discord moderators have unchecked power. Reddit mods are volunteers with no accountability framework. Platform governance is feudal — the owner of the server is the lord, and members are subjects with no verifiable rights. When moderation goes wrong (and it frequently does), there's no record, no appeal process, and no accountability.

The perverse incentive: governance structures benefit incumbents. Those who have power within a governance system can use that power to maintain it. Transparency threatens incumbent power. Accountability threatens incumbent power. So governance systems evolve toward opacity and reduced accountability, not toward transparency — unless the transparency is structural and can't be opted out of.

**THE EVENT GRAPH VERSION**

Every governance decision is an event with causal ancestry. Who proposed it. What information informed it. Who voted and how. What authority was invoked. What the outcome was. What was affected. Not reconstructed from meeting minutes — recorded as it happened, tamper-proof and auditable.

Community membership has explicit terms (Social Contract primitives). Moderation decisions are traceable events that members can see and contest. Rules and their enforcement are on the same graph, so the gap between "what the rules say" and "how the rules are applied" is visible. When rules change, the change is an event with a causal chain showing why it happened, who authorised it, and what precedent it sets.

This is social credit done ethically. Not surveillance-as-control but governance-as-transparency. The community can see itself — its decisions, its patterns, its evolution — and steer accordingly. The graph doesn't tell the community what to do. It shows the community what it's doing, verifiably, so the community can decide if that's what it wants.

---

### **UNIVERSAL TIER**

**LAYERS 11–13: UNIVERSAL**

**LAYER 11 — CULTURE GRAPH**

**How Civilisations Encounter Each Other**

**THE PRIMITIVES**

Translation, Encounter, Dialogue, Pluralism, Syncretism, Interpretation, Critique, Aesthetic, Creativity, Hegemony, Cultural Evolution.

**WHAT EXISTS**

Wikipedia. Google Translate. International media. Cultural exchange programs. UNESCO. Museums. Academic cultural studies. Netflix (yes, seriously — the largest cross-cultural content distributor on earth).

**WHY IT'S BROKEN**

Cross-cultural understanding is mediated by institutions that either flatten difference (global media that homogenises toward Western/English-dominant norms) or exoticise it (tourism, "world music" categories, cultural studies that treats living cultures as objects of academic analysis). The tools for translation are powerful but shallow — Google Translate can convert words between languages but can't convey the conceptual frameworks that give those words meaning in their original context.

Creative attribution across cultures is a mess. When a Western artist samples African music, or an AI model trains on Japanese art styles, or a pharmaceutical company patents a molecule derived from indigenous knowledge — the causal chain of influence is real but unrecorded. There's no infrastructure for tracking how ideas, styles, and knowledge flow between cultures, which means there's no infrastructure for fair compensation or attribution.

The perverse incentive: cultural hegemony benefits the dominant culture. English-language platforms define the norms. Hollywood defines the narratives. Western epistemology defines what counts as "knowledge." Systems that genuinely enabled cross-cultural dialogue on equal terms would redistribute cultural influence away from its current centres. Those centres have no incentive to build such systems.

**THE EVENT GRAPH VERSION**

The Culture Graph tracks the flow of ideas, styles, and knowledge across cultural boundaries with verifiable attribution. When a creative work draws on another tradition, the causal chain is visible. When knowledge transfers across communities, the provenance is recorded. Not as a surveillance system — as a *credit* system. Attribution that's structural rather than optional.

Translation on the Culture Graph isn't just words — it's mapping between conceptual frameworks. The primitives themselves provide a shared vocabulary: every culture has concepts that map onto Agency, Exchange, Society, Justice, Ethics, Identity. The mapping isn't perfect, but it's a starting point for genuine dialogue rather than superficial translation. "What do you mean by justice?" can be answered by showing which primitives a culture weights most heavily, and how those differ from the asker's own weighting.

This is the furthest-out layer in terms of buildability, but the creative attribution piece is urgent now because of AI training data. Every AI model trained on human-created content is a cultural transfer event. The Culture Graph makes those transfers visible and compensable.

**LAYER 12 — META GRAPH**

**The System's Nervous System**

**THE PRIMITIVES**

Self-Organization, Feedback, Complexity, Recursion, Autopoiesis, Co-Evolution, Phase Transition, Paradox, Incompleteness.

**WHAT EXISTS**

Business intelligence platforms. System monitoring tools (Datadog, Grafana). Algorithmic trading systems. Social media analytics. Google Analytics. Epidemiological modelling. Climate models.

**WHY IT'S BROKEN**

Every complex system — a company, a market, a society — generates emergent behaviours that can't be predicted from the behaviour of individual components. A market crash isn't caused by any single trade. A social media pile-on isn't caused by any single post. A pandemic isn't caused by any single infection. These are system-level phenomena that emerge from the interactions between components.

We're terrible at detecting these emergent patterns until they've already materialised as crises. We monitor individual metrics (stock prices, infection rates, engagement numbers) but rarely the *interactions between* metrics that signal phase transitions. The 2008 financial crisis was detectable in the correlation patterns between mortgage-backed securities months before the crash — but nobody was looking at correlation patterns, because the monitoring tools weren't designed for emergent behaviour.

The perverse incentive: complexity benefits intermediaries. Financial complexity benefits banks. Regulatory complexity benefits lawyers. Technological complexity benefits consultants. The people best positioned to simplify systems profit from their complexity. So systems get more complex, emergent risks increase, and crises become more frequent and more severe.

**THE EVENT GRAPH VERSION**

The Meta Graph is the system watching itself. It monitors all the other graphs for emergent patterns: correlations forming between previously independent event chains, feedback loops amplifying signals, network effects approaching tipping points. It's the infrastructure that detects phase transitions *before* they complete.

This isn't a consumer product. It's the nervous system of the entire platform. When the Market Graph shows correlated behaviour across apparently independent transactions, the Meta Graph flags a potential systemic risk. When the Social Graph shows a cascade pattern forming, the Meta Graph can alert governance structures before it becomes a mob. When the Knowledge Graph shows a misinformation pattern propagating, the Meta Graph traces the amplification chain.

The event graph is uniquely suited for this because it already records causal links. Emergent behaviour detection is fundamentally about finding *unexpected causal connections*. The data structure is the detection mechanism.

**LAYER 13 — EXISTENCE GRAPH**

**How We Relate to Everything That Isn't Us**

**THE PRIMITIVES**

Being, Wonder, Acceptance, Presence, Gratitude, Mystery, Transcendence, Finitude, Groundlessness, Return.

**WHAT EXISTS**

Climate monitoring systems. Biodiversity databases. Environmental regulations. Conservation organisations. SETI. Space agencies. Deep ecology movements. Animal welfare organisations. Carbon credit markets.

**WHY IT'S BROKEN**

Every graph below this one is anthropocentric. Every node is a human or an AI agent. But we share the planet with millions of other species whose interests we systematically ignore — not out of malice, but because they're not on the graph. They have no voice in governance, no representation in markets, no identity in identity systems. They're externalities.

The environmental crisis is, at root, an accounting failure. The economic system doesn't account for ecological costs because ecosystems aren't participants in the market graph. A forest has immense value — carbon storage, biodiversity, water filtration, climate regulation — but that value appears nowhere in any transaction until the forest is cut down and sold as timber. At that point its value enters the market. Before that, it's economically invisible.

The perverse incentive: economic growth as measured requires converting natural capital into financial capital. A standing forest contributes nothing to GDP. A clearcut forest does. The measurement system defines nature as worthless until destroyed. Every effort to "price" ecosystem services runs into the problem that the pricing is done by the same system whose foundational assumptions exclude ecosystem value.

**THE EVENT GRAPH VERSION**

The Existence Graph extends the event graph to non-human systems. Ecosystems are event graphs — food webs are causal chains, species are nodes, extinction is a node going permanently offline. The Existence Graph makes ecological relationships visible in the same infrastructure that tracks human relationships. Not as metaphor — as data.

When a development project destroys habitat, that's an event on the Existence Graph with a causal chain linking it to specific ecological consequences. When a species population declines, the causal factors are traceable. When carbon is sequestered or emitted, the chain is recorded. The ecological cost of economic activity becomes visible in the same graph that records the economic benefit — making genuine cost-benefit analysis possible for the first time.

And further out: the Existence Graph is the protocol for encountering non-human intelligence — whether animal cognition (which we're already ignoring), AI systems (which we're already building), or extraterrestrial intelligence (which we're not ready for). The question is whether the event graph generalises. If the primitives are genuinely universal — not just projections of human cognition — then any coordinating intelligence should be able to join the graph. That's a testable claim.

---

**The Pattern**

Walk any layer. The same structure repeats.

There's a coordination problem. Humans built an institution or platform to solve it. That institution has a business model or power structure that requires the problem to remain partially unsolved. The harder the institution works to maintain itself, the more it optimises for its own survival rather than for solving the underlying problem. The gap between "what the system says it does" and "what the system actually does" grows until the system fails or gets replaced.

The event graph breaks this pattern because it's infrastructure, not an institution. It doesn't have a business model. It doesn't need a revenue stream to justify its existence. It doesn't have employees whose livelihoods depend on the problem remaining unsolved. It's a data structure. It records what happened, who did it, and why, in a way that's tamper-proof and independently verifiable.

That's it. That's the whole thesis. Every coordination problem at every layer of human civilisation reduces to: we need to know what happened, and we need the record to be trustworthy. The event graph provides both.

**The Buildable Piece**

Not all of this is buildable tomorrow. The Existence Graph is a decade out, at least. The Culture Graph and Governance Graph require adoption at scales that no individual can catalyse.

But the first three layers — Work, Market, Social — are buildable now, by individuals, with tools that exist today. A Work Graph on top of Claude's API with an event store and hash chains. A Market Graph that uses the same infrastructure to track transactions with verifiable provenance. A Social Graph that gives users ownership of their own connection data.

Each layer bootstraps the next. If you have a Work Graph, you can add marketplace features (Layer 2 rides on Layer 1). If you have marketplace features, you can add social features (Layer 3 rides on Layers 1 and 2). If you have all three, disputes become resolvable with evidence already on the graph (Layer 4 rides on Layers 1-3). Each new layer is incremental, not revolutionary.

I'm starting at Layer 1 this week, deploying at a company that needs to replace hundreds of legacy apps with a unified system. If it works, I'll know because the event graph will show it working. If it fails, I'll know because the event graph will show where it failed and why.

That's the difference. Not "trust me, it's working." Check the chain.

---

*This is Post 11 of a series on Transpara, mind-zero, and the architecture of accountable AI. Previous: The Map So Far (summary of Posts 1-10) Post 6: Fourteen Layers, A Hundred Problems (the problem counterpart to this post) Post 3: The Architecture of Accountable AI (the technical deep-dive) The code is open source: github.com/mattxo/mind-zero-five Matt Searles is the founder of Transpara. Claude is an AI made by Anthropic. They built this together.*
