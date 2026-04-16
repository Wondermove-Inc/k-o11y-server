package sqlmigration

import (
	"context"
	"time"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

type addAlertmanagerSettings struct{}

func NewAddAlertmanagerSettingsFactory() factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_alertmanager_settings"), newAddAlertmanagerSettings)
}

func newAddAlertmanagerSettings(_ context.Context, _ factory.ProviderSettings, _ Config) (SQLMigration, error) {
	return &addAlertmanagerSettings{}, nil
}

func (migration *addAlertmanagerSettings) Register(migrations *migrate.Migrations) error {
	if err := migrations.Register(migration.Up, migration.Down); err != nil {
		return err
	}

	return nil
}

func (migration *addAlertmanagerSettings) Up(ctx context.Context, db *bun.DB) error {
	_, err := db.
		NewCreateTable().
		Model(&struct {
			bun.BaseModel `bun:"table:alertmanager_settings"`
			ID            uint64    `bun:"id,pk,autoincrement"`
			Settings      string    `bun:"settings,notnull,type:text"`
			CreatedAt     time.Time `bun:"created_at,notnull"`
			UpdatedAt     time.Time `bun:"updated_at,notnull"`
			OrgID         string    `bun:"org_id,notnull,unique"`
		}{}).
		ForeignKey(`("org_id") REFERENCES "organizations" ("id")`).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (migration *addAlertmanagerSettings) Down(ctx context.Context, db *bun.DB) error {
	_, err := db.
		NewDropTable().
		Table("alertmanager_settings").
		IfExists().
		Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
