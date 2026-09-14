package server

import (
	"fmt"

	ingestionv1 "github.com/galaxy-io/filament/api/ingestion/v1"
	"github.com/galaxy-io/filament/internal/notifier"
)

func notifierFromInput(in *ingestionv1.NotifierInput) (*ingestionv1.Notifier, error) {
	if in == nil {
		return nil, fmt.Errorf("notifier is required")
	}
	switch in.GetNotificationType() {
	case ingestionv1.NotificationType_NOTIFICATION_TYPE_WEBHOOK:
	default:
		return nil, fmt.Errorf("unsupported notification type")
	}
	return notifier.NormalizeAndValidate(&ingestionv1.Notifier{
		Name: in.GetName(), NotificationType: in.GetNotificationType(), IsEnabled: in.GetIsEnabled(),
		Events: in.GetEvents(), Resources: in.GetResources(),
	})
}
