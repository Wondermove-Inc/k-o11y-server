package alertmanager

import (
	"context"
	"net/url"
	"sync"

	"github.com/prometheus/alertmanager/featurecontrol"
	"github.com/prometheus/alertmanager/matcher/compat"

	"github.com/SigNoz/signoz/pkg/alertmanager/alertmanagerserver"
	"github.com/SigNoz/signoz/pkg/alertmanager/nfmanager"
	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/modules/organization"
	"github.com/SigNoz/signoz/pkg/modules/preference"
	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/SigNoz/signoz/pkg/types/preferencetypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type Service struct {
	// config is the config for the alertmanager service
	config alertmanagerserver.Config

	// stateStore is the state store for the alertmanager service
	stateStore alertmanagertypes.StateStore

	// configStore is the config store for the alertmanager service
	configStore alertmanagertypes.ConfigStore

	// settingsStore is the settings store for alertmanager overrides
	settingsStore alertmanagertypes.SettingsStore

	// organization is the organization module for the alertmanager service
	orgGetter organization.Getter
	// preference is the preference module for org preferences
	preference preference.Module

	// settings is the settings for the alertmanager service
	settings factory.ScopedProviderSettings

	// Map of organization id to alertmanager server
	servers map[string]*alertmanagerserver.Server

	// Mutex to protect the servers map
	serversMtx sync.RWMutex

	notificationManager nfmanager.NotificationManager
}

func New(
	ctx context.Context,
	settings factory.ScopedProviderSettings,
	config alertmanagerserver.Config,
	stateStore alertmanagertypes.StateStore,
	configStore alertmanagertypes.ConfigStore,
	settingsStore alertmanagertypes.SettingsStore,
	orgGetter organization.Getter,
	preferenceModule preference.Module,
	nfManager nfmanager.NotificationManager,
) *Service {
	service := &Service{
		config:              config,
		stateStore:          stateStore,
		configStore:         configStore,
		settingsStore:       settingsStore,
		orgGetter:           orgGetter,
		preference:          preferenceModule,
		settings:            settings,
		servers:             make(map[string]*alertmanagerserver.Server),
		serversMtx:          sync.RWMutex{},
		notificationManager: nfManager,
	}

	return service
}

func (service *Service) SyncServers(ctx context.Context) error {
	compat.InitFromFlags(service.settings.Logger(), featurecontrol.NoopFlags{})
	orgs, err := service.orgGetter.ListByOwnedKeyRange(ctx)
	if err != nil {
		return err
	}

	service.serversMtx.Lock()
	for _, org := range orgs {
		config, err := service.getConfig(ctx, org.ID.StringValue())
		if err != nil {
			service.settings.Logger().ErrorContext(ctx, "failed to get alertmanager config for org", "org_id", org.ID.StringValue(), "error", err)
			continue
		}

		// If the server is not present, create it and sync the config
		if _, ok := service.servers[org.ID.StringValue()]; !ok {
			server, err := service.newServer(ctx, org.ID.StringValue())
			if err != nil {
				service.settings.Logger().ErrorContext(ctx, "failed to create alertmanager server", "org_id", org.ID.StringValue(), "error", err)
				continue
			}

			service.servers[org.ID.StringValue()] = server
		}

		externalURLUpdated := service.updateServerExternalURL(ctx, org.ID.StringValue(), service.servers[org.ID.StringValue()], false)
		if service.servers[org.ID.StringValue()].Hash() == config.StoreableConfig().Hash && !externalURLUpdated {
			service.settings.Logger().DebugContext(ctx, "skipping alertmanager sync for org", "org_id", org.ID.StringValue(), "hash", config.StoreableConfig().Hash)
			continue
		}

		err = service.servers[org.ID.StringValue()].SetConfig(ctx, config)
		if err != nil {
			service.settings.Logger().ErrorContext(ctx, "failed to set config for alertmanager server", "org_id", org.ID.StringValue(), "error", err)
			continue
		}
	}
	service.serversMtx.Unlock()

	return nil
}

func (service *Service) GetAlerts(ctx context.Context, orgID string, params alertmanagertypes.GettableAlertsParams) (alertmanagertypes.DeprecatedGettableAlerts, error) {
	service.serversMtx.RLock()
	defer service.serversMtx.RUnlock()

	server, err := service.getServer(orgID)
	if err != nil {
		return nil, err
	}

	service.updateServerExternalURL(ctx, orgID, server, false)

	alerts, err := server.GetAlerts(ctx, params)
	if err != nil {
		return nil, err
	}

	return alertmanagertypes.NewDeprecatedGettableAlertsFromGettableAlerts(alerts), nil
}

func (service *Service) GetSettings(ctx context.Context, orgID string) (*alertmanagertypes.AlertmanagerSettingsResponse, error) {
	settings, err := service.getSettingsOrDefaults(ctx, orgID)
	if err != nil {
		return nil, err
	}

	response := settings.ToResponse()
	return &response, nil
}

func (service *Service) UpdateSettings(ctx context.Context, orgID string, update alertmanagertypes.AlertmanagerSettingsUpdate) (*alertmanagertypes.AlertmanagerSettingsResponse, error) {
	if service.settingsStore == nil {
		return nil, errors.New(errors.TypeUnsupported, errors.CodeUnsupported, "alertmanager settings store not configured")
	}

	settings, err := service.getSettingsOrDefaults(ctx, orgID)
	if err != nil {
		return nil, err
	}

	settings.ApplyUpdate(update)
	if err := settings.Validate(); err != nil {
		return nil, err
	}

	if err := service.settingsStore.Upsert(ctx, orgID, settings); err != nil {
		return nil, err
	}

	config, err := service.getConfig(ctx, orgID)
	if err != nil {
		return nil, err
	}

	config, err = service.compareAndSelectConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	if err := service.configStore.Set(ctx, config); err != nil {
		return nil, err
	}

	service.serversMtx.RLock()
	server := service.servers[orgID]
	service.serversMtx.RUnlock()
	if server != nil {
		if err := server.SetConfig(ctx, config); err != nil {
			return nil, err
		}
	}

	response := settings.ToResponse()
	return &response, nil
}

func (service *Service) PutAlerts(ctx context.Context, orgID string, alerts alertmanagertypes.PostableAlerts) error {
	service.serversMtx.RLock()
	defer service.serversMtx.RUnlock()

	server, err := service.getServer(orgID)
	if err != nil {
		return err
	}

	service.updateServerExternalURL(ctx, orgID, server, true)

	return server.PutAlerts(ctx, alerts)
}

func (service *Service) TestReceiver(ctx context.Context, orgID string, receiver alertmanagertypes.Receiver) error {
	service.serversMtx.RLock()
	defer service.serversMtx.RUnlock()

	server, err := service.getServer(orgID)
	if err != nil {
		return err
	}

	service.updateServerExternalURL(ctx, orgID, server, true)

	return server.TestReceiver(ctx, receiver)
}

func (service *Service) TestAlert(ctx context.Context, orgID string, receiversMap map[*alertmanagertypes.PostableAlert][]string, config *alertmanagertypes.NotificationConfig) error {
	service.serversMtx.RLock()
	defer service.serversMtx.RUnlock()

	server, err := service.getServer(orgID)
	if err != nil {
		return err
	}

	service.updateServerExternalURL(ctx, orgID, server, true)

	return server.TestAlert(ctx, receiversMap, config)
}

func (service *Service) Stop(ctx context.Context) error {
	var errs []error
	for _, server := range service.servers {
		if err := server.Stop(ctx); err != nil {
			errs = append(errs, err)
			service.settings.Logger().ErrorContext(ctx, "failed to stop alertmanager server", "error", err)
		}
	}

	return errors.Join(errs...)
}

func (service *Service) updateServerExternalURL(ctx context.Context, orgID string, server *alertmanagerserver.Server, applyConfig bool) bool {
	if service.preference == nil || server == nil {
		return false
	}

	orgUUID, err := valuer.NewUUID(orgID)
	if err != nil {
		service.settings.Logger().WarnContext(ctx, "invalid org id for alert base url", "org_id", orgID, "error", err)
		return false
	}

	orgPreference, err := service.preference.GetByOrg(ctx, orgUUID, preferencetypes.NameAlertBaseURL)
	if err != nil {
		service.settings.Logger().WarnContext(ctx, "failed to fetch alert base url preference", "org_id", orgID, "error", err)
		return false
	}

	baseURL, err := orgPreference.Value.StringValue()
	if err != nil {
		service.settings.Logger().WarnContext(ctx, "alert base url preference is not a string", "org_id", orgID, "error", err)
		return false
	}

	normalized, err := preferencetypes.NormalizeAlertBaseURL(baseURL)
	if err != nil {
		service.settings.Logger().WarnContext(ctx, "invalid alert base url preference value", "org_id", orgID, "error", err)
		return false
	}

	externalURL, err := url.Parse(normalized)
	if err != nil {
		service.settings.Logger().WarnContext(ctx, "failed to parse alert base url", "org_id", orgID, "error", err)
		return false
	}

	updated := server.UpdateExternalURL(externalURL)
	if !updated {
		return false
	}

	if applyConfig {
		if err := server.ReloadConfig(ctx); err != nil {
			service.settings.Logger().WarnContext(ctx, "failed to reload alertmanager config for alert base url", "org_id", orgID, "error", err)
		}
	}

	return updated
}

func (service *Service) newServer(ctx context.Context, orgID string) (*alertmanagerserver.Server, error) {
	config, err := service.getConfig(ctx, orgID)
	if err != nil {
		return nil, err
	}

	server, err := alertmanagerserver.New(ctx, service.settings.Logger(), service.settings.PrometheusRegisterer(), service.config, orgID, service.stateStore, service.notificationManager)
	if err != nil {
		return nil, err
	}

	beforeCompareAndSelectHash := config.StoreableConfig().Hash
	config, err = service.compareAndSelectConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	if beforeCompareAndSelectHash == config.StoreableConfig().Hash {
		service.settings.Logger().DebugContext(ctx, "skipping config store update for org", "org_id", orgID, "hash", config.StoreableConfig().Hash)
		return server, nil
	}

	err = service.configStore.Set(ctx, config)
	if err != nil {
		return nil, err
	}

	return server, nil
}

func (service *Service) getConfig(ctx context.Context, orgID string) (*alertmanagertypes.Config, error) {
	globalConfig, routeConfig := service.getGlobalRouteConfig(ctx, orgID)
	config, err := service.configStore.Get(ctx, orgID)
	if err != nil {
		if !errors.Ast(err, errors.TypeNotFound) {
			return nil, err
		}

		config, err = alertmanagertypes.NewDefaultConfig(globalConfig, routeConfig, orgID)
		if err != nil {
			return nil, err
		}
	}

	if err := config.SetGlobalConfig(globalConfig); err != nil {
		return nil, err
	}
	if err := config.SetRouteConfig(routeConfig); err != nil {
		return nil, err
	}

	return config, nil
}

func (service *Service) compareAndSelectConfig(ctx context.Context, incomingConfig *alertmanagertypes.Config) (*alertmanagertypes.Config, error) {
	channels, err := service.configStore.ListChannels(ctx, incomingConfig.StoreableConfig().OrgID)
	if err != nil {
		return nil, err
	}

	matchers, err := service.configStore.GetMatchers(ctx, incomingConfig.StoreableConfig().OrgID)
	if err != nil {
		return nil, err
	}

	globalConfig, routeConfig := service.getGlobalRouteConfig(ctx, incomingConfig.StoreableConfig().OrgID)
	config, err := alertmanagertypes.NewConfigFromChannels(globalConfig, routeConfig, channels, incomingConfig.StoreableConfig().OrgID)
	if err != nil {
		return nil, err
	}

	for ruleID, receivers := range matchers {
		err = config.CreateRuleIDMatcher(ruleID, receivers)
		if err != nil {
			return nil, err
		}
	}

	if incomingConfig.StoreableConfig().Hash != config.StoreableConfig().Hash {
		service.settings.Logger().InfoContext(ctx, "mismatch found, updating config to match channels and matchers")
		return config, nil
	}

	return incomingConfig, nil

}

func (service *Service) getSettingsOrDefaults(ctx context.Context, orgID string) (*alertmanagertypes.AlertmanagerSettings, error) {
	if service.settingsStore == nil {
		settings := alertmanagertypes.NewAlertmanagerSettingsFromConfig(service.config.Global, service.config.Route)
		return &settings, nil
	}

	settings, err := service.settingsStore.Get(ctx, orgID)
	if err != nil {
		if errors.Ast(err, errors.TypeNotFound) {
			defaults := alertmanagertypes.NewAlertmanagerSettingsFromConfig(service.config.Global, service.config.Route)
			return &defaults, nil
		}
		return nil, err
	}

	return settings, nil
}

func (service *Service) getGlobalRouteConfig(ctx context.Context, orgID string) (alertmanagertypes.GlobalConfig, alertmanagertypes.RouteConfig) {
	globalConfig := service.config.Global
	routeConfig := service.config.Route

	if service.settingsStore == nil {
		return globalConfig, routeConfig
	}

	settings, err := service.settingsStore.Get(ctx, orgID)
	if err != nil {
		if !errors.Ast(err, errors.TypeNotFound) {
			service.settings.Logger().WarnContext(ctx, "failed to load alertmanager settings, using defaults", "org_id", orgID, "error", err)
		}
		return globalConfig, routeConfig
	}

	if err := settings.Validate(); err != nil {
		service.settings.Logger().WarnContext(ctx, "invalid alertmanager settings, using defaults", "org_id", orgID)
		return globalConfig, routeConfig
	}

	if err := settings.ApplyToGlobal(&globalConfig); err != nil {
		service.settings.Logger().WarnContext(ctx, "failed to apply alertmanager global settings, using defaults", "org_id", orgID)
		return globalConfig, routeConfig
	}

	if err := settings.ApplyToRoute(&routeConfig); err != nil {
		service.settings.Logger().WarnContext(ctx, "failed to apply alertmanager route settings, using defaults", "org_id", orgID)
		return globalConfig, routeConfig
	}

	return globalConfig, routeConfig
}

// getServer returns the server for the given orgID. It should be called with the lock held.
func (service *Service) getServer(orgID string) (*alertmanagerserver.Server, error) {
	server, ok := service.servers[orgID]
	if !ok {
		return nil, errors.Newf(errors.TypeNotFound, ErrCodeAlertmanagerNotFound, "alertmanager not found for org %s", orgID)
	}

	return server, nil
}
