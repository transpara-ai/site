package graph

import "time"

type OpsCivilizationAssemblyData struct {
	GeneratedAt      string
	AuthoritySource  string
	ProjectionSource string
	ProjectionTarget string
	ProjectionStatus string
	Civilization     ObsCivilization
	Boundary         []OpsCivilizationBoundary
}

type OpsCivilizationBoundary struct {
	Label  string
	State  string
	Detail string
}

func buildOpsCivilizationAssemblyData() *OpsCivilizationAssemblyData {
	civ := buildObsCivilization(nil, nil)
	civ.Findings = append([]string{
		"EventGraph-backed Civilization Assembly projection is not connected in Site yet; this page uses typed Site fallback data for visualization only.",
		"Unknown, unavailable, and not-projected states are preserved instead of inferred from missing projection data.",
		"This route is display-only and carries no authority to execute, deploy, mutate protected settings, or allocate value.",
	}, civ.Findings...)

	return &OpsCivilizationAssemblyData{
		GeneratedAt:      time.Now().UTC().Format("2006-01-02 15:04:05 UTC"),
		AuthoritySource:  "docs#163 v4.0 Site Civilization Assembly authority packet",
		ProjectionSource: "typed Site fallback snapshot; no live EventGraph assembly payload fetched",
		ProjectionTarget: "EventGraph Civilization Assembly projection",
		ProjectionStatus: "non-authoritative visualization",
		Civilization:     civ,
		Boundary: []OpsCivilizationBoundary{
			{
				Label:  "Route authority",
				State:  "bounded",
				Detail: "One read-only Site surface authorized by the merged v4.0 packet.",
			},
			{
				Label:  "Registered method",
				State:  "GET only",
				Detail: "No mutation handler is registered for this page.",
			},
			{
				Label:  "Truth source",
				State:  "target unavailable",
				Detail: "EventGraph remains the intended backing source before this view may claim live truth.",
			},
			{
				Label:  "Runtime control",
				State:  "withheld",
				Detail: "No executor, deploy, protected-setting, Hive-write, or Work-mutation path is exposed.",
			},
		},
	}
}
