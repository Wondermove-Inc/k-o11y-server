package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/modules/user"
	"github.com/SigNoz/signoz/pkg/sharder"
	"github.com/SigNoz/signoz/pkg/tokenizer"
	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	"github.com/SigNoz/signoz/pkg/valuer"
	"golang.org/x/sync/singleflight"
)

const (
	authCrossOrgMessage string = "::AUTH-CROSS-ORG::"
)

type AuthN struct {
	tokenizer       tokenizer.Tokenizer
	headers         []string
	sharder         sharder.Sharder
	logger          *slog.Logger
	sfGroup         *singleflight.Group
	ko11yValidator *KO11yValidator
	ko11ySSO       *user.KO11ySSO
}

func NewAuthN(headers []string, sharder sharder.Sharder, tokenizer tokenizer.Tokenizer, logger *slog.Logger) *AuthN {
	return &AuthN{
		headers:   headers,
		sharder:   sharder,
		tokenizer: tokenizer,
		logger:    logger,
		sfGroup:   &singleflight.Group{},
	}
}

// SetKO11ySSO injects the K-O11y SSO components into the AuthN middleware.
// When set, K-O11y JWT tokens are validated as a fallback when SigNoz JWT fails.
func (a *AuthN) SetKO11ySSO(validator *KO11yValidator, sso *user.KO11ySSO) {
	a.ko11yValidator = validator
	a.ko11ySSO = sso
}

func (a *AuthN) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var values []string
		for _, header := range a.headers {
			values = append(values, r.Header.Get(header))
		}

		ctx, err := a.contextFromRequest(r.Context(), values...)
		if err != nil {
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
			return
		}

		r = r.WithContext(ctx)

		claims, err := authtypes.ClaimsFromContext(ctx)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		if err := a.sharder.IsMyOwnedKey(r.Context(), types.NewOrganizationKey(valuer.MustNewUUID(claims.OrgID))); err != nil {
			a.logger.ErrorContext(r.Context(), authCrossOrgMessage, "claims", claims, "error", err)
			next.ServeHTTP(w, r)
			return
		}

		comment := ctxtypes.CommentFromContext(ctx)
		comment.Set("user_id", claims.UserID)
		comment.Set("org_id", claims.OrgID)

		if isKO11ySSO(ctx) {
			ctx = ctxtypes.SetAuthType(ctx, ctxtypes.AuthTypeTokenizer)
			comment.Set("auth_type", "sso")
		} else {
			ctx = ctxtypes.SetAuthType(ctx, ctxtypes.AuthTypeTokenizer)
			comment.Set("auth_type", ctxtypes.AuthTypeTokenizer.StringValue())
			comment.Set("tokenizer_provider", a.tokenizer.Config().Provider)
		}

		r = r.WithContext(ctxtypes.NewContextWithComment(ctx, comment))

		next.ServeHTTP(w, r)

		// Skip token tracking for K-O11y SSO (tokens not in SigNoz store)
		if isKO11ySSO(r.Context()) {
			return
		}

		accessToken, err := authtypes.AccessTokenFromContext(r.Context())
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		lastObservedAtCtx := context.WithoutCancel(r.Context())
		_, _, _ = a.sfGroup.Do(accessToken, func() (any, error) {
			if err := a.tokenizer.SetLastObservedAt(lastObservedAtCtx, accessToken, time.Now()); err != nil {
				a.logger.ErrorContext(lastObservedAtCtx, "failed to set last observed at", "error", err)
				return false, err
			}

			return true, nil
		})
	})
}

func (a *AuthN) contextFromRequest(ctx context.Context, values ...string) (context.Context, error) {
	ctx, err := a.contextFromAccessToken(ctx, values...)
	if err != nil {
		return ctx, err
	}

	accessToken, err := authtypes.AccessTokenFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	// Try SigNoz JWT first (existing flow)
	authenticatedUser, err := a.tokenizer.GetIdentity(ctx, accessToken)
	if err == nil {
		return authtypes.NewContextWithClaims(ctx, authenticatedUser.ToClaims()), nil
	}

	// Fallback: try K-O11y JWT if SSO is enabled
	if a.ko11yValidator != nil && a.ko11yValidator.IsEnabled() {
		ko11yCtx, ko11yErr := a.contextFromKO11yToken(ctx, accessToken)
		if ko11yErr == nil {
			return ko11yCtx, nil
		}
		a.logger.WarnContext(ctx, "SSO fallback failed", "error", ko11yErr)
	}

	// Return the original SigNoz error
	return ctx, err
}

// contextFromKO11yToken validates a K-O11y JWT and provisions a user.
func (a *AuthN) contextFromKO11yToken(ctx context.Context, tokenString string) (context.Context, error) {
	ko11yClaims, err := a.ko11yValidator.Validate(tokenString)
	if err != nil {
		return ctx, err
	}

	// Check tenant authorization (auto-lock / explicit list / allow-all)
	if err := a.ko11yValidator.CheckTenant(ctx, ko11yClaims); err != nil {
		return ctx, err
	}

	role := a.ko11yValidator.MapRole(ko11yClaims.Role)
	email := ko11yClaims.GetEmail()
	displayName := email
	if ko11yClaims.UserID != "" {
		displayName = ko11yClaims.UserID
	}

	ssoUser, err := a.ko11ySSO.FindOrCreateUser(ctx, email, displayName, role)
	if err != nil {
		return ctx, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "SSO user provisioning failed: %s", err.Error())
	}

	claims := authtypes.Claims{
		UserID: ssoUser.ID.StringValue(),
		Email:  ssoUser.Email.StringValue(),
		Role:   ssoUser.Role,
		OrgID:  ssoUser.OrgID.StringValue(),
	}

	// Mark context as K-O11y SSO authenticated
	ctx = context.WithValue(ctx, ko11ySSOKey{}, true)

	return authtypes.NewContextWithClaims(ctx, claims), nil
}

// ko11ySSOKey is a context key for marking K-O11y SSO authenticated requests.
type ko11ySSOKey struct{}

// isKO11ySSO checks if the request was authenticated via K-O11y SSO.
func isKO11ySSO(ctx context.Context) bool {
	v, ok := ctx.Value(ko11ySSOKey{}).(bool)
	return ok && v
}

func (a *AuthN) contextFromAccessToken(ctx context.Context, values ...string) (context.Context, error) {
	var value string
	for _, v := range values {
		if v != "" {
			value = v
			break
		}
	}

	if value == "" {
		return ctx, errors.New(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing authorization header")
	}

	// parse from
	bearerToken, ok := parseBearerAuth(value)
	if !ok {
		// this will take care that if the value is not of type bearer token, directly use it
		bearerToken = value
	}

	return authtypes.NewContextWithAccessToken(ctx, bearerToken), nil
}

func parseBearerAuth(auth string) (string, bool) {
	const prefix = "Bearer "
	// Case insensitive prefix match
	if len(auth) < len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return "", false
	}

	return auth[len(prefix):], true
}
