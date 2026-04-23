package content

import "github.com/transpara-ai/site/views"

// LoadGoals returns the derived goals for all layers.
// These are permanent — derived from The Weight (suffering) + primitives (capability).
// Each goal states what must be true in the world so the suffering can't happen.
func LoadGoals() map[int][]views.Goal {
	return map[int][]views.Goal{
		0: {
			{ID: "g0.1", Title: "Every significant action can be recorded as an immutable, hash-chained event"},
			{ID: "g0.2", Title: "Causal chains between events are explicit and traversable"},
			{ID: "g0.3", Title: "Trust is earned through verified action, not declared by authority"},
		},
		1: {
			{ID: "g1.1", Title: "Every act of work can be recorded and attributed to its actor"},
			{ID: "g1.2", Title: "Workers own their work history — portable, queryable, not locked in an employer's database"},
			{ID: "g1.3", Title: "AI and human work exist on the same accountable infrastructure"},
		},
		2: {
			{ID: "g2.1", Title: "Value exchange between parties is direct — no mandatory intermediary rent"},
			{ID: "g2.2", Title: "Reputation is portable — follows the worker, not the platform"},
			{ID: "g2.3", Title: "Market participation doesn't require permission from gatekeepers"},
		},
		3: {
			{ID: "g3.1", Title: "Communities set their own norms — not platforms, not algorithms"},
			{ID: "g3.2", Title: "The rules that govern a community are visible to its members"},
			{ID: "g3.3", Title: "Participation is not gated by a platform's business model"},
		},
		4: {
			{ID: "g4.1", Title: "Evidence is structural — interactions on the graph ARE the evidence"},
			{ID: "g4.2", Title: "Disputes under $500 are economically resolvable — not just rich-people justice"},
			{ID: "g4.3", Title: "Accountability chains are visible — who decided what, when, why"},
		},
		5: {
			{ID: "g5.1", Title: "Hypotheses are registered before experiments — pre-registration is structural"},
			{ID: "g5.2", Title: "Analysis history is visible — not just the final successful run"},
			{ID: "g5.3", Title: "Negative results are published — the graph records failure, not just success"},
		},
		6: {
			{ID: "g6.1", Title: "Claims have provenance — who said it, based on what evidence, challenged by whom"},
			{ID: "g6.2", Title: "Correction is structural — wrong answers aren't deleted, they're superseded with causal links"},
			{ID: "g6.3", Title: "Knowledge access doesn't depend on geography or wealth"},
		},
		7: {
			{ID: "g7.1", Title: "Every automated decision is visible — confidence levels, authority chains, inputs"},
			{ID: "g7.2", Title: "Harm patterns are detectable across layers — not siloed in one institution"},
			{ID: "g7.3", Title: "Raising concerns is structurally protected, not career-ending"},
		},
		8: {
			{ID: "g8.1", Title: "Identity emerges from action history, not from categories assigned by others"},
			{ID: "g8.2", Title: "Selective disclosure — share what you choose, not what platforms extract"},
			{ID: "g8.3", Title: "No system can reduce a person to a single dimension"},
		},
		9: {
			{ID: "g9.1", Title: "Consent is continuous, not a one-time checkbox"},
			{ID: "g9.2", Title: "Relationship infrastructure exists that doesn't profit from loneliness"},
			{ID: "g9.3", Title: "Betrayal, repair, and forgiveness are modeled — not just connection and disconnection"},
		},
		10: {
			{ID: "g10.1", Title: "Communities own their own infrastructure — belonging survives platform death"},
			{ID: "g10.2", Title: "Belonging is a gradient, not binary"},
			{ID: "g10.3", Title: "Community memory persists — even when members leave"},
		},
		11: {
			{ID: "g11.1", Title: "Who decides, why, and whom it affects is visible"},
			{ID: "g11.2", Title: "Decision chains are traceable — from policy to impact"},
			{ID: "g11.3", Title: "Governance applies equally to corporations, platforms, governments, and AI"},
		},
		12: {
			{ID: "g12.1", Title: "Cultural artifacts have provenance — derivation chains show where meaning came from"},
			{ID: "g12.2", Title: "Algorithms serve meaning, not engagement"},
			{ID: "g12.3", Title: "Living traditions can be recorded without being reduced"},
		},
		13: {
			{ID: "g13.1", Title: "Economic output and ecological cost exist on the same graph — externalization is structurally impossible"},
			{ID: "g13.2", Title: "The cascade between layers is visible — you can trace despair to its infrastructure root"},
			{ID: "g13.3", Title: "Wellbeing infrastructure exists — not just crisis response"},
		},
	}
}
