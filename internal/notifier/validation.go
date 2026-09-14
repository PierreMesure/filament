package notifier

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	ingestionv1 "github.com/galaxy-io/filament/api/ingestion/v1"
)

// NormalizeAndValidate returns a rule with sorted, unique filters.
// Destination validation and secret ownership checks belong to the selected channel and API.
func NormalizeAndValidate(n *ingestionv1.Notifier) (*ingestionv1.Notifier, error) {
	if n == nil {
		return nil, fmt.Errorf("notifier: rule is required")
	}
	if _, err := NotificationTypeLabel(n.GetNotificationType()); err != nil {
		return nil, fmt.Errorf("notifier: %w", err)
	}
	n.Name = strings.TrimSpace(n.Name)
	if !utf8.ValidString(n.Name) {
		return nil, fmt.Errorf("notifier: name must be valid text")
	}
	n.Events = unique(n.Events)
	if len(n.Events) == 0 {
		return nil, fmt.Errorf("notifier: at least one event is required")
	}
	if slices.Contains(n.Events, "*") && len(n.Events) != 1 {
		return nil, fmt.Errorf("notifier: wildcard cannot be combined with event names")
	}
	for _, name := range n.Events {
		if name != "*" && !IsExported(name) {
			return nil, fmt.Errorf("notifier: events must be exported event names")
		}
	}
	if len(n.GetResources()) != 0 {
		return nil, fmt.Errorf("notifier: resource filters are not supported for run events")
	}
	return n, nil
}

// IsLifecycleEvent identifies notification reports, which must never trigger notifications.
func IsLifecycleEvent(name string) bool { return strings.HasPrefix(name, "notifier.") }

// Matches checks a live, enabled rule against an event name and exact resource name.
func Matches(n *ingestionv1.Notifier, eventName, resource string) bool {
	if n == nil || !n.GetIsEnabled() || n.GetDeletedAt() != 0 || eventName == "" || IsLifecycleEvent(eventName) {
		return false
	}
	if !slices.Contains(n.GetEvents(), "*") && !slices.Contains(n.GetEvents(), eventName) {
		return false
	}
	return len(n.GetResources()) == 0 || resource != "" && slices.Contains(n.GetResources(), resource)
}

func unique(values []string) []string {
	out := append([]string{}, values...)
	slices.Sort(out)
	return slices.Compact(out)
}
