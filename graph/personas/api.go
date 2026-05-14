package personas

import "sort"

// AllNames returns every persona name known at compile time, sourced from the
// generated HiveStatus map. Names are returned sorted for deterministic output
// (warm-up logs, pool stats, snapshot tests).
func AllNames() []string {
	names := make([]string, 0, len(HiveStatus))
	for name := range HiveStatus {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
