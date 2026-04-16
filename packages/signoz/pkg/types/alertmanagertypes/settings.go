package alertmanagertypes

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	amconfig "github.com/prometheus/alertmanager/config"
	commoncfg "github.com/prometheus/common/config"
	"github.com/uptrace/bun"
)

var (
	ErrCodeAlertmanagerSettingsNotFound = errors.MustNewCode("alertmanager_settings_not_found")
)

type AlertmanagerRouteSettings struct {
	GroupWait      string `json:"group_wait"`
	GroupInterval  string `json:"group_interval"`
	RepeatInterval string `json:"repeat_interval"`
}

type AlertmanagerSMTPAuthSettings struct {
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
	Secret   string `json:"secret,omitempty"`
	Identity string `json:"identity"`
}

type AlertmanagerSMTPTLSSettings struct {
	Enabled            bool   `json:"enabled"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify"`
	CAFilePath         string `json:"ca_file_path"`
	CertFilePath       string `json:"cert_file_path"`
	KeyFilePath        string `json:"key_file_path"`
}

type AlertmanagerSMTPSettings struct {
	Address    string                       `json:"address"`
	From       string                       `json:"from"`
	Hello      string                       `json:"hello"`
	RequireTLS bool                         `json:"require_tls"`
	Auth       AlertmanagerSMTPAuthSettings `json:"auth"`
	TLS        AlertmanagerSMTPTLSSettings  `json:"tls"`
}

type AlertmanagerSettings struct {
	Route AlertmanagerRouteSettings `json:"route"`
	SMTP  AlertmanagerSMTPSettings  `json:"smtp"`
}

type AlertmanagerRouteSettingsUpdate struct {
	GroupWait      *string `json:"group_wait,omitempty"`
	GroupInterval  *string `json:"group_interval,omitempty"`
	RepeatInterval *string `json:"repeat_interval,omitempty"`
}

type AlertmanagerSMTPAuthSettingsUpdate struct {
	Username      *string `json:"username,omitempty"`
	Password      *string `json:"password,omitempty"`
	Secret        *string `json:"secret,omitempty"`
	Identity      *string `json:"identity,omitempty"`
	ClearPassword *bool   `json:"clear_password,omitempty"`
	ClearSecret   *bool   `json:"clear_secret,omitempty"`
}

type AlertmanagerSMTPTLSSettingsUpdate struct {
	Enabled            *bool   `json:"enabled,omitempty"`
	InsecureSkipVerify *bool   `json:"insecure_skip_verify,omitempty"`
	CAFilePath         *string `json:"ca_file_path,omitempty"`
	CertFilePath       *string `json:"cert_file_path,omitempty"`
	KeyFilePath        *string `json:"key_file_path,omitempty"`
}

type AlertmanagerSMTPSettingsUpdate struct {
	Address    *string                             `json:"address,omitempty"`
	From       *string                             `json:"from,omitempty"`
	Hello      *string                             `json:"hello,omitempty"`
	RequireTLS *bool                               `json:"require_tls,omitempty"`
	Auth       *AlertmanagerSMTPAuthSettingsUpdate `json:"auth,omitempty"`
	TLS        *AlertmanagerSMTPTLSSettingsUpdate  `json:"tls,omitempty"`
}

type AlertmanagerSettingsUpdate struct {
	Route *AlertmanagerRouteSettingsUpdate `json:"route,omitempty"`
	SMTP  *AlertmanagerSMTPSettingsUpdate  `json:"smtp,omitempty"`
}

type AlertmanagerSMTPAuthSettingsResponse struct {
	Username    string `json:"username"`
	Identity    string `json:"identity"`
	PasswordSet bool   `json:"password_set"`
	SecretSet   bool   `json:"secret_set"`
}

type AlertmanagerSMTPSettingsResponse struct {
	Address    string                               `json:"address"`
	From       string                               `json:"from"`
	Hello      string                               `json:"hello"`
	RequireTLS bool                                 `json:"require_tls"`
	Auth       AlertmanagerSMTPAuthSettingsResponse `json:"auth"`
	TLS        AlertmanagerSMTPTLSSettings          `json:"tls"`
}

type AlertmanagerSettingsResponse struct {
	Route AlertmanagerRouteSettings        `json:"route"`
	SMTP  AlertmanagerSMTPSettingsResponse `json:"smtp"`
}

type StoreableSettings struct {
	bun.BaseModel `bun:"table:alertmanager_settings"`

	ID        uint64    `bun:"id,pk,autoincrement"`
	Settings  string    `bun:"settings,notnull,type:text"`
	CreatedAt time.Time `bun:"created_at,notnull"`
	UpdatedAt time.Time `bun:"updated_at,notnull"`
	OrgID     string    `bun:"org_id,notnull,unique"`
}

type SettingsStore interface {
	Get(ctx context.Context, orgID string) (*AlertmanagerSettings, error)
	Upsert(ctx context.Context, orgID string, settings *AlertmanagerSettings) error
}

func NewAlertmanagerSettingsFromConfig(globalConfig GlobalConfig, routeConfig RouteConfig) AlertmanagerSettings {
	settings := AlertmanagerSettings{
		Route: AlertmanagerRouteSettings{
			GroupWait:      routeConfig.GroupWait.String(),
			GroupInterval:  routeConfig.GroupInterval.String(),
			RepeatInterval: routeConfig.RepeatInterval.String(),
		},
		SMTP: AlertmanagerSMTPSettings{
			Address:    globalConfig.SMTPSmarthost.String(),
			From:       globalConfig.SMTPFrom,
			Hello:      globalConfig.SMTPHello,
			RequireTLS: globalConfig.SMTPRequireTLS,
			Auth: AlertmanagerSMTPAuthSettings{
				Username: globalConfig.SMTPAuthUsername,
				Password: string(globalConfig.SMTPAuthPassword),
				Secret:   string(globalConfig.SMTPAuthSecret),
				Identity: globalConfig.SMTPAuthIdentity,
			},
		},
	}

	if globalConfig.SMTPTLSConfig != nil {
		settings.SMTP.TLS = AlertmanagerSMTPTLSSettings{
			Enabled:            true,
			InsecureSkipVerify: globalConfig.SMTPTLSConfig.InsecureSkipVerify,
			CAFilePath:         globalConfig.SMTPTLSConfig.CAFile,
			CertFilePath:       globalConfig.SMTPTLSConfig.CertFile,
			KeyFilePath:        globalConfig.SMTPTLSConfig.KeyFile,
		}
	}

	return settings
}

func NewAlertmanagerSettingsFromStoreable(storeable *StoreableSettings) (*AlertmanagerSettings, error) {
	settings := new(AlertmanagerSettings)
	if err := json.Unmarshal([]byte(storeable.Settings), settings); err != nil {
		return nil, err
	}
	return settings, nil
}

func (settings AlertmanagerSettings) ToStoreable(orgID string) (*StoreableSettings, error) {
	raw, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	return &StoreableSettings{
		Settings:  string(raw),
		CreatedAt: now,
		UpdatedAt: now,
		OrgID:     orgID,
	}, nil
}

func (settings *AlertmanagerSettings) ApplyUpdate(update AlertmanagerSettingsUpdate) {
	if update.Route != nil {
		if update.Route.GroupWait != nil {
			settings.Route.GroupWait = *update.Route.GroupWait
		}
		if update.Route.GroupInterval != nil {
			settings.Route.GroupInterval = *update.Route.GroupInterval
		}
		if update.Route.RepeatInterval != nil {
			settings.Route.RepeatInterval = *update.Route.RepeatInterval
		}
	}

	if update.SMTP != nil {
		if update.SMTP.Address != nil {
			settings.SMTP.Address = *update.SMTP.Address
		}
		if update.SMTP.From != nil {
			settings.SMTP.From = *update.SMTP.From
		}
		if update.SMTP.Hello != nil {
			settings.SMTP.Hello = *update.SMTP.Hello
		}
		if update.SMTP.RequireTLS != nil {
			settings.SMTP.RequireTLS = *update.SMTP.RequireTLS
		}
		if update.SMTP.Auth != nil {
			if update.SMTP.Auth.Username != nil {
				settings.SMTP.Auth.Username = *update.SMTP.Auth.Username
			}
			if update.SMTP.Auth.Password != nil {
				settings.SMTP.Auth.Password = *update.SMTP.Auth.Password
			}
			if update.SMTP.Auth.Secret != nil {
				settings.SMTP.Auth.Secret = *update.SMTP.Auth.Secret
			}
			if update.SMTP.Auth.Identity != nil {
				settings.SMTP.Auth.Identity = *update.SMTP.Auth.Identity
			}
			if update.SMTP.Auth.ClearPassword != nil && *update.SMTP.Auth.ClearPassword {
				settings.SMTP.Auth.Password = ""
			}
			if update.SMTP.Auth.ClearSecret != nil && *update.SMTP.Auth.ClearSecret {
				settings.SMTP.Auth.Secret = ""
			}
		}
		if update.SMTP.TLS != nil {
			if update.SMTP.TLS.Enabled != nil {
				settings.SMTP.TLS.Enabled = *update.SMTP.TLS.Enabled
			}
			if update.SMTP.TLS.InsecureSkipVerify != nil {
				settings.SMTP.TLS.InsecureSkipVerify = *update.SMTP.TLS.InsecureSkipVerify
			}
			if update.SMTP.TLS.CAFilePath != nil {
				settings.SMTP.TLS.CAFilePath = *update.SMTP.TLS.CAFilePath
			}
			if update.SMTP.TLS.CertFilePath != nil {
				settings.SMTP.TLS.CertFilePath = *update.SMTP.TLS.CertFilePath
			}
			if update.SMTP.TLS.KeyFilePath != nil {
				settings.SMTP.TLS.KeyFilePath = *update.SMTP.TLS.KeyFilePath
			}
		}
	}
}

func (settings AlertmanagerSettings) ToResponse() AlertmanagerSettingsResponse {
	return AlertmanagerSettingsResponse{
		Route: settings.Route,
		SMTP: AlertmanagerSMTPSettingsResponse{
			Address:    settings.SMTP.Address,
			From:       settings.SMTP.From,
			Hello:      settings.SMTP.Hello,
			RequireTLS: settings.SMTP.RequireTLS,
			Auth: AlertmanagerSMTPAuthSettingsResponse{
				Username:    settings.SMTP.Auth.Username,
				Identity:    settings.SMTP.Auth.Identity,
				PasswordSet: settings.SMTP.Auth.Password != "",
				SecretSet:   settings.SMTP.Auth.Secret != "",
			},
			TLS: settings.SMTP.TLS,
		},
	}
}

func (settings AlertmanagerSettings) Validate() error {
	if err := validateDuration("group_wait", settings.Route.GroupWait); err != nil {
		return err
	}
	if err := validateDuration("group_interval", settings.Route.GroupInterval); err != nil {
		return err
	}
	if err := validateDuration("repeat_interval", settings.Route.RepeatInterval); err != nil {
		return err
	}

	if err := validateSMTPAddress(settings.SMTP.Address); err != nil {
		return err
	}

	if err := validateTLSConfig(settings.SMTP.TLS); err != nil {
		return err
	}

	return nil
}

func (settings AlertmanagerSettings) ApplyToGlobal(globalConfig *GlobalConfig) error {
	host, port, err := splitHostPort(settings.SMTP.Address)
	if err != nil {
		return err
	}

	globalConfig.SMTPFrom = settings.SMTP.From
	globalConfig.SMTPHello = settings.SMTP.Hello
	globalConfig.SMTPSmarthost = amconfig.HostPort{Host: host, Port: port}
	globalConfig.SMTPAuthUsername = settings.SMTP.Auth.Username
	globalConfig.SMTPAuthPassword = amconfig.Secret(settings.SMTP.Auth.Password)
	globalConfig.SMTPAuthSecret = amconfig.Secret(settings.SMTP.Auth.Secret)
	globalConfig.SMTPAuthIdentity = settings.SMTP.Auth.Identity
	globalConfig.SMTPRequireTLS = settings.SMTP.RequireTLS

	if settings.SMTP.TLS.Enabled {
		globalConfig.SMTPTLSConfig = &commoncfg.TLSConfig{
			InsecureSkipVerify: settings.SMTP.TLS.InsecureSkipVerify,
			CAFile:             settings.SMTP.TLS.CAFilePath,
			CertFile:           settings.SMTP.TLS.CertFilePath,
			KeyFile:            settings.SMTP.TLS.KeyFilePath,
		}
	} else {
		globalConfig.SMTPTLSConfig = nil
	}

	return nil
}

func (settings AlertmanagerSettings) ApplyToRoute(routeConfig *RouteConfig) error {
	groupWait, err := time.ParseDuration(settings.Route.GroupWait)
	if err != nil {
		return errors.Newf(errors.TypeInvalidInput, ErrCodeAlertmanagerConfigInvalid, "invalid group_wait duration")
	}
	groupInterval, err := time.ParseDuration(settings.Route.GroupInterval)
	if err != nil {
		return errors.Newf(errors.TypeInvalidInput, ErrCodeAlertmanagerConfigInvalid, "invalid group_interval duration")
	}
	repeatInterval, err := time.ParseDuration(settings.Route.RepeatInterval)
	if err != nil {
		return errors.Newf(errors.TypeInvalidInput, ErrCodeAlertmanagerConfigInvalid, "invalid repeat_interval duration")
	}

	routeConfig.GroupWait = groupWait
	routeConfig.GroupInterval = groupInterval
	routeConfig.RepeatInterval = repeatInterval
	return nil
}

func validateDuration(fieldName string, value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.Newf(errors.TypeInvalidInput, ErrCodeAlertmanagerConfigInvalid, "%s is required", fieldName)
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return errors.Newf(errors.TypeInvalidInput, ErrCodeAlertmanagerConfigInvalid, "%s must be a valid duration", fieldName)
	}
	if duration <= 0 {
		return errors.Newf(errors.TypeInvalidInput, ErrCodeAlertmanagerConfigInvalid, "%s must be greater than 0", fieldName)
	}
	return nil
}

func validateSMTPAddress(address string) error {
	if strings.TrimSpace(address) == "" {
		return errors.Newf(errors.TypeInvalidInput, ErrCodeAlertmanagerConfigInvalid, "smtp address is required")
	}
	_, _, err := splitHostPort(address)
	if err != nil {
		return errors.Newf(errors.TypeInvalidInput, ErrCodeAlertmanagerConfigInvalid, "smtp address must be in host:port format")
	}
	return nil
}

func validateTLSConfig(config AlertmanagerSMTPTLSSettings) error {
	if !config.Enabled {
		if config.CAFilePath != "" || config.CertFilePath != "" || config.KeyFilePath != "" {
			return errors.Newf(errors.TypeInvalidInput, ErrCodeAlertmanagerConfigInvalid, "smtp tls is disabled but tls files are set")
		}
		return nil
	}

	if (config.CertFilePath == "") != (config.KeyFilePath == "") {
		return errors.Newf(errors.TypeInvalidInput, ErrCodeAlertmanagerConfigInvalid, "smtp tls cert and key must be provided together")
	}

	return nil
}

func splitHostPort(address string) (string, string, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(port) == "" {
		return "", "", fmt.Errorf("port is required")
	}
	return host, port, nil
}
