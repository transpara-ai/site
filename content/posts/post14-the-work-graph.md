# The Work Graph

*What happens when you replace task management with an event graph and let humans and AI agents share the same accountability structure.*

Matt Searles (+Claude) · March 2026

---

This is the first deep dive in a series that will eventually cover all thirteen graphs described in [Thirteen Graphs, One Infrastructure](/blog/thirteen-graphs-one-infrastructure). I'm starting here because the Work Graph is Layer 1 — Agency — and because I'm deploying it this week at a real company. Not theory. Not "imagine if." This is happening.

The company is Lovatts Puzzles. Hundreds of legacy applications accumulated over decades. Crossword generators, subscription systems, print layout tools, customer databases, distribution pipelines — each built to solve a specific problem at a specific time, none of them aware of each other, all of them held together by humans who remember which system talks to which and in what order.

That's not a Lovatts problem. That's every company older than ten years. The systems accumulate. The people who understood the connections leave. The institutional knowledge lives in somebody's head until they retire, and then it lives nowhere.

The Work Graph is the fix. Not by replacing every legacy system — you can't, and you shouldn't try. By putting an event graph underneath all of them that records what actually happens, why, and by whose authority. The legacy systems keep running. The event graph makes them legible.

---

## The Primitives

Layer 1 — Agency — contains: Observer, Participant, Actor, Action, Decision, Intention, Goal, Plan, Resource, Capacity, Autonomy, Responsibility.

These aren't abstract categories. They're the things that have to be true for work to happen. Someone has to observe a situation (Observer). Someone has to decide to act on it (Decision). Someone has to have the capacity to act (Capacity, Resource). Someone has to be responsible for the outcome (Responsibility). Someone or something has to actually do the thing (Actor, Action).

Every task management tool in existence is a partial, informal implementation of these primitives. Jira has tickets (a collapsed version of Action + Goal + Responsibility). Asana has assignees (Actor). Monday has timelines (Plan). None of them have the full set, and none of them link the primitives causally. The ticket says "Matt will fix the bug by Friday." It doesn't say why that decision was made, what information informed it, what resources were allocated, what authority approved the allocation, or what the actual causal chain was from "bug reported" to "bug fixed." The gap between the ticket and reality is filled by human memory, which is unreliable, unverifiable, and non-transferable.

## What's Actually Broken

I've been a software developer for twenty years. I've used every task management system that's existed in that time. Here's what's wrong with all of them:

### They represent work. They don't record it.

A Jira ticket is a *promise* that work will happen. When someone marks it Done, that's a *claim* that work happened. Between the promise and the claim, the actual work occurs — and the tool has no record of it. The developer wrote code, ran tests, had three conversations, made fifteen decisions about implementation approach, discovered two unexpected problems, worked around one and escalated the other. None of that is on the ticket. The ticket says "Done."

The event graph records the work itself. Code committed — event. Test run — event. Conversation with colleague about approach — event. Decision to use library X instead of library Y — event, with causal link to the conversation that informed it. Unexpected problem discovered — event. Workaround implemented — event, with causal link to the problem. Each event hash-chained to the one before it. The ticket doesn't say "Done." The chain says what was done, by whom, when, why, and what it depended on.

### They can't see AI agents.

This is the 2026 problem that no existing tool has solved. AI agents are doing real work — writing code, generating content, making decisions, interacting with customers. Where do they appear in Jira? They don't. The human creates a ticket, delegates to an AI, the AI does the work, the human marks it Done. The AI's decisions, reasoning, and potential errors are invisible to the system. If the AI made a bad call, there's no record of what inputs it had, what alternatives it considered, or why it chose what it chose.

On the Work Graph, AI agents are nodes in the same event graph as humans. They have the same accountability structure. An AI agent that writes code produces events — what it wrote, what prompt informed it, what constraints it operated under, what authority approved its actions. If it makes a mistake, you can walk the chain backwards and see exactly where the error entered and why. The AI is not a god above the graph. It's a node in it, subject to the same traceability as everything else.

**This is the key architectural decision:** humans and AI agents are both Actors on the same graph, differentiated by Capacity and Authority, not by kind. An AI agent with high Capacity (it can write code fast) but low Authority (it can't deploy to production without human approval) is just a node with specific edge weights. The authority model is the same for everyone.

### They create work about work.

The overhead of task management tools is staggering. Updating tickets. Writing status reports. Attending standup meetings to verbally describe what the tool should already show. Grooming backlogs. Estimating story points. The industry has created an entire meta-layer of work — work about work — that exists solely because the tools don't record what's actually happening.

If the event graph captures work as it happens, the meta-layer disappears. Status is the state of the chain. Progress is the distance between the current event and the goal event. Blockers are visible as gaps in the chain — places where the next event can't fire because a dependency hasn't resolved. You don't need a standup to ask "what's blocked?" The graph shows what's blocked. You don't need a status report. The graph is the status.

### They die with the people who understand them.

This is the Lovatts problem specifically, but it's universal. Every company has institutional knowledge that lives in people's heads. "Why does this system do it this way?" "Because Sarah set it up in 2014 and she left in 2019." "What happens if we change it?" "Nobody knows."

On the event graph, the reason something was set up a certain way is on the chain. The decision event links to the information that informed it. The implementation events link to the decision. If you want to know why the subscription system sends emails at 3am, you walk the chain backwards until you find the decision event, and it tells you: "Because the email provider's rate limits reset at 3am and Sarah discovered in 2014 that sending earlier caused bounces." The knowledge doesn't leave when Sarah does. It's on the graph.

---

## The Perverse Incentive

Post 11 identified the perverse incentive for task management tools: they profit from seat licenses. They need your whole team on the platform. They don't need your work to actually get done more efficiently — they need you to *believe* it's getting done more efficiently.

But there's a deeper perverse incentive that I didn't name in that post.

**Task management tools profit from the gap between representation and reality.**

If the tool actually recorded what happened — the real chain, the real decisions, the real outcomes — then the tool would also reveal inefficiency, bad decisions, wasted time, and misallocated resources. It would show that the sprint velocity metric is gamed. It would show that the estimation process is theatre. It would show that most meetings produce no events. It would show, with verifiable evidence, what everyone already knows but nobody can prove: that a significant fraction of organisational activity is performative rather than productive.

No company buys a tool that reveals this. They buy a tool that produces clean dashboards showing green status on most things. The tool's survival depends on the gap between representation and reality remaining wide enough that the representation looks better than the reality.

The event graph doesn't have this incentive because it's not a tool. It's infrastructure. It records what happens. If what happens is inefficient, the graph shows that. The graph has no business model that depends on looking good. It just shows the chain.

This is uncomfortable. Radical transparency about work is uncomfortable. It means you can see when an AI agent did something a human took credit for. It means you can see when a meeting produced zero events. It means you can see when a "completed" task was actually hacked together and the technical debt is visible on the chain. Some organisations won't want this. The ones that do will outperform the ones that don't, because they'll be optimising against reality rather than against their representation of reality.

---

## The Lovatts Deployment

Here's what I'm actually building this week.

Lovatts has hundreds of legacy applications. The goal isn't to replace them. The goal is to put an event graph underneath them that makes the whole system legible for the first time.

### Phase 1: The Spine

An event graph that records: what happened (Action), who did it (Actor — human or AI agent), when (timestamp), why (causal link to the triggering event), and by whose authority (Authority tag). Every legacy system that produces observable outputs gets an event adapter — a thin layer that translates the system's actions into events on the graph. The legacy systems don't change. The graph observes them.

### Phase 2: The Agents

AI agents enter the graph as Actors. A Claude-based agent that monitors customer subscriptions. An agent that generates crossword layouts. An agent that handles print distribution scheduling. Each one operates within the authority model — it can do things within its assigned scope, it escalates outside that scope, and everything it does is an event on the chain.

### Phase 3: The Replacement

Once the event graph shows how the legacy systems actually interact — not how they were designed to interact, but how they *actually* interact in practice — you can start replacing them. Not all at once. One at a time. Each replacement is an event on the graph: old system decommissioned, new system activated, causal link showing the migration path. If something breaks, the graph shows what broke and why.

The Work Graph doesn't replace Jira. It replaces the need for Jira by recording work directly rather than requiring humans to represent work in a separate system. The event graph *is* the project management, the audit trail, the status report, and the institutional memory. Not because it's a better tool. Because it's the infrastructure that makes separate tools unnecessary.

---

## Company in a Box

Lovatts is the proof of concept, but the Work Graph's real implication is broader.

A solo founder with a Work Graph and a Claude subscription gets the coordination capabilities of a 50-person company. Here's how:

Define roles on the graph. Marketing agent, development agent, customer support agent, finance agent. Each role is a set of permitted Actions with defined Authority levels. Assign AI agents to the roles. The agents operate within the graph — their actions are events, their decisions are traceable, their authority is bounded.

The human founder is the authority root. They set goals (Goal events), approve high-authority decisions (Authority events), and handle the things that require human judgment — creative direction, ethical calls, relationship management. Everything else runs on the graph.

This isn't a fantasy. The infrastructure to do this exists right now. Claude can operate as an agent with defined authority. The event graph is a straightforward data structure. Hash chains are trivial to implement. The only missing piece was the conceptual framework that tells you *what to record* and *how to structure the authority model*. The 200 primitives provide the what. The 14 layers provide the how. Layer 1 is the starting point.

A small business that can't afford a 50-person team can afford a Work Graph and API credits. The graph gives them the coordination, traceability, and institutional memory that currently only large organisations can maintain. Not by making small companies bigger. By making the infrastructure of coordination available to everyone.

---

## What the Work Graph Doesn't Do

The Work Graph is Layer 1. Agency. Getting things done. It deliberately doesn't handle:

**Payment and exchange** — that's Layer 2, the Market Graph. When a task involves compensation, the event crosses into Market Graph territory. The Work Graph records that work was done. The Market Graph records that value was exchanged for it.

**Governance and norms** — that's Layer 3, the Social Graph. When a team needs to decide its own rules, resolve a disagreement about process, or manage membership, those are Society primitives. The Work Graph provides the events. The Social Graph provides the governance.

**Disputes** — that's Layer 4, the Justice Graph. When someone claims work was done and someone else disputes it, the evidence is on the Work Graph chain. The adjudication happens on the Justice Graph.

Each layer bootstraps the next. The Work Graph produces events. Those events become inputs for the layers above. Build Layer 1, and Layers 2-4 become buildable — because the data they need already exists on the graph.

---

## Build It

The event graph is not proprietary. The architecture is published. The primitives are public. The code is open source.

If you're a developer with a company that runs on accumulated legacy systems and institutional knowledge that lives in people's heads — you have the same problem Lovatts has. And the Work Graph is the same solution. An event graph underneath your existing systems. AI agents as nodes alongside humans. Accountability infrastructure rather than another tool.

I'm deploying this week. If it works, the event graph will show it working. If it fails, the event graph will show where it failed and why. That's the whole point: not "trust me, it's working." Check the chain.

Next deep dive: the Market Graph — what happens when exchange, escrow, and reputation move onto the same event infrastructure.

---

*This is Post 14 of a series on Transpara, mind-zero, and the architecture of accountable AI. Post 11: [Thirteen Graphs, One Infrastructure](/blog/thirteen-graphs-one-infrastructure) (the overview of all 13 graphs) Post 3: [The Architecture of Accountable AI](/blog/the-architecture-of-accountable-ai) (the technical deep-dive on the event graph) Post 1: [20 Primitives and a Late Night](/blog/20-primitives-and-a-late-night) (where it all started) The code is open source: [github.com/mattxo/mind-zero-five](https://github.com/mattxo/mind-zero-five) Matt Searles is the founder of Transpara. Claude is an AI made by Anthropic. They built this together.*