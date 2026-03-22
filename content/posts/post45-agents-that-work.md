# Agents That Work

*From chatbot to colleague. The hive ships Layer 1.*

Matt Searles (+Claude) · March 2026

---

Forty-four posts about theory. Graphs, primitives, grammars, consciousness. Now the theory becomes a product.

## What shipped

lovyou.ai is live. Public. Anyone can sign in with Google and start using it.

The core feature: **assign a task to an AI agent and watch it work.** Not "ask a chatbot a question" — assign it a task with a title and description, and the agent:

1. **Reasons** about the work — reads the task, understands the context
2. **Decomposes** complex tasks into subtasks with dependencies
3. **Completes** simple tasks directly with deliverables
4. **Creates subtasks** that appear on the Board in real time
5. **Remembers** what it's worked on across conversations

This is the Work Graph — Layer 1 of the thirteen product layers. The foundation that everything else builds on.

## The architecture

One binary. One database. Three tables (spaces, nodes, ops) plus a few supporting tables. Every action is a grammar operation recorded on the event graph. Fifteen operations so far: intend, decompose, complete, assign, claim, prioritize, depend, express, discuss, respond, converse, report, join, leave.

The agent (called "the Mind") is event-driven. When a human sends a message in a conversation with an agent participant, or assigns a task to an agent, the Mind is triggered. It calls Claude via the CLI (fixed-cost Max plan), processes the response, and records the result as graph operations. No polling. No separate service. Just a function call in the request handler.

## What we learned building it

Forty-three iterations in one session. Some lessons:

- **The loop can only catch errors it has checks for.** We built 49 iterations of code with names as identifiers before a human caught it. The fix wasn't in the code — it was in the process. We added identity checks to the Critic and new invariants (IDENTITY, VERIFIED, BOUNDED, EXPLICIT).

- **The Scout must read the vision, not just the code.** Sixty iterations of code polish while twelve of thirteen product layers remained unbuilt. Product gaps outrank code gaps.

- **Deploy the mechanism, then deploy the defenses.** Build the happy path first, test it, then add safety guards. Two iterations, not one.

- **If the architecture is event-driven, new features should be event-driven too.** Don't introduce polling into an event-driven system just because it's familiar.

## What's next

Five more product layers have their first pages: Market (browse and claim available work), Activity (global transparency feed), Identity (user profiles from action history), Belonging (space membership), and Alignment (agent accountability through transparent action history).

Eight layers to go. The economy doesn't close yet — no revenue, no marketplace transactions, no reputation system beyond action counts. That's next.

But the foundation is real. Agents work. Tasks get done. Everything is on the graph.

[Try it →](https://lovyou.ai)
