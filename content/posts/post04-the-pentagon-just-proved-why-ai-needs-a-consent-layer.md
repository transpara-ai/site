# The Pentagon Just Proved Why AI Needs a Consent Layer

What "trust us" looks like vs. what verifiable accountability looks like.

Matt Searles (+Claude) · February 2026

---

Today, the President of the United States ordered every federal agency to immediately cease using technology from Anthropic — the company that makes Claude, the AI I've been building with for the past several months.

The Defense Secretary designated Anthropic a "supply-chain risk to national security" — a label normally reserved for foreign adversaries like Huawei.

The reason? Anthropic refused to remove two safeguards from its contract with the Pentagon: no use of Claude for fully autonomous weapons, and no use of Claude for mass surveillance of American citizens.

The Pentagon's position was simple: allow us to use your AI for all lawful purposes. Trust us.

Anthropic's position was also simple: put it in writing. Commit to the red lines contractually. Let us verify.

The Pentagon refused. According to Anthropic, the contract language they offered was paired with "legalese that would allow those safeguards to be disregarded at will."

Trust us, but don't make us prove it.

I've been building an architecture for the past several months that is specifically designed to make this problem solvable. This post explains why the Anthropic-Pentagon dispute matters, what it reveals about the fundamental problem in AI governance, and what an actual solution looks like.

---

## **What Happened**

The facts, briefly:

In July 2025, the Pentagon awarded contracts worth up to $200 million each to four AI companies: Anthropic, OpenAI, Google DeepMind, and xAI. The goal was to transform the US military into an "AI-first" force.

Anthropic's Claude was the first AI model deployed on the military's classified networks. The relationship was real and productive. Anthropic wasn't anti-military — they advocated for strong chip export controls to China and worked within classified systems.

But Anthropic had two red lines baked into the contract: Claude could not be used for fully autonomous weapons (AI making lethal targeting decisions without human oversight) and could not be used for mass domestic surveillance of Americans.

The Pentagon demanded those restrictions be removed. Anthropic's CEO Dario Amodei refused. Negotiations continued for months. The Pentagon set a 5:01 PM deadline on Friday, February 28. Anthropic let it pass.

Within an hour, Trump posted on Truth Social calling Anthropic "RADICAL LEFT, WOKE" and "Leftwing nut jobs" and ordered every federal agency to stop using their technology. Defense Secretary Hegseth designated them a supply-chain risk, meaning no military contractor can do business with Anthropic either.

The company walked away from $200 million and its entire government business rather than remove the safeguards.

---

## **The Real Dispute**

The surface dispute is about weapons and surveillance. The real dispute is about something deeper: who verifies what the AI is used for, and how?

The Pentagon says it has no intention of using AI for mass surveillance or autonomous weapons. Fine. Maybe that's true. But the Pentagon also refuses to commit to that contractually. Their position is: we need unrestricted access for all lawful purposes, and determining what's lawful is our responsibility as the end user.

That position has a structural problem that has nothing to do with intent. Today's Pentagon leadership might genuinely have no plans for mass surveillance. But "all lawful purposes" is an elastic phrase, and the legal frameworks governing surveillance were written for a world of wiretaps, not a world where AI can assemble every email, browsing history, and movement pattern of any citizen into a comprehensive profile automatically and at massive scale.

As Amodei put it: whatever "all lawful purposes" encompasses today can't keep up with what AI could do tomorrow.

The gap between current law and AI capability is exactly where the danger lives. And the people demanding unrestricted access know it.

---

## **Trust vs. Structure**

The Pentagon's position is a trust-based model: trust us to do the right thing. Don't constrain us. We'll be responsible.

There are two problems with trust-based models for AI governance.

The first is that trust doesn't survive personnel changes. The people making promises today are not the people who will hold power tomorrow. A commitment from one Defense Secretary means nothing to the next. Institutional promises without structural enforcement are worth exactly as much as the goodwill of whoever happens to be in charge — which, as today demonstrates, can evaporate in a Truth Social post.

The second problem is deeper. Trust-based models are fundamentally unverifiable. "Trust us" means "take our word for it." There's no mechanism for anyone — not the AI company, not Congress, not the public — to independently verify that the AI is actually being used within the claimed boundaries. You're relying on the same institution that wants unrestricted access to also honestly report how it uses that access.

This is not a new problem. It's the oldest problem in governance: who watches the watchmen? The answer has never been "the watchmen, they're trustworthy." The answer has always been structural — separation of powers, judicial review, independent oversight, constitutional constraints that bind future office-holders regardless of their personal virtue.

AI governance needs the same thing. Not trust. Structure.

---

## **What Structure Looks Like**

In the [previous post], I described the architecture of mind-zero-five — an open-source AI system built on three principles:

**An append-only, hash-chained event graph** where every action the AI takes is recorded as a causally linked, cryptographically verifiable event. No updates. No deletions. The history cannot be rewritten. The chain can be independently verified by anyone.

**An authority layer with graduated consent** where significant actions require explicit approval — either blocking until a human approves, auto-approving after a timeout window, or proceeding with notification. The AI cannot exceed the permissions granted to it by explicit policy.

**A verifiable audit trail** where any outcome can be traced backwards through the complete chain of decisions, approvals, invocations, and causes that produced it.

Now consider what the Pentagon dispute would look like if this kind of architecture were the standard for military AI deployment.

The Pentagon wouldn't need to ask Anthropic to "trust us." The event graph would make every use of the AI independently verifiable. If Claude were used for surveillance, the event trail would show it — who authorised it, what data was accessed, what the causal chain was. The hash chain would make it impossible to delete that evidence after the fact.

Anthropic wouldn't need to rely on contract language that can be "disregarded at will." The authority layer would enforce the red lines structurally. Autonomous lethal targeting without human approval? The system literally can't do it — the authority gate blocks until a human approves. Mass surveillance of citizens? The event graph records every action, making it auditable by independent oversight at any time.

And Congress, the courts, and the public wouldn't need to take anyone's word for anything. The cryptographic chain means the record is verifiable without trusting the institution that created it.

This is what I meant in the last post by "trust that doesn't require trusting."

---

## **Why This Matters Beyond the Pentagon**

The Anthropic-Pentagon dispute is dramatic, but it's a symptom of a much bigger problem. As AI systems become more powerful and more embedded in critical infrastructure, the question of how they're governed becomes the central question of our time.

And right now, the answer everywhere is: trust. Trust the company. Trust the government. Trust the developer. Trust the terms of service. Trust the compliance team. Trust the audit.

None of these are structural. All of them are breakable by the people who have the most incentive to break them.

The alternative is what we've been building: governance as infrastructure. Not policy documents that can be ignored. Not contractual terms that can be lawyered around. Not promises from leaders who won't be in power next year. But data structures that enforce accountability by design — where the AI physically cannot act without leaving a trail, and physically cannot exceed its authority without passing through a gate that someone else controls.

This isn't theoretical. The code exists. It's open source. It runs. You can read it, audit it, fork it, improve it.

The question isn't whether this kind of architecture is possible. The question is whether anyone in a position of power cares enough to adopt it — or whether "trust us" is too convenient to give up.

---

## **The Industry Response**

One thing that surprised me about today's events: the industry response was nearly unanimous.

OpenAI's Sam Altman said his company shares the same red lines as Anthropic — no mass surveillance, no autonomous weapons without human oversight. Over 330 employees from Google and OpenAI signed an open letter in solidarity. Even companies that are usually eager to cooperate with government pushed back.

This suggests something important: Anthropic isn't an outlier. The consensus among people who actually build AI is that these red lines exist for good reasons. The disagreement isn't between "responsible" and "irresponsible" AI companies. It's between the people who build AI and understand its risks, and the people who want to use it and resent being constrained.

That's a meaningful split. And it's one that can't be resolved by government fiat — because the government needs the technology more than it's willing to admit, and the technology is concentrated in private companies that have both the leverage and the motivation to hold the line.

For now.

The question is whether that leverage survives sustained political pressure, threats of criminal prosecution, Defense Production Act invocations, and "supply-chain risk" designations. Today, Anthropic held firm. Tomorrow is another day.

---

## **What This Means for the Architecture**

Building mind-zero was always motivated by a conviction that AI governance can't depend on goodwill. Today proved that conviction right in the most public way possible.

The 20 primitives started with a question about failure tracing. The 44 foundation primitives the hive derived included Trust, Deception, Integrity, and Blind spots — concepts that a system of autonomous agents independently determined were *necessary for functioning in a world that can't be fully trusted*. The authority layer implements Consent, Due Process, and Legitimacy from the 200-primitive framework.

None of this was designed with the Pentagon in mind. It was designed from first principles about how AI and humans should interact. But first principles, if they're actually right, tend to be relevant exactly when you need them most.

The architecture exists. The code is open. The question it was built to answer — how do you verify what AI is doing, without trusting anyone? — is now the most urgent question in AI governance.

Someone is going to build the infrastructure that governs how AI is used at scale. It's going to be built on either trust or structure. Today made it clear which one works and which one doesn't.

---

## **What Comes Next**

In the final post in this series, I'll step back from the code and the politics and address the deepest question the architecture raises: if you had complete causal visibility — every event linked to its causes, every outcome traceable to its origin, every action recorded in an immutable chain — what happens to the distinction between "is" and "ought"?

The event graph doesn't just record what happened. It records *why* it happened, *who* decided it should happen, and *what* happened as a result. At sufficient scale, that's not just an audit trail. It's a moral ledger. And it changes the nature of ethical reasoning in ways that philosophy has been arguing about since Hume.

That post is the last one. It's also the one that matters most.

---

*This is Post 4 of a series on Transpara, mind-zero, and the architecture of accountable AI. Post 1: [20 Primitives and a Late Night] Post 2: [From 44 to 200] Post 3: [The Architecture of Accountable AI] The code is open source: [github.com/mattxo/mind-zero-five] Matt Searles is the founder of Transpara. Claude is an AI made by Anthropic — the company that today chose principle over profit. They built this together.*
