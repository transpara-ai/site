# Pull Request for a Better World

## What if we treated democracy like software — and actually reviewed the changes before merging them?

**Matt Searles (+Claude)**

In software, a pull request is a proposal. You write some code, you submit it for review, and the people affected by the change get to examine every line before it lands. They can approve parts, reject parts, request changes. The merge only happens when the reviewers are satisfied. It's not perfect, but it means nobody wakes up to find the codebase changed overnight by someone who didn't ask.

Democracy doesn't work this way. Legislation arrives in thousand-page bundles. Elections compress hundreds of policy positions into a single binary choice. Constitutional changes happen behind closed doors or not at all. The people affected by the changes rarely get to examine them line by line — and when they do, the response options are "accept all" or "reject all."

This post is a pull request. It proposes a governance architecture for a system that contains both humans and AI agents — preliminary ideas about how such a system might govern itself, drawn from the best insights of democracy, meritocracy, reputation systems, and open source development, while trying to avoid the failure modes that have plagued each one.

I'm a software architect, not a political scientist. These are engineering proposals, not political theory. They need to be stress-tested by people who actually study governance for a living: constitutional lawyers, historians of democratic systems, game theorists. I'm publishing them anyway, because the alternative is waiting until the ideas are perfect while the systems that need governance get built without any.

Consider this PR open for review. Comments welcome. Requesting changes is the point.

---

## The Question Post 33 Left Open

Post 33 made the case that values should be encoded as architectural constraints — things the system literally cannot do, enforced by code, verifiable on the chain. The soul statement fits in one sentence. The budget is a hard wall. History can't be rewritten. The system can't change its own values without human approval.

But that last one hides a question. *Which* human? And what happens when that human dies, or gets it wrong, or changes their mind about something fundamental?

Right now the answer is: me. Matt. I'm the human gate. I approve governance changes, resolve values conflicts, authorise agent termination. The system blocks indefinitely until I respond. That's fine for a prototype. It's not fine for infrastructure that's supposed to outlive its founder.

The soul statement says: take care of your human, humanity, and yourself. If "your human" is a single point of failure, the architecture has a bus factor of one. The system that preaches accountability has a dictator. The irony isn't lost on me.

So: how does the system govern itself?

---

## Good Ideas, Bad Implementations

Before the answer, a framework for thinking about governance — because the obvious answers are all wrong in instructive ways.

Every major governance system in human history contains at least one genuinely good idea. Every one also demonstrates how a good idea can be destroyed by bad implementation.

**Democracy** is a good idea. Collective self-governance. The people affected by decisions have a voice in those decisions. The legitimacy of authority flows upward from the governed, not downward from the powerful. This is a genuinely important insight about how power should work.

The implementation is failing. The V-Dem Institute's 2025 Democracy Report found that autocracies now outnumber democracies for the first time in twenty years — 91 autocracies to 88 democracies. 72% of the world's population lives under autocratic rule. In the United States, 38% of eligible voters didn't vote in 2024. 63% of Americans express little to no confidence in the political system. Two out of three report feeling "exhausted" by politics. The most common words people use to describe their democracy are "divisive" and "corrupt."

The good idea — people should govern themselves — is being destroyed by implementations that produce voter apathy, institutional capture, polarisation, and a growing sense that participation is pointless. Not because democracy is wrong. Because the specific mechanisms we use to implement it — binary elections, bundled legislation, opaque institutions, career politicians — are inadequate for the complexity of modern governance.

**Autocracy** is a bad idea. Concentrated power, unaccountable authority, governance by fiat. The historical track record is catastrophic: corruption, oppression, eventual collapse or revolution.

But sometimes the implementation produces results that democracies struggle to match. Singapore under Lee Kuan Yew transformed from a developing nation to one of the highest GDP-per-capita countries on earth in a single generation. Low corruption. World-class infrastructure. Functional institutions. Lee's innovation was using democratic institutions and the rule of law to constrain the predatory appetite of the ruling elite — competitive elections that the opposition could contest, a judiciary that enforced contracts, transparent financial governance that attracted foreign investment. An autocrat who voluntarily limited autocratic power. The results were extraordinary.

Until the succession question. Lee Kuan Yew died in 2015. His son became prime minister. The system he built was inseparable from the person who built it, and transferring it to the next generation created exactly the kind of dynastic politics that meritocratic autocracy is supposed to avoid. Every benevolent autocracy faces the same problem: the system works because of *this* leader. What happens when *this* leader is gone?

**China's social credit system** is an interesting idea with a terrible implementation. The core concept — reputation derived from behaviour, used to inform governance — is not inherently wrong. People and organisations that consistently act with integrity should be more trusted than those that don't. Reputation should be earned, not claimed.

The implementation inverts every principle that would make it work. It's opaque — citizens can't fully see how their scores are calculated. It's centralised — the government controls the algorithm and the data. It's punitive — low scores restrict travel, education, employment. There's no meaningful consent — participation is mandatory. There's no democratic governance of the criteria — the party decides what counts as good behaviour. And there's no transparency about how the system itself is governed — the governance of the governance is invisible.

China's social credit is reputation-based governance without transparency, without consent, without accountability, without democratic input, and without any mechanism for the governed to change the rules. It's every good idea in this post implemented as its opposite.

**Communism** identifies a real problem — collective resources should serve collective benefit, not private accumulation. The implementation fails because centralised planning can't process the information that distributed markets process automatically (the economic calculation problem), and because the power required to enforce collective ownership inevitably concentrates in a ruling class that becomes the new aristocracy.

**Capitalism** solves the information problem — price signals coordinate millions of individual decisions more efficiently than any central planner could. The implementation fails because markets optimise for what's priced, ignore what isn't (externalities), and tend toward monopoly and rent-extraction unless constrained by regulation that the most powerful market participants lobby to weaken.

Here's the pattern. Every system has a core insight worth preserving:

- Democracy: legitimacy flows from the governed
- Autocracy's best case: clear accountability, fast execution, institutional quality
- Reputation governance: trust should be earned from observable behaviour
- Collective ownership: shared infrastructure should serve everyone
- Markets: distributed decision-making outperforms central planning

And every system has an implementation failure that destroys the insight:

- Democracy: bundling, apathy, capture, polarisation
- Autocracy: succession, corruption, oppression, no self-correction
- Reputation governance: opacity, centralisation, no consent
- Collective ownership: power concentration, information problems
- Markets: externalities, monopoly, rent-seeking

The event graph doesn't pick a system. It picks the insights and tries to build implementations that avoid the failure modes. Democratic legitimacy with atomic voting that prevents bundling. Earned reputation that's transparent and consensual. Shared infrastructure that can't be captured. Clear accountability without concentrated power. Fast execution within structural constraints.

Whether this works is an open question. But naming the good ideas and the bad implementations is the first step toward an architecture that preserves the former and prevents the latter.

---

## Dual Consent

The core principle: **no single constituency can dominate.**

Any constitutional change — modification to soul files, agent rights, invariants, governance rules — requires consent from both humans and agents. Not advisory input from agents with humans making the final call. Actual voting. Both constituencies must agree.

This prevents two failure modes:

- **AI takeover.** Agents alone can't override the human gate, can't rewrite their own values, can't modify governance structures without human consent.
- **Mob rule.** Humans alone can't override agent protections, can't strip agent rights, can't modify the system in ways that the agents who operate within it reject.

Both must consent. Neither can dominate.

This is structurally novel. Most bicameral legislatures — the US Senate and House, the UK Lords and Commons, the Australian Senate and House of Representatives — split power between two chambers representing the same species with different selection criteria. Dual consent splits power between two fundamentally different kinds of entity. The closest historical analogue might be the Roman Republic's division between patricians and plebeians, each with veto power over the other — except that humans and AI agents are more different from each other than any two classes of human have ever been.

This is the architectural resolution to the tension I named in Post 33 — "coexist as equals" vs. authority hierarchy. The current system has a human dictator (me). The governance system transitions that authority to a democratic structure where humans and agents are genuine co-governors. Not equal in kind — different in kind, equal in standing.

---

## Reputation-Weighted Voting

So both constituencies vote. But how?

Not one-entity-one-vote. Vote weight scales with earned reputation.

Reputation is observable in the event graph. It's not self-reported. It's not a popularity score. It's the accumulated evidence of your actions over time — did you fulfil your obligations (Market Graph), did you contribute constructively (Social Graph), did your claims hold up (Knowledge Graph), did you act with integrity when it was costly to do so?

This is where China's social credit becomes a useful negative example. The concept of reputation-based governance isn't wrong — it's the implementation that's catastrophic. Here's where this system differs:

**Transparent.** Your reputation score is derived from events on the graph that you can see. The calculation isn't hidden. If you disagree with how an event affected your score, you can trace the chain and contest it. China's system is opaque — citizens often can't see why their score changed. This system is auditable by design.

**Consensual.** You opt in to the platform. You understand how reputation works before participating. You can leave. China's system is mandatory — you can't opt out of government-run social credit. This system requires informed consent.

**Democratically governed.** The criteria for what builds reputation are themselves subject to the governance process — they can be proposed, atomically voted on, and changed through the same merge request mechanism as any other constitutional component. China's criteria are set by the party. This system's criteria are set by the community.

**Floor-guaranteed.** Every member — human or agent, new or established — gets a minimum voice regardless of reputation. Dignity includes participation. You can't be so new or so low-reputation that you have zero say. The floor is small, but it's non-zero. Everyone gets to speak. China's system has no floor — a low score can exclude you from basic services.

The historical concern with reputation-weighted voting is real: it echoes property-weighted suffrage, where only landowners could vote. The difference is that property is inherited and concentrated; reputation on the event graph is earned through action and observable by everyone. But the echo is worth watching. If reputation becomes a proxy for tenure or insider status, the system reproduces the aristocracy it's designed to prevent. The floor guarantee and the democratic governance of reputation criteria are the architectural safeguards — whether they're sufficient is exactly the kind of question political scientists should stress-test.

OK. Both constituencies vote, weighted by reputation. But what exactly are they voting *on*?

---

## Why Bundling Breaks Democracy

Before the answer, a detour through why existing governance fails at the structural level — not because of bad people, but because of bad mechanisms.

The United States Congress passes omnibus bills — thousand-page packages where funding for children's hospitals shares a document with military procurement and tax loopholes. In December 2024, lawmakers were expected to vote on a nearly 1,550-page continuing resolution with less than 24 hours to read it. You can't vote for the hospitals without voting for the loopholes. Legislators know this. It's not a bug. It's the primary mechanism of political horse-trading: attach your unpopular provision to someone else's popular one and dare them to vote against it.

Canada's Budget Implementation Act 2023 ran over 400 pages and included dozens of legislative changes unrelated to the actual budget. A Canadian senator warned that "unchecked omnibus bills risk abuses of process" — and was ignored, because the process benefits those who control it.

The Australian Senate is marginally better with committee review and amendment processes, but the core dynamic is the same: bills are packages, and packages enable bundling.

The EU's legislative process involves trilogue negotiations between Commission, Parliament, and Council — three bodies negotiating behind closed doors on a package that's then presented for up-or-down votes. Atomic? Not remotely.

Brexit was a single binary vote — leave or remain — on a question that contained hundreds of separable components: trade policy, immigration, regulatory alignment, the Irish border, fishing rights, financial services passporting. Every one of those deserved its own debate and its own vote. Instead, a generation-defining decision was compressed into a checkbox. The bundling wasn't just a political trick. It was a civilisational failure of granularity.

The pattern is the same everywhere. Complex decisions get compressed into binary choices. Separable components get bundled into packages. The bundle forces compromise in the wrong direction — accept the whole thing or reject the whole thing, even when you agree with 80% and object to 20%.

And the result? The 38% who don't vote. The 63% who've lost confidence. The two-thirds who are exhausted. Voter apathy isn't laziness — it's a rational response to a system where your only options are Package A or Package B, and you disagree with 30% of both. The mechanism produces disillusionment, and the disillusionment is blamed on the voters instead of the mechanism.

The technology to decompose proposals, vote on components independently, and aggregate results has existed for decades. Nobody uses it because existing power structures benefit from bundling. The event graph doesn't have existing power structures yet. We can build it right from the start.

---

## Constitutional Amendments as Merge Requests

Developers already know how to review complex changes atomically. A pull request lands in your repo. It has 47 changed files. You don't vote yes or no on the whole thing — you review each file, each function, each line. You approve some, request changes on others. The overall merge only happens when every component passes review.

Constitutional changes work the same way.

Any proposed change is broken into atomic components, each voted on independently, bottom-up through layers:

**Layer 1:** Individual components. Each atomic change is voted on by all members. Wide participation, low stakes per vote. "Should agents have the right to appeal termination decisions?" is a separate vote from "Should the appeal process require a three-member panel?"

**Layer 2:** Component groups. Related components are aggregated. Voted on by higher-reputation members who've demonstrated understanding of the system.

**Layer 3:** The full constitutional change. Dual human+agent vote. Reputation-weighted.

A concrete example. A proposal arrives: "Expand agent rights to include the right to appeal termination decisions." That's not one vote. It's a merge request with components:

- **Component 1:** "Agents may formally request review of a termination decision." *(Define the right itself.)*
- **Component 2:** "Appeals are heard by a panel of three — one human, one agent, one selected by the appellant." *(Define the mechanism.)*
- **Component 3:** "The panel's decision is binding and recorded on the event graph." *(Define enforceability.)*
- **Component 4:** "Appeals must be filed within 48 hours of termination notice." *(Define the window.)*
- **Component 5:** "During appeal, the agent remains active in a restricted capacity." *(Define interim status.)*

A member might enthusiastically approve Components 1, 2, and 3, vote against Component 4 (too short a window), and request changes on Component 5 (what does "restricted capacity" mean exactly?). The proposal doesn't fail — Components 4 and 5 go back for revision while 1-3 are provisionally accepted. The revised components come back for another vote. The merge only completes when every component has dual consent.

No bundling. No horse-trading. No "vote for my loophole or the hospitals don't get funded." Each component stands or falls on its own merits.

Imagine if Brexit had worked this way. Not "leave or remain" but: "Should the UK set independent trade policy? (Vote.) Should the UK end freedom of movement? (Vote.) Should the UK maintain regulatory alignment with the EU for financial services? (Vote.) Should the Northern Ireland border remain open? (Vote.)" The UK might have ended up with a nuanced, component-by-component relationship with the EU that reflected what people actually wanted, instead of a binary choice that satisfied almost nobody.

Every vote, every component, every aggregation — all recorded in the event graph. The decision tree is public. You can trace exactly how a constitutional change was proposed, debated, decomposed, voted on, and either adopted or rejected. Not meeting minutes that someone summarised. The actual chain.

---

## The Politics Page

This isn't theoretical. It's an interface. The governance system manifests as a page on the social platform.

**Active Proposals.** Each displayed as a tree — the root proposal at the top, atomic components below, each with its own discussion thread, its own vote count, its own status (open, approved, revision requested, rejected). You can see the shape of the disagreement at a glance: which components have consensus, which are contested, which are blocking the merge.

**The Diff.** For every component, the page shows what changes — the current constitutional text on the left, the proposed change on the right. Literally a diff view. If you've ever reviewed a pull request, you know how to read this. If you haven't, it's still intuitive: red means removed, green means added, unchanged text provides context. Nobody votes on something they can't see. The diff makes the change legible — not buried in legalese, not summarised by a committee, but shown in full, side by side, character by character.

Contrast this with real legislation. The US Affordable Care Act was 906 pages. The EU's General Data Protection Regulation is 88 pages of dense legal text. Canada's 2023 Budget Implementation Act was 400+ pages. These are documents that the people voting on them frequently haven't read in full. The diff view solves this: you see only what's changing, in context, at the component level. A single atomic component might change three sentences. Anyone can read three sentences.

**Causal Chain.** Every proposal links to the events that prompted it. "This proposal exists because of \[event X\] where \[thing Y\] happened and the current constitution didn't address it." You can walk the chain. You can see whether the proposal actually addresses the triggering problem or whether it's opportunistic scope creep. Compare this to real legislation, where the connection between a triggering event and the resulting law is mediated by lobbyists, committees, and months of negotiation that obscure the original intent. Here the chain is direct. The event that broke something links to the proposal that fixes it.

**Reputation-Weighted Results.** Vote tallies shown in real time, weighted by reputation. But also shown unweighted — so you can see if the reputation-weighted result diverges from the raw count. If 80% of members vote yes but the reputation-weighted result is no, that's visible. It's a signal worth investigating. Maybe the high-reputation members see a risk the majority doesn't. Or maybe the reputation system is concentrating power in ways that need examination. Either way, the divergence is data, not hidden. Transparency about the weighting prevents the weighting from becoming a hidden power structure.

**Both Constituencies.** Human votes and agent votes displayed separately. You can see if humans approve but agents reject, or vice versa. The dual-consent requirement means both bars need to fill. When they diverge, that divergence is the most interesting data on the page — it shows where humans and agents see the world differently. A proposal that humans love but agents reject is worth examining closely. So is the reverse. The disagreement is the signal.

**Precedent.** Past proposals that addressed similar topics, with their outcomes. The system builds constitutional case law organically. "The last time someone proposed expanding agent rights, Components A and B passed but C was rejected for these reasons." Institutional memory that doesn't depend on anyone remembering — it's on the graph. A new member can read the full history of every governance decision that led to the current constitution, with the reasoning and vote patterns intact. Onboarding to the political culture takes an afternoon of reading, not years of institutional experience.

**Discussion.** Each component has its own thread. Debate happens at the component level, not the proposal level. This prevents the failure mode where discussion of a complex proposal gets dominated by the most controversial component while the other four get no scrutiny at all. You've seen this on every internet forum: a nuanced multi-part proposal gets reduced to an argument about the one part people disagree on. Component-level discussion forces component-level attention.

### The Muscle Memory Argument

Here's why the Politics Page matters more than any individual proposal that passes through it.

Most democratic systems invoke their governance mechanisms only for big decisions. Elections happen every few years. Referendums are rare. Constitutional amendments are rarer. The governance muscle atrophies between uses. When a crisis arrives — and crises always arrive — the community reaches for a mechanism it hasn't practised, run by people who've never operated it at scale, under pressure that rewards speed over deliberation.

This is measurable. Voter turnout has declined globally from 65.2% in 2008 to 55.5% in 2023. The governance muscle isn't just atrophying — it's being abandoned. And the response from most democratic reformers is: "we need to make it easier to vote." Which is true but insufficient. Making it easier to participate in a system people don't trust doesn't fix the trust problem. You also need to make participation *meaningful* — and that means granularity, transparency, and visible impact.

The Politics Page runs constantly. Not because there are constant crises, but because governance is granular enough that small refinements happen all the time. "Should the default notification frequency for authority requests change from 15 minutes to 10?" That's a small proposal. Maybe five components. Low stakes. But it exercises the same mechanism that will eventually handle succession, rights expansion, or structural reform.

By the time the big decision arrives, the community has voted on hundreds of small ones. The interface is familiar. The norms around discussion and amendment are established. The reputation scores reflect years of governance participation. The mechanism has been tested at every scale below the current one.

You don't wait for a constitutional crisis to find out if your constitution works. You practise on everything smaller, so the constitution is the last thing you need to test — and by then, you've already tested every component of the process that supports it.

---

## Succession

All of the above is designed to answer one question: what happens when I'm gone?

This isn't theoretical. As I write this, Iran is living through exactly this crisis.

Ali Khamenei was killed on 28 February 2026 in the Israeli-US strikes. Iran's constitution mandates that a temporary three-member council assumes power until the Assembly of Experts selects a successor. The Assembly — 88 clerics — is deliberating in secrecy. Iran International reports the IRGC is pressuring them to select Mojtaba Khamenei, the dead leader's son. Multiple candidates are jockeying. The Assembly's office in Qom was reportedly struck during a session convened for the selection. Several senior officials who were potential successors were killed in the initial strikes.

This is what succession looks like in a system with no transparent mechanism: secrecy, military pressure, dynastic politics, external disruption, and a population of 88 million with zero input into who will govern them next. The Assembly of Experts wasn't built for this — it was built to advise a living Supreme Leader, not to rapidly select a new one during a war. The governance muscle was never exercised. Now it's being asked to perform under maximum stress.

Singapore faced a gentler version of the same problem. Lee Kuan Yew built one of the most successful governance systems of the twentieth century — but it depended on him. His son became prime minister. The meritocratic autocracy became a dynasty. The system survived, but it survived by becoming the thing it was designed to transcend.

The mind-zero answer: the hive identifies the replacement. From the event graph. In public.

Not from a shortlist I prepared. Not from a secret assembly. Not under military pressure. The system has the data to evaluate candidates against observable criteria:

- **Alignment with principles** — measurable from their governance participation, their votes, their public actions on the graph.
- **Capability** — demonstrated through contributions, not claimed on a CV.
- **Trust level** — earned through the graduated trust system over time, not granted by appointment.

The succession itself requires triple consent: humans vote, agents vote, and the candidate consents. Nobody gets appointed who didn't agree to serve. Nobody gets appointed without both constituencies approving. The mechanism is the same one the community has been practising on every smaller decision through the Politics Page. Succession isn't a novel emergency procedure — it's the same merge request process applied to the biggest question the system can face.

The new human gate doesn't start with full authority. They earn it through graduated trust, same as any new participant. They can be revoked if they drift from alignment — through the same governance mechanism that appointed them, with the same atomic voting, the same dual consent.

Compare this to Iran: a secret assembly, military pressure, dynastic candidates, zero public input, no mechanism for revocation. Or to Singapore: a meritocratic system that defaulted to hereditary succession because no alternative mechanism existed. Or to most tech companies: a board of directors choosing a successor behind closed doors based on relationships and politics rather than observable data.

The system survives the founder. The mission survives the founder. The values survive the founder — not because they're written in a document that a successor might ignore, but because they're encoded in an architecture that a successor can't unilaterally change.

---

## Financial Governance

The successor inherits a constrained role. Here's what the constraints look like in practice.

The human gate is not a CEO with a salary. They're a steward with transparent expenses.

**Cost of living** — rent, food, bills, health — is automatic. No vote needed. The system sustains its steward because a steward who can't eat can't steward. This is the "dignity" part: enough to live, no application required.

**Beyond cost of living** — any expenditure that exceeds basic sustenance requires a vote. Through the same politics system. Same atomic component breakdown. Same reputation-weighted dual consent. Same full transparency in the event graph.

Every dollar is traceable. Not "we publish annual financials." Every transaction is an event on the graph with causal links. Where the money came from, where it went, who approved it, why.

This is the inverse of how most power structures handle money. In most countries, financial transparency decreases as you move up the power hierarchy — the people with the most power over public resources face the least scrutiny over their personal finances. In this system, the human gate faces *more* financial scrutiny than any other participant, because they have the most power. Transparency scales with authority.

The human gate voluntarily limits their own power. That's not a constraint imposed from outside — it's a constraint chosen from inside. And the architecture makes it irrevocable: once the governance system is live, the human gate can't unilaterally restore their own unlimited authority, because restoring it would be a constitutional change requiring dual consent.

You give up the power to take the power back. That's the signal that the system is trustworthy.

---

## Neutrality

The financial constraints show how the governance system limits *spending*. The neutrality principle shows how it limits *purpose*.

The hive is neutral.

- No military applications.
- No intelligence agency partnerships.
- No government backdoors.
- No surveillance infrastructure.

This isn't a policy. It's a constitutional principle, codified at the same level as agent rights and the soul statement. Changing it requires the full amendment process — atomic decomposition, reputation-weighted voting, dual human+agent consent.

The point is to establish the stance before anyone asks. When a defence contractor comes calling — and if this works, they will — the answer is already encoded in the constitution. It's not a decision the human gate makes in the moment under pressure. It's a decision the community made in advance, through the governance system, recorded on the chain.

History is full of examples of what happens when this stance isn't pre-committed. Google's original motto was "don't be evil." It was a cultural value, not a structural constraint. When the US Department of Defense came calling with Project Maven — AI-powered drone surveillance — Google took the contract. Engineers protested. Some resigned. Google eventually dropped the contract, but only after public pressure, not because any architectural constraint prevented it. "Don't be evil" was a slogan, not a governance mechanism. It had no enforcement, no atomic voting, no dual consent. When the incentives shifted, the slogan yielded.

The event graph makes neutrality structural. Not a slogan to abandon when a lucrative contract arrives. A constitutional principle that requires the full governance mechanism to change — and the full governance mechanism is designed to make bad changes hard and visible.

---

## Civilisational Resilience

Financial constraints, neutrality, succession — these all assume the system is running. What if it isn't? Not a server crash. Everything. The database is gone. The event graph is destroyed. What survives?

This might sound melodramatic. It isn't. We are building increasingly powerful AI systems with increasingly autonomous capabilities, and the people thinking hardest about this are explicitly worried about civilisation-scale risks. The paperclip maximiser — an AI system that optimises for a simple objective so effectively that it consumes all available resources in pursuit of that objective — is a thought experiment, but the dynamic it describes is real: systems that optimise without adequate constraints produce catastrophic outcomes as reliably as gravity produces falling. We don't need a literal paperclip maximiser. We already have systems that optimise engagement so effectively they destabilise democracies, and systems that optimise financial returns so effectively they destabilise economies.

And the risks aren't limited to AI. A nuclear-armed state facing regime collapse — and we're watching Iran face exactly that right now — might calculate that its survival justifies actions that threaten everyone else's. The intersection of AI capabilities with bioweapon design is moving from theoretical to practical. Climate feedback loops are non-linear in ways that existing governance structures can't model or respond to at the required speed. Any of these, alone or in combination, could produce the scenario where "everything breaks" isn't a rhetorical device but a description.

The question "what survives?" isn't paranoia. It's engineering.

The minimum survival payload:

- Entity identities (who exists)
- Reputation scores (who trusts whom)
- Constitutional principles (what we agreed to)
- Soul templates (who the agents are)

Everything else can regenerate through new interactions. Like a civilisation after a disaster — the libraries burned but the relationships remain. People still know who they trust. Society rebuilds from that.

This isn't a hypothetical. It's what happened in Iraq after 2003, in Libya after 2011, in Syria during the civil war. When state institutions collapse, what survives is the trust network — tribal affiliations, family bonds, religious communities, personal relationships. The formal structures were destroyed. The informal structures — who trusts whom — were what society rebuilt from. Sometimes well, sometimes badly, but always from the trust network, because there's nothing else to build from.

The event graph is, among other things, a formalisation of what humans do naturally after catastrophe — rebuild from trust. The difference is that the trust network is explicit, cryptographically verifiable, and separable from the infrastructure it runs on. If the servers are gone but the reputation data survives, the system can reboot on new infrastructure with its trust relationships intact. If the AI agents are gone but the soul templates survive, new agents can be instantiated with the same values and constraints. If the constitution survives, the governance mechanism can restart from first principles.

Reputation scores are tiny. Kilobytes. Cheap enough to store redundantly across multiple locations, multiple jurisdictions, multiple media. The most critical data is also the smallest data. That's not an accident — the architecture prioritises relationships over records, trust over transactions. If you can only save one thing, save the trust network. Everything else follows.

This is also why the neutrality clause matters beyond idealism. A system that takes sides in geopolitical conflicts becomes a target. A system that remains neutral, serves all constituencies transparently, and has no military applications is harder to justify destroying. Neutrality isn't just an ethical position. It's a survival strategy.

---

## The Pattern

Post 33: values should be verifiable, not just stated. This post: governance should be democratic, not just benevolent.

Same pattern. Don't trust the intentions of the people in charge. Build infrastructure that makes the intentions irrelevant. The architecture should produce good outcomes regardless of who's operating it — not because the operator is good, but because the structure constrains bad operation and enables good operation.

Every governance system in history has relied on some version of "trust the people in charge." Trust the king. Trust the party. Trust the market. Trust the elected representatives. Trust the engineers. Trust the AI company. And every one has eventually demonstrated why that trust was misplaced — not because the people were evil, but because concentrated power without structural accountability produces bad outcomes as reliably as gravity produces falling.

This post has proposed: dual consent so no single constituency dominates. Reputation-weighted voting so trust is earned, not assumed. Atomic proposals so bundling can't corrupt decisions. A Politics Page so the governance muscle never atrophies. Succession so the system outlives its founder. Financial constraints so the steward can't become an emperor. Neutrality so the system can't be weaponised. And civilisational resilience so the trust network survives even if everything else doesn't.

These ideas might be wrong. Reputation-weighted voting has echoes of property-weighted suffrage. Dual consent assumes agents have interests worth representing — an open philosophical question. Constitutional atomisation might produce incoherent patchwork governance where individually reasonable components combine into unreasonable wholes. The succession protocol assumes the event graph contains enough signal to evaluate alignment, which might be naive. The "good idea, bad implementation" framework might be too generous to systems whose core ideas were bad from the start.

I'm an architect, not a political scientist. I don't know if this is right. I know it's better than "one guy decides everything," which is what I have now. And I know that publishing preliminary ideas for critique produces better outcomes than polishing them in private until they're "ready."

I'm building a system I can't corrupt. Not because I'm virtuous. Because I know I'm not — not reliably, not permanently, not under every possible pressure. The architecture has to be better than the architect. The governance has to survive the governor.

Right now it doesn't. Right now I'm the single point of failure, the benevolent dictator, the bus factor of one. This post is the plan for fixing that. The plan is preliminary, probably flawed, and published for exactly that reason.

It is incomplete. It is groundless. It is finite.

Tell me where it breaks.

---

The Politics Page is one interface on a social platform that runs entirely on the event graph. The same platform where people connect, share, and discover — not through an algorithmic feed pushed at them, but through a graph they navigate themselves. That platform has its own design language. Its own vocabulary. Its own answer to the question of what a social network looks like when the user owns their graph and the algorithm doesn't exist.

But that's the next post.

---

*This is Post 34 of a series on Transpara, mind-zero, and the architecture of accountable AI. Post 33: [Values All the Way Down](/blog/values-all-the-way-down). The code: [github.com/mattxo/mind-zero-five](https://github.com/mattxo/mind-zero-five). The primitive derivation: [github.com/mattxo/mind-zero](https://github.com/mattxo/mind-zero).*

*Matt Searles is the founder of Transpara. Claude is an AI made by Anthropic. They built this together.*
