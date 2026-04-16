package middleware

import (
	"context"
	"crypto/rsa"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/golang-jwt/jwt/v5"
)

// KO11yClaims represents the JWT claims from a K-O11y platform token.
type KO11yClaims struct {
	jwt.RegisteredClaims
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Role     string `json:"role"`
	Email    string `json:"email,omitempty"`
}

// GetEmail returns the email from claims, falling back to a synthetic email if not provided.
func (c *KO11yClaims) GetEmail() string {
	if c.Email != "" {
		return c.Email
	}
	return c.UserID + "@ko11y.local"
}

// TenantStore abstracts ClickHouse access for SSO tenant auto-lock.
type TenantStore interface {
	GetSSOAllowedTenants(ctx context.Context) ([]SSOTenant, error)
	InsertSSOAllowedTenant(ctx context.Context, tenantID, lockedBy string) error
}

// SSOTenant represents an allowed tenant from ko11y.sso_config.
type SSOTenant struct {
	TenantID string
}

// tenantPolicy describes how tenant validation is performed.
type tenantPolicy int

const (
	// tenantPolicyAutoLock: env empty → first login locks, then table-based check
	tenantPolicyAutoLock tenantPolicy = iota
	// tenantPolicyAllowAll: env = "*" → skip all tenant checks
	tenantPolicyAllowAll
	// tenantPolicyExplicit: env = "t1,t2" → only listed tenants allowed
	tenantPolicyExplicit
)

// KO11yValidator validates K-O11y JWT tokens (RS256) and maps claims
// to SigNoz auth types. It is used as a fallback in the AuthN
// middleware when the primary SigNoz tokenizer fails.
type KO11yValidator struct {
	publicKey      *rsa.PublicKey
	issuer         string
	defaultRole    types.Role
	enabled        bool
	policy         tenantPolicy
	allowedTenants map[string]struct{} // for tenantPolicyExplicit
	tenantStore    TenantStore         // for tenantPolicyAutoLock (nil until set)
	logger         *slog.Logger

	// Cache for auto-lock tenants (avoids DB query on every request)
	cacheMu      sync.RWMutex
	cachedSet    map[string]struct{}
	cacheExpires time.Time
}

const tenantCacheTTL = 60 * time.Second

// NewKO11yValidator creates a KO11yValidator from the given parameters.
// Returns nil if enabled is false or publicKey is nil.
func NewKO11yValidator(enabled bool, publicKey *rsa.PublicKey, issuer string, defaultRole string, allowedTenants []string, logger *slog.Logger) *KO11yValidator {
	if !enabled || publicKey == nil {
		return nil
	}

	role, err := types.NewRole(defaultRole)
	if err != nil {
		role = types.RoleEditor
	}

	// Determine tenant policy from allowedTenants list
	policy := tenantPolicyAutoLock
	tenantSet := make(map[string]struct{})

	if len(allowedTenants) == 1 && allowedTenants[0] == "*" {
		policy = tenantPolicyAllowAll
		logger.Info("K-O11y SSO tenant policy: allow all (internal mode)")
	} else if len(allowedTenants) > 0 {
		policy = tenantPolicyExplicit
		for _, t := range allowedTenants {
			tenantSet[t] = struct{}{}
		}
		logger.Info("K-O11y SSO tenant policy: explicit list", "tenants", allowedTenants)
	} else {
		logger.Info("K-O11y SSO tenant policy: auto-lock (first login locks tenant)")
	}

	return &KO11yValidator{
		publicKey:      publicKey,
		issuer:         issuer,
		defaultRole:    role,
		enabled:        true,
		policy:         policy,
		allowedTenants: tenantSet,
		logger:         logger,
	}
}

// SetTenantStore injects the DB store for auto-lock tenant management.
func (v *KO11yValidator) SetTenantStore(store TenantStore) {
	if v != nil {
		v.tenantStore = store
	}
}

// IsEnabled reports whether the K-O11y SSO validator is active.
func (v *KO11yValidator) IsEnabled() bool {
	return v != nil && v.enabled
}

// Validate parses and validates a K-O11y JWT token string.
// It verifies the RS256 signature, expiration, and required claims.
// Tenant validation is NOT performed here — use CheckTenant separately.
func (v *KO11yValidator) Validate(tokenString string) (*KO11yClaims, error) {
	if !v.IsEnabled() {
		return nil, errors.New(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "SSO is not enabled")
	}

	var claims KO11yClaims
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return v.publicKey, nil
	})
	if err != nil {
		return nil, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "invalid token: %s", err.Error())
	}

	if !token.Valid {
		return nil, errors.New(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "invalid token")
	}

	// Verify issuer if configured
	if v.issuer != "" {
		issuer, _ := claims.GetIssuer()
		if issuer != v.issuer {
			return nil, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "invalid issuer: expected %s, got %s", v.issuer, issuer)
		}
	}

	// Validate required claims
	if claims.UserID == "" {
		return nil, errors.New(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing user_id in token")
	}

	return &claims, nil
}

// CheckTenant validates the tenant_id against the configured policy.
// For auto-lock: if no tenants in DB, inserts the first one and allows access.
func (v *KO11yValidator) CheckTenant(ctx context.Context, claims *KO11yClaims) error {
	switch v.policy {
	case tenantPolicyAllowAll:
		return nil

	case tenantPolicyExplicit:
		if claims.TenantID == "" {
			return errors.New(errors.TypeForbidden, errors.CodeForbidden, "missing tenant_id in token")
		}
		if _, ok := v.allowedTenants[claims.TenantID]; !ok {
			return errors.Newf(errors.TypeForbidden, errors.CodeForbidden, "tenant %s is not allowed", claims.TenantID)
		}
		return nil

	case tenantPolicyAutoLock:
		return v.checkTenantAutoLock(ctx, claims)

	default:
		return errors.New(errors.TypeInternal, errors.CodeInternal, "unknown tenant policy")
	}
}

// checkTenantAutoLock implements the auto-lock flow:
// 1. Query DB for allowed tenants (with cache)
// 2. If empty → INSERT current tenant (first-login lock)
// 3. If non-empty → check if current tenant is in the list
func (v *KO11yValidator) checkTenantAutoLock(ctx context.Context, claims *KO11yClaims) error {
	if claims.TenantID == "" {
		return errors.New(errors.TypeForbidden, errors.CodeForbidden, "missing tenant_id in token")
	}

	if v.tenantStore == nil {
		// No store configured — allow (graceful degradation)
		v.logger.WarnContext(ctx, "SSO: tenant store not configured, skipping auto-lock check")
		return nil
	}

	// Check cache first
	if v.isAllowedCached(claims.TenantID) {
		return nil
	}

	// Query DB
	tenants, err := v.tenantStore.GetSSOAllowedTenants(ctx)
	if err != nil {
		v.logger.ErrorContext(ctx, "SSO: failed to query tenant store", "error", err)
		// Allow on DB error (graceful degradation — JWT signature already verified)
		return nil
	}

	// Empty table → first-login auto-lock
	if len(tenants) == 0 {
		v.logger.InfoContext(ctx, "SSO: first login, auto-locking tenant",
			"tenant_id", claims.TenantID, "email", claims.GetEmail())
		if err := v.tenantStore.InsertSSOAllowedTenant(ctx, claims.TenantID, claims.GetEmail()); err != nil {
			v.logger.ErrorContext(ctx, "SSO: failed to insert auto-lock tenant", "error", err)
			// Allow anyway — the lock will happen on next request
		}
		v.updateCache([]SSOTenant{{TenantID: claims.TenantID}})
		return nil
	}

	// Build set and update cache
	v.updateCache(tenants)

	// Check if tenant is allowed
	if v.isAllowedCached(claims.TenantID) {
		return nil
	}

	return errors.Newf(errors.TypeForbidden, errors.CodeForbidden,
		"tenant %s is not allowed to access this environment", claims.TenantID)
}

// isAllowedCached checks the in-memory cache for tenant authorization.
func (v *KO11yValidator) isAllowedCached(tenantID string) bool {
	v.cacheMu.RLock()
	defer v.cacheMu.RUnlock()

	if v.cachedSet == nil || time.Now().After(v.cacheExpires) {
		return false
	}
	_, ok := v.cachedSet[tenantID]
	return ok
}

// updateCache refreshes the in-memory tenant cache.
func (v *KO11yValidator) updateCache(tenants []SSOTenant) {
	v.cacheMu.Lock()
	defer v.cacheMu.Unlock()

	v.cachedSet = make(map[string]struct{}, len(tenants))
	for _, t := range tenants {
		v.cachedSet[t.TenantID] = struct{}{}
	}
	v.cacheExpires = time.Now().Add(tenantCacheTTL)
}

// MapRole converts a role string to a SigNoz Role.
// "admin" and "tenant_admin" map to ADMIN, everything else maps to the configured default role.
func (v *KO11yValidator) MapRole(role string) types.Role {
	switch role {
	case "admin", "tenant_admin":
		return types.RoleAdmin
	default:
		return v.defaultRole
	}
}

// ToIdentity converts validated K-O11y claims to a SigNoz Identity.
func (v *KO11yValidator) ToIdentity(claims *KO11yClaims, userID valuer.UUID, orgID valuer.UUID) *authtypes.Identity {
	email, _ := valuer.NewEmail(claims.GetEmail())
	role := v.MapRole(claims.Role)
	return authtypes.NewIdentity(userID, orgID, email, role)
}
