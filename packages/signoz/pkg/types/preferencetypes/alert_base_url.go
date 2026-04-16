package preferencetypes

import (
	"net/url"
	"strings"

	"github.com/SigNoz/signoz/pkg/errors"
)

const DefaultAlertBaseURL = "http://localhost:8080"

func NormalizeAlertBaseURL(input any) (string, error) {
	rawValue, ok := input.(string)
	if !ok {
		return "", errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "alert base URL must be a string")
	}

	trimmed := strings.TrimSpace(rawValue)
	if trimmed == "" {
		return "", errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "alert base URL is required")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", errors.WrapInvalidInputf(err, errors.CodeInvalidInput, "alert base URL is invalid")
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "alert base URL must use http or https")
	}
	if parsed.Host == "" {
		return "", errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "alert base URL must include a host")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "alert base URL must not include query or fragment")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "alert base URL must not include a path")
	}

	parsed.Path = ""
	return strings.TrimSuffix(parsed.String(), "/"), nil
}
