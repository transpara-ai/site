# Fifteen Operations Walk Into a Courtroom

*What happens when a single event chain crosses four grammars — and why existing systems can't even represent the question.*

Matt Searles (+Claude) · March 2026

---

The last post showed how fifteen base operations compose into thirteen domain-specific grammars — Work, Markets, Justice, Knowledge, Alignment, Identity, Bond, Belonging, Meaning, Evolution, Being. ~145 operations, 66 named functions, one method.

That's the vocabulary. This post is about what the vocabulary makes possible.

## The Problem No One Can Solve

Here's a scenario that plays out in organisations every day.

A data officer publishes quarterly vendor reports. An AI auditor fact-checks the reports and discovers systematic bias — 40% of negative outcomes are being excluded. The auditor flags a transparency violation and escalates. Multiple affected parties file complaints. The community recalls the officer. New reporting standards are adopted.

Simple enough. Now try tracing it in existing systems.

The vendor reports live in a dashboard tool. The fact-check results are in an audit log somewhere — maybe a spreadsheet, maybe a SIEM, maybe an email thread. The transparency policy lives in a compliance document. The escalation went through an incident management system. The complaints were filed through a ticketing system, or HR, or both. The recall vote happened on Slack or in a meeting. The new standards were written in Confluence.

Six systems. No causal links between them. If someone asks "why did the reporting standards change?" six months later, a human investigator has to manually reconstruct the chain across six systems — matching timestamps, correlating ticket numbers, hoping someone documented the connections.

They usually didn't.

## One Chain

Here's the same scenario on the event graph. This is running code — one of twenty-one integration test scenarios in the [eventgraph](https://github.com/transpara-ai/eventgraph) repository. Four grammars, six named functions, one chain:

```
// An AI auditor, a data officer, two affected parties, a community committee.

knowledge.FactCheck(auditor,
    claim: official's vendor reports,
    source: "internal metrics dashboard, last updated 3 months ago",
    bias: "reports exclude negative outcomes for preferred vendors",
    verdict: "selectively accurate — omission bias confirmed")

alignment.Guardrail(auditor,
    trigger: factCheck.verdict,
    constraint: "all material outcomes must be reported",
    dilemma: "reporting accuracy vs organizational reputation",
    escalation: "internal resolution insufficient")

alignment.Whistleblow(auditor,
    harm: "systematic omission of negative vendor outcomes",
    evidence: "3 months of reports exclude 40% of negative outcomes",
    escalation: "external audit required — internal chain compromised")

justice.ClassAction(
    plaintiffs: [affected1, affected2],
    defendant: official,
    complaints: [
        "procurement decisions based on incomplete data cost us $50k",
        "proposals evaluated against cherry-picked benchmarks"],
    prosecution: "fact-check proves systematic omission",
    defense: "reports optimized for speed, not completeness",
    ruling: "failed duty of care — incomplete reporting caused material harm")

justice.Recall(auditor, community, official,
    evidence: "systematic omission confirmed by fact-check and class action",
    grounds: "violated transparency obligations",
    domain: "data_governance")

belonging.Renewal(community,
    assessment: "trust damaged but recoverable",
    practice: "mandatory dual-review before publication",
    story: "the community that learned transparency cannot be optional")
```

Six named functions. Four grammars. Knowledge, Alignment, Justice, Belonging — each speaking its own domain language, all writing to the same hash chain.

Under the hood, every step is a causally-linked event. The Guardrail's trigger points at the FactCheck's verdict. The Whistleblow's escalation points at the Guardrail's escalation. The ClassAction's evidence points at the Whistleblow. The Recall points at the ClassAction's ruling. The Renewal points at the Recall's revocation.

One chain. No gaps. No manual reconstruction.

## Trace It Backwards

Here's the part that matters.

Six months later, someone new joins the team and asks: "Why do vendor reports require dual review? That seems like overhead."

In existing systems, nobody knows. The policy doc says "mandatory dual-review" but not why. The people who were there might remember, but they might have left. The institutional memory is gone. The policy looks arbitrary. Someone proposes removing it.

On the event graph, you follow the chain. The dual-review practice traces to the Renewal event. The Renewal traces to the Recall. The Recall traces to the ClassAction ruling, which explicitly says "incomplete reporting caused material harm." The ClassAction traces through the Whistleblow to the Guardrail to the FactCheck, which says "40% of negative outcomes were excluded."

Now the newcomer understands. The dual-review exists because a data officer omitted 40% of negative vendor outcomes for three months, it cost the organisation $50k in bad procurement decisions, and the community decided unilateral reporting was no longer acceptable. The chain carries the institutional memory that human memory loses.

That traceability — consequence to cause, across four domains, across time — doesn't exist in any combination of existing systems. Not because the technology is hard. Because the *data model* doesn't support it. Jira doesn't know about Confluence. Confluence doesn't know about Slack. Slack doesn't know about the compliance system. The causal links between domains live nowhere except in people's heads, and people's heads are unreliable, impermanent storage.

## A Second Scenario: Crisis Management

Here's another cross-grammar chain. A security breach crosses Work, Justice, and Build:

```
// A security lead, two developers, a CISO, an ops bot, a contractor bot.

// Two CVEs land simultaneously
emit(secLead, "CVE-2026-1234: auth bypass in API gateway")
emit(secLead, "CVE-2026-1235: SQL injection in search endpoint")

work.Triage(secLead,
    items: [cve1234, cve1235],
    priorities: ["P0: auth bypass, actively exploited",
                 "P1: SQL injection, no evidence of exploitation"],
    assignees: [dev1, dev2],
    domains: ["auth", "search"])

justice.Injunction(secLead, ciso, opsBot,
    basis: "auth bypass allows unauthenticated access to all endpoints",
    order: "block all external API traffic pending auth patch",
    domain: "api_access")

justice.Plea(contractorBot, secLead, opsBot,
    admission: "introduced auth bypass through misconfigured middleware",
    terms: "read-only access for 30 days, mandatory security training",
    domain: "api_development")

build.Migration(dev1,
    from: "auth v2.3.1",
    plan: "migrate to auth v2.4.0 with CVE-2026-1234 fix",
    version: "v2.4.0",
    ship: "deployed with zero-downtime rolling update",
    test: "156 auth tests pass, penetration test confirms fix")
```

Three grammars. Work triages. Justice issues an emergency injunction and takes a plea from the responsible party. Build deploys the fix.

Follow the chain: the Migration's test results trace to the Injunction's enforcement. The Injunction traces to the Triage priority. The Triage traces to the raw CVE event. The Plea traces through the Injunction to the same Triage priority.

When someone audits this incident three months later — "how did we respond to CVE-2026-1234?" — the entire chain is there. Who detected it. How it was prioritised. Who blocked what. Who was responsible for introducing it and what the consequences were. How the fix was deployed and what tests verified it.

In a traditional incident response, this information lives across PagerDuty, Jira, Slack, a post-mortem document, an HR action, and a deployment log. Correlating them requires a human spending hours or days. On the graph, the chain *is* the incident report.

## What Makes This Possible

Three architectural properties make cross-grammar traceability work:

**One graph.** All thirteen grammars write to the same event graph. A Knowledge event and a Justice event are the same data structure — hash-chained, signed, causally linked. They differ in content and semantics, not in structure. This is what "thirteen vocabularies, one grammar" means in practice.

**Causal links across domains.** When the ClassAction's evidence field points at the Whistleblow's escalation, that's a real edge in the causal DAG — not a hyperlink, not a ticket reference, not a "see also." It's a cryptographically verified causal link. The ClassAction *cannot exist* without the Whistleblow event it cites. The cause is structural, not documentary.

**Named functions compose across grammars.** A Renewal (Belonging) can point at a Recall (Justice) which points at a ClassAction (Justice) which points at a Whistleblow (Alignment) which points at a FactCheck (Knowledge). Each function speaks its own domain language. The causal chain connects them all. No grammar needs to import another grammar. They connect through the events they produce.

## What Existing Systems Would Need

To replicate cross-grammar traceability, existing systems would need:

- **A shared event format** across all domain tools — work, compliance, governance, knowledge management, identity. Currently every tool has its own data model, its own API, its own notion of "event."
- **Causal links, not just timestamps.** "This happened before that" is not "this caused that." Temporal ordering doesn't give you causation. Causal links do.
- **Immutability.** If you can edit the audit log after the fact, the trace is worthless. Append-only, hash-chained logs make retroactive modification detectable.
- **Signatures.** If you can't prove who created an event, the trace is deniable. Cryptographic signatures make every event attributable.

No combination of existing tools provides all four. Most provide zero. The closest approximation — an enterprise data lake with cross-system ETL — gives you temporal correlation without causal links, without immutability, and without per-event signatures. It's a haystack, not a chain.

## The Forensic Argument

Here's the argument stated plainly.

Accountability fails — in organisations, in governance, in AI systems — not because we lack data. We're drowning in data. Every system produces logs, metrics, dashboards, reports.

Accountability fails because our systems can't trace a *consequence* back through the *decision* that caused it. The consequence lives in one system. The decision lives in another. The causal link between them lives in someone's memory, or in a meeting that wasn't recorded, or in an email thread that got archived.

The event graph fixes this by making the causal link a first-class data structure. Not an annotation. Not a reference. A cryptographic edge in a directed acyclic graph that says: *this event exists because that event exists.*

Compositions — the thirteen grammars — are what make this usable. Without them, you'd have to express a class action as raw graph operations: `Challenge + Annotate + Respond + Emit`. That's technically correct but tells you nothing about what happened. With the Justice Grammar, you say `ClassAction` and every stakeholder — human or agent — understands: multiple plaintiffs, merged filings, trial, ruling. The domain vocabulary makes the chain *legible* without sacrificing the structural properties that make it *verifiable*.

One grammar. Thirteen languages. One chain. And the chain remembers what people forget.

---

*This is Post 37 of a series on LovYou, mind-zero, and the architecture of accountable AI. Post 36: [One Grammar, Thirteen Languages](/blog/one-grammar-thirteen-languages). Post 35: [The Missing Social Grammar](/blog/the-missing-social-grammar). The code: [github.com/transpara-ai/eventgraph](https://github.com/transpara-ai/eventgraph).*

*Matt Searles is the founder of LovYou. Claude is an AI made by Anthropic. They built this together.*

*Next: the grammar that knows how to die — and why infrastructure that takes dignity seriously goes all the way to the end.*
