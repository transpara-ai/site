package graph

// Bridge view stubs — replaced by templ-generated code in Task 3.
// These exist so bridge_handlers.go compiles before the templ views are written.

import "github.com/a-h/templ"

func BridgeIndexPage(pending []BridgeAction, recent []BridgeAction, user ViewUser) templ.Component {
	return templ.NopComponent
}

func BridgeActionFeed(pending []BridgeAction) templ.Component {
	return templ.NopComponent
}

func BridgeActionDetailPage(action *BridgeAction, user ViewUser) templ.Component {
	return templ.NopComponent
}

func BridgeAgentsPage(agents []string, user ViewUser) templ.Component {
	return templ.NopComponent
}

func BridgeAgentDetailPage(name string, events []BridgeEvent, user ViewUser) templ.Component {
	return templ.NopComponent
}

func BridgePreferencesPage(prefs []NotifyPreference, user ViewUser) templ.Component {
	return templ.NopComponent
}
