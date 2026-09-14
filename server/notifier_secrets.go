package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/galaxy-io/filament"
	ingestionv1 "github.com/galaxy-io/filament/api/ingestion/v1"
	"github.com/galaxy-io/filament/internal/notifier/webhook"
)

// notifierDestination returns the selected reference and whether this request wrote it.
func (a *Server) notifierDestination(ctx context.Context, n *ingestionv1.Notifier, in *ingestionv1.WebhookNotifierInput) (string, bool, error) {
	switch source := in.GetDestinationSource().(type) {
	case *ingestionv1.WebhookNotifierInput_Destination:
		destination, err := webhook.NormalizeAndValidateDestination(webhook.Destination{
			URL: source.Destination.GetUrl(), Headers: source.Destination.GetHeaders(),
		})
		if err != nil {
			return "", false, connect.NewError(connect.CodeInvalidArgument, err)
		}
		ref, err := a.writeNotifierDestination(ctx, n, destination)
		return ref, err == nil, err
	case *ingestionv1.WebhookNotifierInput_DestinationSecretRef:
		ref := source.DestinationSecretRef
		if err := a.validateNotifierDestinationRef(ctx, n, ref); err != nil {
			return "", false, err
		}
		return ref, false, nil
	default:
		if ref := n.GetSecretRefs()["destination"]; ref != "" {
			return ref, false, nil
		}
		return "", false, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("webhook destination or destination_secret_ref is required"))
	}
}

func (a *Server) writeNotifierDestination(ctx context.Context, n *ingestionv1.Notifier, destination webhook.Destination) (string, error) {
	if a.secrets == nil {
		return "", connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("a secrets provider is required"))
	}
	value, err := json.Marshal(destination)
	if err != nil {
		return "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("encode webhook destination: %w", err))
	}
	ref := notifierSecretPrefix(n) + uuid.NewString()
	if err := a.secrets.Write(ctx, ref, filament.Secret{Tenant: filament.TenantID(n.GetTenantId()), Value: value}); err != nil {
		a.deleteNotifierSecret(ctx, n, ref)
		return "", connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("could not store webhook destination; inline destinations require a writable secrets provider"))
	}
	return ref, nil
}

func (a *Server) validateNotifierDestinationRef(ctx context.Context, n *ingestionv1.Notifier, ref string) error {
	tenant := filament.TenantID(n.GetTenantId())
	if ref == "" || filament.ValidateConnectionSecretRef(ref, tenant) != nil {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid destination secret reference"))
	}
	// Only the current managed reference may be reused. Old revisions can be
	// undergoing cleanup, and another notifier's secret has its own lifecycle.
	if strings.HasPrefix(ref, filament.ConnectionSecretPrefix) &&
		(ref != n.GetSecretRefs()["destination"] || !ownsNotifierSecret(n, ref)) {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("managed destination reference must be this notifier's current reference"))
	}
	if a.secrets == nil {
		return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("a secrets provider is required"))
	}
	secret, err := a.secrets.Read(ctx, ref)
	if err != nil || secret.Tenant != "" && secret.Tenant != tenant {
		return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("could not resolve destination secret"))
	}
	if _, err := webhook.ParseDestination(secret.Value); err != nil {
		return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("invalid webhook destination secret: %w", err))
	}
	return nil
}

func notifierSecretPrefix(n *ingestionv1.Notifier) string {
	return fmt.Sprintf("%s%s/notifier/%s/destination/", filament.ConnectionSecretPrefix, n.GetTenantId(), n.GetId())
}

func ownsNotifierSecret(n *ingestionv1.Notifier, ref string) bool {
	revision, ok := strings.CutPrefix(ref, notifierSecretPrefix(n))
	return ok && uuid.Validate(revision) == nil
}

func (a *Server) deleteNotifierSecret(ctx context.Context, n *ingestionv1.Notifier, ref string) {
	if a.secrets == nil || !ownsNotifierSecret(n, ref) {
		return
	}
	// A disconnected client must not prevent cleanup after a completed write.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := a.secrets.Delete(ctx, ref); err != nil && !errors.Is(err, filament.ErrNotFound) && a.log != nil {
		a.log.Warn("notifier secret cleanup failed",
			filament.Field{Key: "tenant.id", Value: n.GetTenantId()},
			filament.Field{Key: "pipeline.id", Value: n.GetPipelineId()},
			filament.Field{Key: "notifier.id", Value: n.GetId()})
	}
}

func (a *Server) deletePipelineNotifierSecrets(ctx context.Context, tenant filament.TenantID, pipelineID string) {
	if a.secrets == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	// Read after pipeline deletion commits so concurrent notifier updates cannot
	// introduce references that cleanup misses.
	rules, err := a.store.ListNotifiers(ctx, tenant, pipelineID, true)
	if err != nil {
		if a.log != nil {
			a.log.Warn("could not list notifier secrets for cleanup",
				filament.Field{Key: "tenant.id", Value: tenant},
				filament.Field{Key: "pipeline.id", Value: pipelineID})
		}
		return
	}
	for _, n := range rules {
		a.deleteNotifierSecret(ctx, n, n.GetSecretRefs()["destination"])
	}
}
