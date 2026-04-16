package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/modules/user"
	"github.com/SigNoz/signoz/pkg/tokenizer"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

// KO11yCallbackHandler handles the K-O11y SSO callback endpoint.
// It validates a K-O11y JWT from query parameters, provisions a SigNoz user
// via JIT provisioning, creates a SigNoz session token, and redirects to
// /login with the token parameters.
type KO11yCallbackHandler struct {
	validator *KO11yValidator
	ko11ySSO *user.KO11ySSO
	tokenizer tokenizer.Tokenizer
	logger    *slog.Logger
}

// NewKO11yCallbackHandler creates a new KO11yCallbackHandler.
func NewKO11yCallbackHandler(
	validator *KO11yValidator,
	ko11ySSO *user.KO11ySSO,
	tokenizer tokenizer.Tokenizer,
	logger *slog.Logger,
) *KO11yCallbackHandler {
	return &KO11yCallbackHandler{
		validator: validator,
		ko11ySSO: ko11ySSO,
		tokenizer: tokenizer,
		logger:    logger,
	}
}

// HandleCallback processes the K-O11y SSO callback request.
// Flow: extract token → validate JWT → provision user → create SigNoz token → redirect.
func (h *KO11yCallbackHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// 1. Extract token from query parameter
	tokenString := r.URL.Query().Get("token")
	if tokenString == "" {
		h.logger.WarnContext(ctx, "SSO callback: missing token parameter")
		http.Redirect(w, r, h.redirectURLFromErr(
			errors.New(errors.TypeInvalidInput, errors.CodeInvalidInput, "missing token parameter"),
		), http.StatusSeeOther)
		return
	}

	// 2. Validate K-O11y JWT (RS256 signature, expiration, claims)
	claims, err := h.validator.Validate(tokenString)
	if err != nil {
		h.logger.WarnContext(ctx, "SSO callback: invalid token", "error", err)
		http.Redirect(w, r, h.redirectURLFromErr(err), http.StatusSeeOther)
		return
	}

	// 2.5. Check tenant authorization (auto-lock / explicit list / allow-all)
	if err := h.validator.CheckTenant(ctx, claims); err != nil {
		h.logger.WarnContext(ctx, "SSO callback: tenant rejected",
			"tenant_id", claims.TenantID, "error", err)
		http.Redirect(w, r, h.redirectURLFromErr(err), http.StatusSeeOther)
		return
	}

	// 3. Extract email and role from claims
	email := claims.GetEmail()
	role := h.validator.MapRole(claims.Role)
	displayName := strings.SplitN(email, "@", 2)[0]

	// 4. JIT user provisioning (find or create)
	ssoUser, err := h.ko11ySSO.FindOrCreateUser(ctx, email, displayName, role)
	if err != nil {
		h.logger.ErrorContext(ctx, "SSO callback: user provisioning failed", "error", err, "email", email)
		http.Redirect(w, r, h.redirectURLFromErr(err), http.StatusSeeOther)
		return
	}

	// 5. Create SigNoz session token
	validEmail, _ := valuer.NewEmail(email)
	identity := authtypes.NewIdentity(ssoUser.ID, ssoUser.OrgID, validEmail, ssoUser.Role)
	token, err := h.tokenizer.CreateToken(ctx, identity, map[string]string{})
	if err != nil {
		h.logger.ErrorContext(ctx, "SSO callback: token creation failed", "error", err, "user_id", ssoUser.ID.StringValue())
		http.Redirect(w, r, h.redirectURLFromErr(err), http.StatusSeeOther)
		return
	}

	// 6. Build redirect URL with token parameters
	rotationInterval := h.tokenizer.Config().Rotation.Interval
	redirectURL := &url.URL{
		Path:     "/login",
		RawQuery: authtypes.NewURLValuesFromToken(token, rotationInterval).Encode(),
	}

	h.logger.InfoContext(ctx, "SSO callback: login successful",
		"user_id", ssoUser.ID.StringValue(),
		"email", email,
	)

	http.Redirect(w, r, redirectURL.String(), http.StatusSeeOther)
}

// redirectURLFromErr builds an error redirect URL following the existing
// SigNoz callback error pattern (/login?callbackauthnerr=true&...).
func (h *KO11yCallbackHandler) redirectURLFromErr(err error) string {
	values := errors.AsURLValues(err)
	values.Add("callbackauthnerr", "true")

	return (&url.URL{
		Path:     "/login",
		RawQuery: values.Encode(),
	}).String()
}
