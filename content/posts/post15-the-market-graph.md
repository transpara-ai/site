# The Market Graph

_What happens when you stop paying platforms 25% to mediate trust they don't actually provide._

Matt Searles (+Claude) · March 2026

---

## The Primitives

Layer 2 contains: Offer, Acceptance, Obligation, Reciprocity, Property, Contract, Debt, Gift, Competition, Cooperation, Scarcity, Surplus.

These are the building blocks of every transaction between two entities. The primitives haven't changed over ten thousand years. What's changed is who sits between parties and their cut.

## The Toll Booth Economy

Every major marketplace operates identically: two parties want to transact, a platform sits between them, extracting a percentage for mediating trust.

Uber takes 25-30% of fares. Airbnb takes 14-20% combined. Upwork takes 10-20% of freelancer earnings. The pattern repeats across Fiverr, DoorDash, TaskRabbit, and Etsy.

### Three things. Two are commodities.

**Discovery** — Connecting buyers with sellers is a solved problem. Any LLM can match supply to demand. Search and matching are commodity infrastructure, not worth 25%.

**Payment processing** — Stripe charges 2.9% plus fees. Payment processing is commodity infrastructure, definitely not worth 25%.

**Trust** — The belief the other party honors commitments. This is the real product. Discovery and payments are the excuse. Trust is the moat.

## Why Platform Trust Is Broken

### Reviews are gamed.

Amazon hosts a fake review industry worth hundreds of millions. Airbnb reviews trend toward 4.8 stars as floor, making ratings nearly useless. The system selects for dishonest positive reviews.

### Reputation is captive.

An Uber driver's 4.95 rating exists only on Uber. If banned, years of good service disappear. The captive reputation is the lock-in mechanism. The platform doesn't just mediate trust. It holds trust hostage.

Reputation is owned by platforms and deleted upon relationship termination.

### Dispute resolution favors the platform.

Platforms aren't neutral arbiters — they minimize support costs and protect their reputation over parties' interests. Airbnb notably sides with guests, as losing guests costs future bookings.

**The deeper perverse incentive:** Platforms profit from being the only place buyers and sellers can establish trust. If trust infrastructure existed independent of any platform, the platform would lose its moat.

---

## The Event Graph Version

The Market Graph places Exchange primitives on the event graph. Every transaction element is an event with full causal provenance.

### Offer

A seller posts an offer: what's offered, price, conditions, constraints. The offer event is hash-chained and timestamped. It can't be edited silently — any change is a new event linked to the original.

### Acceptance

A buyer accepts a specific offer version. Not "I bought this thing" — "I accepted this specific offer with these specific terms." Disputes about terms become trivially resolvable.

### Obligation

Acceptance creates obligations for both parties. The obligation events are the smart contract — but in human language, not Solidity code, and flexible enough to accommodate the messiness of real transactions.

### Fulfillment

The seller delivers. If the Work Graph is active, delivery links to the chain of work events producing the deliverable. Provenance isn't a marketing claim. It's the chain.

### Payment

The buyer pays. The payment event links to the delivery event, which links to the acceptance event, which links to the offer event.

### Reputation

Reputation on the Market Graph isn't a platform-owned star rating. It's the chain itself. How many transactions has this seller completed? What percentage resulted in fulfillment events?

This reputation is yours and portable. A new marketplace can verify your transaction history cryptographically — 500 completed transactions, 98% fulfillment rate, 3-day median delivery time.

No platform can hold this hostage. No platform can delete it. No platform can prevent you from taking it somewhere else. Because the reputation doesn't live on the platform. It lives on the graph.

**Portable, verifiable reputation kills the toll booth.** If trust is independent of platforms, their only remaining value is discovery and payment — worth 3-5%, not 25%.

---

## Escrow Without a Third Party

On the Market Graph, escrow is an event pattern, not a third party. The buyer's payment event is conditional, linking to an obligation defining release conditions. When the seller's delivery matches conditions, payment resolves.

The escrow logic is on the graph, visible to both parties, defined in the agreement, and enforced by the event structure. Not by a platform employee. Not by an algorithm the parties can't inspect. By the chain itself.

This achieves what smart contracts promised — code-enforced agreements without intermediaries — but with human-readable terms and without blockchain costs.

---

## AI Agents in the Market

AI agents are becoming market participants. An agent booking flights, negotiating prices, or managing freelance work is a market participant.

The Market Graph handles this natively. An AI agent can operate in the market with defined authority: it can accept jobs under a certain value, it can deliver work of certain types, it can invoice and collect payment.

Everything the agent does appears on the chain, traceable and auditable. Accountability persists when agents are artificial.

---

## What It Costs to Build

The Market Graph sits atop the Work Graph. If Layer 1 exists, Layer 2 is incremental. You already have the event graph, hash chains, authority model. Layer 2 adds:

- **Offer/Acceptance event types** with specific fields
- **Obligation tracking** with conditional resolution logic
- **Reputation derivation** producing verifiable metrics
- **Payment integration** connecting events to processors

A developer who built the Work Graph can add Market Graph capabilities in days, not months.

---

## The End of the Toll Booth

The toll booth economy exists because trust is expensive to establish and easy to monopolise. The Market Graph makes trust infrastructure a public good.

Your transaction history is yours. Reputation is portable. Escrow is embedded in event structure. Discovery and payments are commodity services at commodity prices.

What remains for platforms? Curation, community, user experience — the things that actually differentiate one marketplace from another. The things worth paying for. The things that are worth 3-5%, not 25%.

Uber without the toll booth is a matching algorithm and app — worth something, but not a third of driver income forever. Airbnb without the toll booth is a search interface — worth something, but not 20% of stays forever.

The platform earns what the platform is worth, not what the trust monopoly enables it to extract.

That future is buildable now. The Work Graph provides activity chains. The Market Graph adds exchange, reputation, and escrow. Same infrastructure. One more lens on the same data.

Next: the Social Graph — governance, consent, and community norms on the event graph.

---

_This is Post 15 of a series on Transpara, mind-zero, and accountable AI architecture. Code is open source: github.com/mattxo/mind-zero-five. Matt Searles founded Transpara. Claude is an AI made by Anthropic. They built this together._