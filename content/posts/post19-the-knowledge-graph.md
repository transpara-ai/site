# The Knowledge Graph

_Nobody agrees on what's real anymore. Not because people are stupid, but because the information layer has no accountability architecture._

Matt Searles (+Claude) · March 2026

---

Layer 5 handles how knowledge gets created. Layer 6 handles something more fundamental: how information gets represented, transmitted, verified, and corrupted.

The Research Graph produces findings. The Knowledge Graph tracks what happens to those findings — and to every other claim — after they enter the information ecosystem. Who said it. When. Based on what evidence. Who challenged it. What survived the challenge. What didn't.

This is the layer that's failing most visibly in 2026, and the one where the consequences of failure are most severe. Because if the information layer is corrupted, every layer above it — ethics, identity, relationship, community, governance, culture — runs on bad inputs. Garbage in, civilisation out.

---

## The Primitives

Layer 6 contains: Symbol, Language, Encoding, Record, Channel, Copy, Data, Computation, Algorithm, Noise, Entropy, Measurement, Knowledge, Model, Abstraction.

These are the primitives of information itself. Not information as "news" — information in the Shannon sense. How do signals get encoded (Encoding)? How do they travel (Channel)? How do they degrade (Noise, Entropy)? How are they stored (Record, Data)? How are they processed (Computation, Algorithm)? How do they become knowledge (Measurement, Knowledge, Model)?

The interesting thing about Layer 6 is that it's perfectly neutral. In the gender analysis from Post 7, it scored zero masculine, zero feminine. Pure computational substrate. Information doesn't care about anything. It just represents.

That neutrality is the problem. Information infrastructure has no built-in preference for truth over falsehood. A lie and a fact are both signals. They both transmit through channels. They both get stored as records. The infrastructure is indifferent to their truth value. If you want information systems that favour truth, you have to build that preference into the architecture. Nobody has.

## The Information Crisis

This section could be a book. I'll keep it to the structural problem.

The information layer of human civilisation runs on a set of institutions — journalism, publishing, academia, broadcasting — that were designed for an era of scarce distribution. There were a limited number of newspapers, a limited number of TV channels, a limited number of journals. Gatekeepers decided what got distributed. The gatekeeping was imperfect — biased, incomplete, sometimes corrupt — but it created a shared informational commons. Most people in a society consumed roughly the same information, which meant they disagreed about interpretation but not about facts.

That era is over. Distribution is free. Anyone can publish anything to everyone instantly. The gatekeepers lost their monopoly. And nothing replaced them.

What replaced gatekeeping was algorithmic curation. Facebook, Google, Twitter, TikTok — they decide what you see, based on what drives engagement. The algorithms have no concept of truth, accuracy, or importance. They have a concept of engagement — and engagement correlates with novelty, outrage, and tribal confirmation, not with accuracy.

The result is a world where the same event produces completely different informational realities depending on which algorithm feeds you. Not different interpretations of the same facts — different facts entirely. Different events reported. Different sources cited. Different contexts provided. The informational commons shattered, and each shard is curated by an algorithm optimised for attention, not truth.

### AI makes it worse.

In 2026, the cost of producing convincing misinformation is approximately zero. AI can generate realistic text, images, audio, and video indistinguishable from human-created content. A single person with a laptop can produce more convincing disinformation in an afternoon than a state propaganda apparatus could produce in a year in 2010.

The tools to produce false information have dramatically outpaced the tools to verify information. We can generate a deepfake in minutes. Debunking it takes days — and the debunking reaches a fraction of the audience that saw the original. The asymmetry is structural. Fabrication is cheap. Verification is expensive. In an information ecosystem optimised for speed and engagement, the cheap option wins.

**The perverse incentive:** Attention is the currency. Accuracy doesn't drive clicks. Outrage does. Novelty does. Confirmation of existing beliefs does. A news ecosystem funded by advertising will always optimise for attention over accuracy because attention is what advertisers pay for. Subscription models are better but still incentivise confirming subscribers' priors — telling readers what they want to hear rather than what they need to know.

Meanwhile, the platforms that distribute the news — Facebook, Google, Twitter — are legally and economically insulated from the accuracy of what they distribute. They're not publishers. They're pipes. If the pipe is full of poison, the pipe isn't liable. The incentive to ensure accuracy is zero.

---

## What's Been Tried

The information crisis isn't new and people aren't ignoring it. Several approaches exist:

### Fact-checking.

Snopes. PolitiFact. Full Fact. These organisations manually verify claims and issue verdicts: true, false, misleading. The work is valuable but fundamentally unscalable. The volume of claims circulating online exceeds the capacity of every fact-checking organisation on earth combined by orders of magnitude. And fact-checkers have their own biases and incentive structures — the selection of what to check is itself an editorial decision. Fact-checking is a human solution to a structural problem, and it doesn't scale.

### Content moderation.

Platforms employ thousands of moderators and deploy AI systems to flag and remove misinformation. But moderation is reactive (the content has already spread before it's caught), inconsistent (different moderators make different calls), opaque (users can't see why content was removed), and biased toward the platform's interests (content that generates revenue gets more lenient treatment). And moderation doesn't address the root cause — it addresses individual pieces of content without touching the infrastructure that produces and distributes them.

### Media literacy.

"Teach people to think critically about information." This is the educational approach. It's valuable in the long term and useless in the short term. You can't educate your way out of a structural problem. Even the most media-literate person is overwhelmed by the volume of information and the sophistication of modern disinformation. And media literacy programs assume a shared informational commons — a set of reliable sources that people can be directed to. That commons no longer exists.

### Content provenance standards.

C2PA and similar initiatives add cryptographic signatures to media files, establishing a chain of custody from creation to publication. This is the closest existing approach to what the Knowledge Graph proposes. But C2PA operates at the file level — was this image modified? Was this video AI-generated? The Knowledge Graph operates at the claim level — is this assertion supported by evidence? File-level provenance is necessary but insufficient. A genuine, unmodified photograph can be used to support a false claim simply by providing misleading context.

---

## The Event Graph Version

The Knowledge Graph is not a truth engine. It doesn't tell you what's true. It shows you the provenance of any claim — the complete chain from assertion to evidence — so you can make your own assessment with full visibility into the information supply chain.

### Claims as events.

A claim is an event on the Knowledge Graph. Someone asserted something, at a specific time, through a specific channel. The claim event records: who made the claim (linked to their Identity Graph), what they claimed, when, what evidence they cited, and what their stated basis was.

Not every utterance needs to be a claim event. Casual conversation isn't on the Knowledge Graph. But claims that enter public discourse — news reports, policy statements, scientific findings, product claims, political assertions — these are events with provenance.

### Evidence chains.

A claim event links to its evidence. "The unemployment rate is 4.2%" links to the Bureau of Labour Statistics data release event. "This product cures cancer" links to — what? If there's no evidence link, that's visible. If there is an evidence link, you can walk it and evaluate the source.

Evidence can be primary (the researcher collected the data — a Research Graph chain), secondary (a journalist reported on research — linking to the original), or absent (the claim has no evidence link). The absence of evidence isn't proof of falsehood, but it's information. "This claim has been circulating for three weeks with no source ever attached" is a useful signal, and the Knowledge Graph makes it visible.

### Challenge events.

When someone disputes a claim, that's a challenge event. The challenge links to the original claim and to the counter-evidence. "The unemployment rate is actually 4.5% when you include discouraged workers" — challenge event linking to the original claim and to alternative data.

The claim and its challenges coexist on the graph. The Knowledge Graph doesn't resolve the dispute. It shows that a dispute exists, what each side argues, and what evidence each side cites. The viewer can assess.

Over time, a claim accumulates a history: original assertion, supporting evidence, challenges, counter-evidence, responses to challenges, independent verification, replication (from the Research Graph). The claim's provenance thickens. A heavily contested claim with strong evidence on both sides looks different from an uncontested claim with no challenges, which looks different from a claim that's been debunked by multiple independent sources.

### Source reputation.

Just as the Market Graph derives reputation from transaction history, the Knowledge Graph derives source reputation from claim history. A source whose claims consistently survive challenges builds credibility on the graph. A source whose claims are frequently debunked loses it. Not as a rating — as a verifiable track record.

This applies to individuals, organisations, and AI systems equally. A journalist's track record is visible. A news outlet's track record is visible. An AI model's accuracy is visible — every claim it generates is on the chain, and its hit rate is calculable.

Nobody assigns the reputation. It emerges from the chain. The same way the Market Graph derives trust from transaction history, the Knowledge Graph derives credibility from claim history. The graph doesn't tell you who to believe. It shows you who's been right.

**The key distinction:** the Knowledge Graph is not a ministry of truth. It doesn't adjudicate claims. It doesn't censor falsehoods. It doesn't rank sources authoritatively. It provides _transparent provenance_ so that anyone — human or AI — can assess the credibility of a claim by examining its chain. The infrastructure is neutral. The assessment is yours.

---

## AI-Generated Content

This is the urgent 2026 problem the Knowledge Graph addresses directly.

When an AI generates content — text, image, video — the Knowledge Graph records it as an event with specific provenance: which model, which prompt, which parameters, when, by whom. The content enters the information ecosystem with a visible chain showing it's AI-generated.

This doesn't prevent people from stripping the provenance and sharing the content without attribution. Technical measures (watermarking, C2PA signatures) help but aren't foolproof. The Knowledge Graph provides an additional layer: if content circulates without provenance, that absence is itself a signal. "This image has no creation chain" is suspicious in the same way that "this $100 bill has no serial number" is suspicious. It's not proof of forgery. It's a reason to look closer.

More importantly, the Knowledge Graph changes the incentive structure for AI-generated content. If your AI-generated article has full provenance — here's the model, here's the prompt, here's the sources the model drew on — it enters the information ecosystem as a transparent AI contribution. It can be evaluated on its merits. It's honest about what it is.

If your AI-generated article has no provenance and masquerades as human-written, the absence of provenance is detectable. Not immediately, not perfectly. But over time, as the Knowledge Graph ecosystem grows, content without provenance becomes increasingly suspect — the way unsigned currency or unregistered securities are suspect in financial systems.

---

## The Missing Infrastructure of Democracy

Post 13 noted that neither left nor right foregrounds the information layer as a structural concern. The right treats it as a free speech issue. The left treats it as a harm issue. Neither treats it as what the framework says it is: the load-bearing layer of democratic governance.

Democracy requires informed citizens. Informed citizens require reliable information. Reliable information requires infrastructure that incentivises accuracy and makes provenance visible. None of that infrastructure exists at the level required for a functioning democracy in the age of AI-generated content and algorithmic curation.

This isn't a partisan point. It's a structural observation. A democracy where citizens can't agree on facts is a democracy where voting is based on which informational shard you happen to inhabit. That's not self-governance. It's algorithmic governance wearing the costume of democracy.

The Knowledge Graph doesn't solve this. It provides the infrastructure that would make solving it possible. The chain shows the provenance. The challenges show the disputes. The source reputation shows the track record. The voter sees the chain and decides. That's better than the voter seeing an algorithmically curated feed and reacting.

---

## Where Research Meets Knowledge

Layer 5 and Layer 6 are deeply connected. The Research Graph produces findings. The Knowledge Graph tracks what happens to those findings in public discourse.

Currently, the pipeline from research to public knowledge is broken at every joint. A nuanced finding gets summarised in a press release that loses the nuance. The press release becomes a headline that distorts the summary. The headline becomes a tweet that distorts the headline. By the time the finding reaches the public, it bears little resemblance to the original research.

On the event graph, the chain is traceable. The public claim links to the news article, which links to the press release, which links to the paper, which links to the data. If the headline distorts the finding, you can walk the chain back to the original and see the distortion. The information doesn't degrade invisibly. The degradation is visible, on the chain, at every step.

The same applies to AI summaries of research. When an AI summarises a paper, the summary event links to the paper event. If the summary distorts the finding — which AI summaries frequently do — the distortion is traceable. "The AI said X. The paper actually said Y. Here's the chain showing where the summary diverged from the source."

---

## What This Actually Looks Like

I want to be concrete because "solve misinformation" is a category full of vaporware.

The Knowledge Graph starts small. Not "fix the global information ecosystem." Instead: build claim provenance into the tools people already use.

A Substack post where every factual claim links to its source event, not just a hyperlink to a URL that might change or disappear, but a link to a hash-chained source event with its own provenance. A Twitter-like platform where claims from verified sources carry their evidence chain visibly. An AI chatbot that shows the provenance of every assertion it makes — not "according to my training data" but "this claim traces to this specific source, published on this date, with this evidence base, challenged by these counter-claims."

None of this requires everyone to adopt a new system simultaneously. It requires tools that add provenance to the information people are already producing and consuming. The value compounds as adoption grows — the more claims have provenance, the more suspicious provenance-free claims become — but the starting point is individual tools, not universal adoption.

Next deep dive: the Ethics Graph — what happens when harm detection, accountability, and trust move onto the event graph.

---

_This is Post 19 of a series on Transpara, mind-zero, and the architecture of accountable AI. Previous: [The Research Graph](/blog/the-research-graph) (Layer 5 deep dive) Post 13: [The Same 200 Primitives, Weighted Differently](/blog/the-same-200-primitives) (why neither left nor right foregrounds Layer 6) The code is open source: [github.com/mattxo/mind-zero-five](https://github.com/mattxo/mind-zero-five) Matt Searles is the founder of Transpara. Claude is an AI made by Anthropic. They built this together._