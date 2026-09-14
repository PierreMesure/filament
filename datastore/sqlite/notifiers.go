package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/galaxy-io/filament"
	ingestionv1 "github.com/galaxy-io/filament/api/ingestion/v1"
	"github.com/galaxy-io/filament/datastore/sqlite/sqlcgen"
)

// CreateNotifier adds a rule at version 1.
func (s *Store) CreateNotifier(ctx context.Context, n *ingestionv1.Notifier) (*ingestionv1.Notifier, error) {
	if n == nil {
		return nil, fmt.Errorf("create notifier: notifier is required")
	}
	kind, err := notifierTypeToRow(n.GetNotificationType())
	if err != nil {
		return nil, fmt.Errorf("create notifier %q: %w", n.GetId(), err)
	}
	data, err := marshalNotifier(n)
	if err != nil {
		return nil, fmt.Errorf("create notifier %q: %w", n.GetId(), err)
	}
	saved, err := s.writeNotifier(ctx, filament.TenantID(n.GetTenantId()), n.GetPipelineId(), n.GetId(), func(q *sqlcgen.Queries) (*sqlcgen.Notifier, error) {
		now := nowMillis()
		return q.CreateNotifier(ctx, sqlcgen.CreateNotifierParams{
			NotifierID: n.GetId(), TenantID: n.GetTenantId(), PipelineID: n.GetPipelineId(),
			CreatedAt: now, UpdatedAt: now,
			Name: n.GetName(), NotificationType: kind, IsEnabled: boolInt(n.GetIsEnabled()),
			Events: string(data.events), Resources: string(data.resources), Config: string(data.config), SecretRefs: string(data.refs),
			CreatedByUserID: sql.NullString{String: n.GetCreatedByUserId(), Valid: n.GetCreatedByUserId() != ""}, UpdatedByUserID: sql.NullString{String: n.GetUpdatedByUserId(), Valid: n.GetUpdatedByUserId() != ""},
		})
	})
	if err != nil {
		return nil, fmt.Errorf("create notifier %q: %w", n.GetId(), err)
	}
	return saved, nil
}

// LoadNotifier returns a rule, including deleted rules for secret cleanup.
func (s *Store) LoadNotifier(ctx context.Context, tenant filament.TenantID, pipelineID, id string) (*ingestionv1.Notifier, error) {
	row, err := s.q.GetNotifier(ctx, sqlcgen.GetNotifierParams{TenantID: string(tenant), PipelineID: pipelineID, NotifierID: id})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("load notifier %q: %w", id, filament.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("datastore/sqlite: load notifier %q: %w", id, err)
	}
	return notifierFromRow(row)
}

// ListNotifiers returns one pipeline's rules ordered by ID.
func (s *Store) ListNotifiers(ctx context.Context, tenant filament.TenantID, pipelineID string, includeDeleted bool) ([]*ingestionv1.Notifier, error) {
	rows, err := s.q.ListNotifiers(ctx, sqlcgen.ListNotifiersParams{
		TenantID: string(tenant), PipelineID: pipelineID, IncludeDeleted: includeDeleted,
	})
	if err != nil {
		return nil, fmt.Errorf("datastore/sqlite: list notifiers: %w", err)
	}
	out := make([]*ingestionv1.Notifier, 0, len(rows))
	for _, row := range rows {
		n, err := notifierFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

// UpdateNotifier replaces settings when the stored version matches.
func (s *Store) UpdateNotifier(ctx context.Context, n *ingestionv1.Notifier) (*ingestionv1.Notifier, error) {
	if n == nil {
		return nil, fmt.Errorf("update notifier: notifier is required")
	}
	kind, err := notifierTypeToRow(n.GetNotificationType())
	if err != nil {
		return nil, fmt.Errorf("update notifier %q at version %d: %w", n.GetId(), n.GetVersion(), err)
	}
	data, err := marshalNotifier(n)
	if err != nil {
		return nil, fmt.Errorf("update notifier %q at version %d: %w", n.GetId(), n.GetVersion(), err)
	}
	tenant := filament.TenantID(n.GetTenantId())
	saved, err := s.writeNotifier(ctx, tenant, n.GetPipelineId(), n.GetId(), func(q *sqlcgen.Queries) (*sqlcgen.Notifier, error) {
		row, err := notifierForUpdate(ctx, q, tenant, n.GetPipelineId(), n.GetId(), n.GetVersion())
		if err != nil {
			return nil, err
		}
		if row.NotificationType != kind {
			return nil, fmt.Errorf("notifier notification type cannot change")
		}
		return q.UpdateNotifier(ctx, sqlcgen.UpdateNotifierParams{
			TenantID: n.GetTenantId(), PipelineID: n.GetPipelineId(), NotifierID: n.GetId(), ExpectedVersion: n.GetVersion(),
			UpdatedAt: nowMillis(),
			Name:      n.GetName(), IsEnabled: boolInt(n.GetIsEnabled()), Events: string(data.events), Resources: string(data.resources),
			Config: string(data.config), SecretRefs: string(data.refs), UpdatedByUserID: sql.NullString{String: n.GetUpdatedByUserId(), Valid: n.GetUpdatedByUserId() != ""},
		})
	})
	if err != nil {
		return nil, fmt.Errorf("update notifier %q at version %d: %w", n.GetId(), n.GetVersion(), err)
	}
	return saved, nil
}

// DeleteNotifier marks a rule deleted and returns its metadata for secret cleanup.
func (s *Store) DeleteNotifier(ctx context.Context, tenant filament.TenantID, pipelineID, id string, version int64) (*ingestionv1.Notifier, error) {
	saved, err := s.writeNotifier(ctx, tenant, pipelineID, id, func(q *sqlcgen.Queries) (*sqlcgen.Notifier, error) {
		if _, err := notifierForUpdate(ctx, q, tenant, pipelineID, id, version); err != nil {
			return nil, err
		}
		now := nowMillis()
		return q.DeleteNotifier(ctx, sqlcgen.DeleteNotifierParams{
			TenantID: string(tenant), PipelineID: pipelineID, NotifierID: id, ExpectedVersion: version,
			DeletedAt: sql.NullInt64{Int64: now, Valid: true}, UpdatedAt: now,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("delete notifier %q at version %d: %w", id, version, err)
	}
	return saved, nil
}

// Lock the parent before changing a rule so pipeline deletion cannot race it.
// Decode the returned row before committing; callers receive exactly what they wrote.
func (s *Store) writeNotifier(ctx context.Context, tenant filament.TenantID, pipelineID, id string, write func(*sqlcgen.Queries) (*sqlcgen.Notifier, error)) (*ingestionv1.Notifier, error) {
	if tenant == "" || pipelineID == "" || id == "" {
		return nil, fmt.Errorf("notifier tenant, pipeline id, and id are required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("datastore/sqlite: begin notifier write: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := s.q.WithTx(tx)
	if _, err := q.LockNotifierPipeline(ctx, sqlcgen.LockNotifierPipelineParams{TenantID: string(tenant), PipelineID: pipelineID}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("pipeline %q: %w", pipelineID, filament.ErrNotFound)
		}
		return nil, fmt.Errorf("datastore/sqlite: lock notifier pipeline: %w", err)
	}
	row, err := write(q)
	if err != nil {
		return nil, fmt.Errorf("datastore/sqlite: write notifier: %w", err)
	}
	n, err := notifierFromRow(row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("datastore/sqlite: commit notifier write: %w", err)
	}
	return n, nil
}

func notifierForUpdate(ctx context.Context, q *sqlcgen.Queries, tenant filament.TenantID, pipelineID, id string, version int64) (*sqlcgen.Notifier, error) {
	row, err := q.GetNotifier(ctx, sqlcgen.GetNotifierParams{TenantID: string(tenant), PipelineID: pipelineID, NotifierID: id})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, filament.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if row.IsDeleted != 0 || row.Version != version {
		return nil, filament.ErrVersionConflict
	}
	return row, nil
}

func notifierFromRow(row *sqlcgen.Notifier) (*ingestionv1.Notifier, error) {
	kind, err := notifierTypeFromRow(row.NotificationType)
	if err != nil {
		return nil, err
	}
	n := &ingestionv1.Notifier{
		Id: row.ID, TenantId: row.TenantID, PipelineId: row.PipelineID,
		Name: row.Name, NotificationType: kind, IsEnabled: row.IsEnabled != 0, Version: row.Version,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, DeletedAt: row.DeletedAt.Int64,
		CreatedByUserId: row.CreatedByUserID.String, UpdatedByUserId: row.UpdatedByUserID.String, DeletedByUserId: row.DeletedByUserID.String,
	}
	if err := json.Unmarshal([]byte(row.Events), &n.Events); err != nil {
		return nil, fmt.Errorf("datastore/sqlite: unmarshal notifier %q events: %w", row.ID, err)
	}
	if err := json.Unmarshal([]byte(row.Resources), &n.Resources); err != nil {
		return nil, fmt.Errorf("datastore/sqlite: unmarshal notifier %q resources: %w", row.ID, err)
	}
	if err := json.Unmarshal([]byte(row.SecretRefs), &n.SecretRefs); err != nil {
		return nil, fmt.Errorf("datastore/sqlite: unmarshal notifier %q secret_refs: %w", row.ID, err)
	}
	return n, nil
}

func notifierTypeToRow(kind ingestionv1.NotificationType) (string, error) {
	switch kind {
	case ingestionv1.NotificationType_NOTIFICATION_TYPE_WEBHOOK:
		return "webhook", nil
	default:
		return "", fmt.Errorf("invalid notification type %d", kind)
	}
}

func notifierTypeFromRow(kind string) (ingestionv1.NotificationType, error) {
	switch kind {
	case "webhook":
		return ingestionv1.NotificationType_NOTIFICATION_TYPE_WEBHOOK, nil
	default:
		return ingestionv1.NotificationType_NOTIFICATION_TYPE_UNSPECIFIED, fmt.Errorf("invalid notification type %q", kind)
	}
}

type notifierJSON struct{ events, resources, config, refs []byte }

func marshalNotifier(n *ingestionv1.Notifier) (notifierJSON, error) {
	events := n.GetEvents()
	if events == nil {
		events = []string{}
	}
	resources := n.GetResources()
	if resources == nil {
		resources = []string{}
	}
	refs := n.GetSecretRefs()
	if refs == nil {
		refs = map[string]string{}
	}
	var data notifierJSON
	var err error
	data.events, err = json.Marshal(events)
	if err != nil {
		return notifierJSON{}, fmt.Errorf("datastore/sqlite: marshal notifier events: %w", err)
	}
	data.resources, err = json.Marshal(resources)
	if err != nil {
		return notifierJSON{}, fmt.Errorf("datastore/sqlite: marshal notifier resources: %w", err)
	}
	data.config, err = json.Marshal(map[string]any{})
	if err != nil {
		return notifierJSON{}, fmt.Errorf("datastore/sqlite: marshal notifier config: %w", err)
	}
	data.refs, err = json.Marshal(refs)
	if err != nil {
		return notifierJSON{}, fmt.Errorf("datastore/sqlite: marshal notifier secret_refs: %w", err)
	}
	return data, nil
}
