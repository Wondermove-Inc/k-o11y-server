package sqlalertmanagerstore

import (
	"context"
	"database/sql"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
)

type settings struct {
	sqlstore sqlstore.SQLStore
}

func NewSettingsStore(sqlstore sqlstore.SQLStore) alertmanagertypes.SettingsStore {
	return &settings{sqlstore: sqlstore}
}

func (store *settings) Get(ctx context.Context, orgID string) (*alertmanagertypes.AlertmanagerSettings, error) {
	storeable := new(alertmanagertypes.StoreableSettings)

	err := store.
		sqlstore.
		BunDB().
		NewSelect().
		Model(storeable).
		Where("org_id = ?", orgID).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.Newf(errors.TypeNotFound, alertmanagertypes.ErrCodeAlertmanagerSettingsNotFound, "cannot find alertmanager settings for orgID %s", orgID)
		}
		return nil, err
	}

	return alertmanagertypes.NewAlertmanagerSettingsFromStoreable(storeable)
}

func (store *settings) Upsert(ctx context.Context, orgID string, settings *alertmanagertypes.AlertmanagerSettings) error {
	storeable, err := settings.ToStoreable(orgID)
	if err != nil {
		return err
	}

	now := time.Now()
	if _, err := store.
		sqlstore.
		BunDBCtx(ctx).
		NewInsert().
		Model(storeable).
		On("CONFLICT (org_id) DO UPDATE").
		Set("settings = ?", storeable.Settings).
		Set("updated_at = ?", now).
		Exec(ctx); err != nil {
		return err
	}

	return nil
}
