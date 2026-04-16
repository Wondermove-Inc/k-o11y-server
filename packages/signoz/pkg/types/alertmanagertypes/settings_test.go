package alertmanagertypes

import (
	"testing"
)

func TestAlertmanagerSettingsValidate(t *testing.T) {
	valid := AlertmanagerSettings{
		Route: AlertmanagerRouteSettings{
			GroupWait:      "30s",
			GroupInterval:  "5m",
			RepeatInterval: "1h",
		},
		SMTP: AlertmanagerSMTPSettings{
			Address:    "smtp.example.com:587",
			From:       "alerts@example.com",
			Hello:      "localhost",
			RequireTLS: true,
			Auth: AlertmanagerSMTPAuthSettings{
				Username: "user",
				Password: "secret",
			},
			TLS: AlertmanagerSMTPTLSSettings{
				Enabled:            true,
				InsecureSkipVerify: false,
				CAFilePath:         "/etc/ssl/certs/ca.pem",
				CertFilePath:       "/etc/ssl/certs/client.pem",
				KeyFilePath:        "/etc/ssl/private/client.key",
			},
		},
	}

	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid settings, got error: %v", err)
	}

	invalidDuration := valid
	invalidDuration.Route.GroupWait = "not-a-duration"
	if err := invalidDuration.Validate(); err == nil {
		t.Fatalf("expected duration validation error")
	}

	invalidAddress := valid
	invalidAddress.SMTP.Address = "smtp.example.com"
	if err := invalidAddress.Validate(); err == nil {
		t.Fatalf("expected smtp address validation error")
	}

	invalidTLS := valid
	invalidTLS.SMTP.TLS.CertFilePath = ""
	if err := invalidTLS.Validate(); err == nil {
		t.Fatalf("expected tls cert/key validation error")
	}
}
