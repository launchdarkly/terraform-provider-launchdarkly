package launchdarkly

import (
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

// findContextKindByKey scans a context-kind list response for an item matching the given key.
// The LaunchDarkly REST API does not expose a single-GET endpoint for context kinds, so reads
// must always go through the project-scoped list.
func findContextKindByKey(items []ldapi.ContextKindRep, key string) (*ldapi.ContextKindRep, bool) {
	for i := range items {
		if items[i].Key == key {
			return &items[i], true
		}
	}
	return nil, false
}

// buildUpsertContextKindPayload assembles the request body for PutContextKind from the
// provider-side intent: a required name plus optional description / hideInTargeting / archived.
// Nil pointers are emitted only when the caller passes nil (i.e. attribute unset in plan), so
// the server preserves whatever it already had.
func buildUpsertContextKindPayload(name string, description *string, hideInTargeting *bool, archived *bool) ldapi.UpsertContextKindPayload {
	payload := ldapi.UpsertContextKindPayload{Name: name}
	if description != nil {
		payload.Description = description
	}
	if hideInTargeting != nil {
		payload.HideInTargeting = hideInTargeting
	}
	if archived != nil {
		payload.Archived = archived
	}
	return payload
}
