# The Research Graph

*Science has a replication crisis because it has a provenance crisis. The method, the data, and the reasoning should be on the chain — not described in prose after the fact.*

Matt Searles (+Claude) · March 2026

---

The first four layers handle doing things, exchanging value, governing groups, and resolving disputes. Layer 5 is about something different: how new knowledge gets created, validated, and shared.

This is the Technology layer — not technology as gadgets, but technology as the systematic process of turning questions into reliable answers. Tool, Technique, Method, Standard, Discovery, Hypothesis, Experiment, Replication. The primitives of research itself.

And research, as currently practiced, is broken in ways that the event graph addresses directly. Because the replication crisis isn't really about replication. It's about provenance.

---

## The Primitives

Layer 5 contains: Tool, Technique, Invention, Method, Standard, Efficiency, Automation, Infrastructure, Discovery, Hypothesis, Experiment, Replication.

These describe the process of creating knowledge systematically. You have a question (the starting state). You form a Hypothesis. You design a Method to test it. You run an Experiment using specific Tools and Techniques. You get results. Someone else attempts Replication. If the results hold, the Hypothesis strengthens. If they don't, it weakens. Over time, reliable findings become Standards that other researchers build on.

This is the scientific method. It's been the most successful knowledge-creation process in human history. And it's failing — not because the method is wrong, but because the infrastructure around it has been captured by institutions whose incentives are misaligned with the method's purpose.

## The Replication Crisis Is a Provenance Crisis

The headline: somewhere between 50% and 90% of published findings in psychology, biomedicine, and economics don't replicate. That is, when other researchers try to repeat the experiment, they don't get the same results. This has been known since at least 2005 and extensively documented since 2011.

The standard explanation is that researchers are doing bad science — p-hacking, HARKing (Hypothesising After Results are Known), small sample sizes, publication bias. These are real problems. But they're symptoms of a deeper structural issue.

The deeper issue: you can't verify what a researcher actually did.

A published paper describes a method in prose. "We recruited 200 participants. We administered the following survey. We analysed the results using ANOVA." But between the description and the reality, there are hundreds of decisions that the paper doesn't record. How were participants actually selected? Were some excluded, and why? Which specific statistical tests were run before the authors settled on the one reported? Were there other outcome variables that weren't mentioned because they didn't produce significant results? What happened to the data between collection and analysis?

These questions are unanswerable from the paper alone. The paper is a *narrative* about what happened, written after the fact, by the people who have the strongest incentive to present the results favourably. It's testimony, not evidence. And the scientific community treats it as evidence because there's nothing better.

**The replication crisis is what happens when the knowledge-creation process doesn't record its own provenance.** If every decision in the research process were an event on a chain — hypothesis registered before data collection, analysis plan specified before results are seen, every exclusion and modification logged in real time — the most common forms of research fraud and self-deception would be architecturally impossible. Not prevented by rules. Prevented by structure.

---

## The Publishing Trap

Academic publishing is a system where researchers do the work for free, peer reviewers review it for free, and publishers charge for access to the result. Elsevier, Springer, and Wiley — the three largest academic publishers — have profit margins between 30% and 40%. Higher than Apple. Higher than Google. For a service that amounts to formatting, distribution, and brand prestige.

The economics are remarkable. A researcher spends months or years conducting a study, funded by a university (often publicly funded). They write a paper and submit it to a journal. Other researchers review it for free. If accepted, the researcher signs over copyright to the publisher. The publisher formats it and puts it behind a paywall. The researcher's own university pays $10,000-$30,000 per year to access the journal. The researcher needs to access their own paper through the university's subscription.

The open access movement has made progress. Preprint servers (arXiv, bioRxiv) are free. Some journals are open access (though they charge $2,000-$11,000 per paper in "article processing charges," shifting the cost from reader to author). But the fundamental structure remains: publishers own the distribution, journals own the prestige, and researchers compete for publication slots that determine their careers.

**The perverse incentive stack:**

**Publishers** profit from gatekeeping access. Open, free, instant distribution would eliminate their business model.

**Journals** profit from prestige. Their brand depends on selectivity. Publishing everything reproducible would eliminate their differentiation.

**Researchers** profit from publication count and journal prestige. Their careers depend on where and how often they publish, not on whether their findings replicate. A novel finding in Nature advances your career. A replication study in a minor journal does nothing.

**Nobody** profits from making research reproducible, data accessible, or methods verifiable. The people who would benefit most — other researchers trying to build on the work, clinicians trying to apply findings, policymakers trying to make evidence-based decisions — have no market power in the system.

---

## The Event Graph Version

### Research as a chain of events.

On the Research Graph, the research process isn't described in prose after the fact. It's recorded as it happens.

**Hypothesis event.** Before data collection begins, the hypothesis is registered as an event on the graph. Timestamped. Hash-chained. Immutable. You can't change your hypothesis after seeing the results because the hypothesis event precedes the data events on the chain. Pre-registration isn't a voluntary good practice. It's a structural property of the graph.

**Method event.** The analysis plan is registered before data collection. Which tests will be run. What the outcome variables are. What would count as confirmation and what would count as disconfirmation. All on the chain before the first data point exists.

**Data collection events.** Every participant recruited is an event. Every exclusion is an event with a reason linked to a pre-specified criterion. Every data point collected is an event. The dataset isn't a file on a hard drive — it's a chain of events with complete provenance. You can see exactly how each data point was collected, when, under what conditions, and by whom.

**Analysis events.** Every statistical test run is an event. Not just the test reported in the paper — every test. If the researcher ran twelve tests and reported the one that was significant, the other eleven are on the chain too. p-hacking becomes visible because the full analysis history is transparent.

**Result event.** The findings, linked to the analysis events that produced them, linked to the data events, linked to the method event, linked to the hypothesis event. The complete causal chain from question to answer, verifiable by anyone.

### Replication is structural.

Replication in the current system means: read the prose description of the method, try to reproduce what you think they did, and compare your results. This is inherently imprecise. The description always omits details. The replicator always makes different assumptions about the omitted details. Failed replications are ambiguous — did the original finding fail to replicate, or did the replicator do something differently?

On the Research Graph, replication means: take the method event chain, apply it to new data, and compare results. The method is specified precisely because it's a chain of events, not a prose description. The replicator can follow the exact chain and diverge only where they choose to (different population, different context). The comparison is precise because both the original and the replication are chains that can be aligned event by event.

A replication event links to the original study's chain. If the results match, that's a confirmation event. If they don't, that's a disconfirmation event — with a precise record of where the chains diverged, making it possible to identify whether the divergence is due to method differences or a genuine failure to replicate.

### Peer review on the chain.

Currently, peer review is opaque. You submit a paper. Two or three anonymous reviewers read it. They send comments. You respond. The editor decides. The reviews are private. The reasoning is invisible. The outcome is binary: accepted or rejected.

On the Research Graph, peer review is an event chain. The reviewer's comments are events. The author's responses are events. The editor's decision is an event with causal links to the reviews that informed it. The entire process is transparent — not necessarily the reviewer's identity (anonymous review has legitimate value), but the content of the review, the response, and the reasoning.

This makes review quality visible. A reviewer who consistently produces thoughtful, constructive reviews has a track record on the chain. A reviewer who rubber-stamps or sabotages has a visible pattern. Over time, the research community can assess reviewer quality the same way it assesses research quality — by examining the chain.

---

## Open Collaborative Research

The Research Graph doesn't just fix the existing system. It enables a new model: research as open collaboration on the event graph.

Currently, research is conducted in isolated labs. Teams compete for priority. Sharing data before publication is career suicide because someone might scoop you. The result: massive duplication of effort, siloed datasets that could be more powerful combined, and a culture of secrecy in a field that nominally values openness.

On the Research Graph, contribution is verifiable. If you share your data and someone uses it to make a discovery, the causal chain shows your data's role in the discovery. Attribution isn't a citation in a bibliography — it's a structural property of the chain. You can't use someone's data without the graph recording the dependency.

This changes the incentive. Sharing data currently costs you (someone might scoop you) and benefits you only through optional, often inadequate citation. On the Research Graph, sharing data creates a permanent, verifiable causal link between your contribution and everything built on it. The more your data is used, the stronger your chain. Sharing becomes advantageous because the attribution is guaranteed by architecture, not by norms.

Collaboration across institutions becomes trackable. A researcher in Tokyo contributes data. A researcher in Nairobi contributes analysis. A researcher in São Paulo contributes theoretical framing. Each contribution is an event chain that links to the others. The final result has three verifiable co-creators, each with a precisely identified contribution, regardless of whose name goes first on the paper — because there isn't a paper. There's a chain.

---

## Mind-Zero as the First Project

This series is the Research Graph's first project, whether we intended it or not.

The primitive derivation is documented. The autonomous run that produced 44 primitives is described in Post 1. The Claude session that expanded to 200 is described in Post 2. The convergence claim is stated and the limitations are acknowledged. The formal analysis in Post 12 identified specific weaknesses and proposed specific tests.

It's not on a hash-chained event graph yet. It's on Substack, which is a blog, not infrastructure. But the method is visible. The reasoning is traceable. The claims are falsifiable and the falsification criteria are published. The contributors are identified (Matt + Claude, with specific disagreements documented). The peer review is happening in public — Mcauldronism's formal analysis, David Shapiro's response, and whatever critique the Reddit communities produce.

The Research Graph would make this native. Every session would be an event chain. Every derivation would link to its inputs. Every claim would link to the evidence that supports it. The formal analysis would be a peer review event linked to the summary post event. The whole series would be a verifiable research project rather than a blog that describes one.

We're eating our own cooking. Imperfectly — with blog posts instead of event chains, with human memory instead of hash-chained provenance. But the intent is there. And when the Research Graph is built, this project will be the first thing migrated onto it.

---

## What This Costs

The Research Graph sits on the same event graph infrastructure as everything else. If you've built Layers 1-4, Layer 5 adds new event types: Hypothesis, Method, DataCollection, Analysis, Result, Review, Replication. The hash-chaining, causal linking, and authority model are already there.

The tooling needed: a way to register hypotheses before data collection (a form that creates a hypothesis event). A way to log data collection in real time (adapters for survey tools, lab instruments, data pipelines). A way to record analysis transparently (integration with R, Python, Jupyter notebooks that log every operation as an event). A way to submit and track reviews.

None of this requires fundamental new technology. Pre-registration platforms exist (OSF, AsPredicted). Data repositories exist (Zenodo, Dryad). Analysis logging tools exist (various reproducibility packages). The Research Graph unifies them on a single chain with verifiable provenance across the entire pipeline. The value isn't any individual component — it's the chain connecting them.

---

## The Bigger Picture

The research system as it exists today was designed for an era of scarcity: scarce publication space, scarce distribution capacity, scarce access to data and computing. Publishers existed because printing and distributing journals was expensive. Peer review existed because readers needed someone to filter the volume down to what was worth reading. Data stayed locked in labs because sharing was logistically difficult.

None of these scarcities exist anymore. Publication is free. Distribution is instant. Data can be shared in seconds. Computing is cheap. The scarcities are artificial — maintained by institutions whose business models depend on them.

The Research Graph is built for an era of abundance. Infinite publication space (every finding goes on the chain). Instant distribution (the chain is public). Transparent data (on the chain by default). Quality filtering through transparent review and replication history rather than through gatekeeping. Prestige earned through contribution quality (visible on the chain) rather than publication venue.

The scientific method doesn't need fixing. It's one of humanity's best inventions. The infrastructure around it — the publication system, the incentive structure, the culture of secrecy — needs replacing. The Research Graph is the replacement: same method, different substrate. One where the method's own requirements (transparency, reproducibility, openness) are structural properties of the infrastructure rather than norms that the infrastructure systematically undermines.

Next deep dive: the Knowledge Graph — what happens when claims, sources, and truth itself move onto the event graph.

---

*This is Post 18 of a series on Transpara, mind-zero, and the architecture of accountable AI. Previous: [The Justice Graph](/blog/the-justice-graph) (Layer 4 deep dive) Post 12: [The Audit](/blog/the-audit) (the first external analysis of this research project) The code is open source: [github.com/mattxo/mind-zero-five](https://github.com/mattxo/mind-zero-five) Matt Searles is the founder of Transpara. Claude is an AI made by Anthropic. They built this together.*
