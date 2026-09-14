package notifier

import (
	"slices"

	"github.com/galaxy-io/filament/events"
)

// Exported are the event kinds a notifier may deliver. The catalog stays
// internal; each kind here has a webhook shape in Project.
var Exported = []string{events.RunCompleted.Name(), events.RunFailed.Name()}

// IsExported reports whether name may trigger a notification.
func IsExported(name string) bool { return slices.Contains(Exported, name) }
