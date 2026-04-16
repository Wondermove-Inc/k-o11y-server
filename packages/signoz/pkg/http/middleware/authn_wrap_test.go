package middleware

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/modules/user"
	"github.com/SigNoz/signoz/pkg/tokenizer"
	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// trackingMockTokenizer is a mockTokenizer that calls a hook on SetLastObservedAt,
// allowing tests to verify whether token tracking occurred.
type trackingMockTokenizer struct {
	mockTokenizer
	onSetLastObservedAt func()
}

// SetLastObservedAt invokes the tracking hook and returns nil.
func (m *trackingMockTokenizer) SetLastObservedAt(_ context.Context, _ string, _ time.Time) error {
	if m.onSetLastObservedAt != nil {
		m.onSetLastObservedAt()
	}
	return nil
}

// GetIdentity always returns an error, forcing the K-O11y SSO fallback.
func (m *trackingMockTokenizer) GetIdentity(_ context.Context, _ string) (*authtypes.Identity, error) {
	return nil, assert.AnError
}

// Config returns a mock tokenizer config.
func (m *trackingMockTokenizer) Config() tokenizer.Config {
	return tokenizer.Config{Provider: "tracking-mock"}
}

// failLastObservedAtTokenizer succeeds on GetIdentity but fails on SetLastObservedAt,
// allowing tests to exercise the SetLastObservedAt error path.
type failLastObservedAtTokenizer struct {
	mockTokenizer
	identity *authtypes.Identity
}

// GetIdentity returns the configured identity without error.
func (m *failLastObservedAtTokenizer) GetIdentity(_ context.Context, _ string) (*authtypes.Identity, error) {
	return m.identity, nil
}

// SetLastObservedAt always fails to simulate a storage error.
func (m *failLastObservedAtTokenizer) SetLastObservedAt(_ context.Context, _ string, _ time.Time) error {
	return assert.AnError
}

// Config returns a mock tokenizer config.
func (m *failLastObservedAtTokenizer) Config() tokenizer.Config {
	return tokenizer.Config{Provider: "fail-last-observed"}
}

// TestWrap_KO11ySSO_FallbackSuccess tests the full middleware chain when
// SigNoz JWT fails and K-O11y JWT succeeds as a fallback.
func TestWrap_KO11ySSO_FallbackSuccess(t *testing.T) {
	org := testOrg()
	orgGetter := &mockOrgGetterForAuthn{org: org}
	userMod := newMockUserModuleForAuthn()
	ko11ySSO := user.NewKO11ySSO(userMod, orgGetter, testLogger())
	validator := NewKO11yValidator(true, testPublicKey, "", "EDITOR", nil, testLogger())

	// SigNoz tokenizer always fails - forces K-O11y SSO fallback
	mockTok := &mockTokenizer{err: assert.AnError}
	authN := NewAuthN([]string{"Authorization"}, &mockSharder{err: nil}, mockTok, testLogger())
	authN.SetKO11ySSO(validator, ko11ySSO)

	tokenStr := makeKO11yTokenForAuthn(t, testPrivateKey, "usr_wrap_test")

	var capturedCtx context.Context
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCtx = r.Context()
		w.WriteHeader(http.StatusOK)
	})

	handler := authN.Wrap(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Request should pass through with K-O11y SSO claims
	require.NotNil(t, capturedCtx)
	claims, err := authtypes.ClaimsFromContext(capturedCtx)
	require.NoError(t, err)
	assert.Equal(t, "usr_wrap_test@ko11y.local", claims.Email)

	// Auth type should be set in context
	authType, ok := ctxtypes.AuthTypeFromContext(capturedCtx)
	require.True(t, ok)
	assert.Equal(t, ctxtypes.AuthTypeTokenizer, authType)
}

// TestWrap_MissingAuthHeader tests that requests without auth header are passed through.
func TestWrap_MissingAuthHeader(t *testing.T) {
	mockTok := &mockTokenizer{err: assert.AnError}
	authN := NewAuthN([]string{"Authorization"}, &mockSharder{}, mockTok, testLogger())

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := authN.Wrap(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.True(t, called)
}

// TestWrap_NoKO11ySSO tests middleware when K-O11y SSO is not configured.
func TestWrap_NoKO11ySSO(t *testing.T) {
	mockTok := &mockTokenizer{err: assert.AnError}
	authN := NewAuthN([]string{"Authorization"}, &mockSharder{}, mockTok, testLogger())
	// No SetKO11ySSO called

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := authN.Wrap(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer sometoken")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.True(t, called)
}

// TestWrap_SharderError tests that a sharder ownership error passes through to the next handler.
func TestWrap_SharderError(t *testing.T) {
	org := testOrg()
	orgGetter := &mockOrgGetterForAuthn{org: org}
	userMod := newMockUserModuleForAuthn()
	ko11ySSO := user.NewKO11ySSO(userMod, orgGetter, testLogger())
	validator := NewKO11yValidator(true, testPublicKey, "", "EDITOR", nil, testLogger())

	// SigNoz tokenizer always fails - forces K-O11y fallback
	mockTok := &mockTokenizer{err: assert.AnError}
	// Sharder returns an error for any key
	sharder := &mockSharder{err: assert.AnError}
	authN := NewAuthN([]string{"Authorization"}, sharder, mockTok, testLogger())
	authN.SetKO11ySSO(validator, ko11ySSO)

	tokenStr := makeKO11yTokenForAuthn(t, testPrivateKey, "usr_sharder_test")

	callCount := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	})

	handler := authN.Wrap(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// next is still called even when sharder returns error
	assert.Equal(t, 1, callCount)
}

// TestWrap_SigNozTokenSuccess tests the standard (non-K-O11y SSO) path through Wrap
// when the SigNoz tokenizer successfully validates the token.
func TestWrap_SigNozTokenSuccess(t *testing.T) {
	userID := valuer.GenerateUUID()
	orgID := valuer.GenerateUUID()
	email, err := valuer.NewEmail("test@example.com")
	require.NoError(t, err)

	identity := authtypes.NewIdentity(userID, orgID, email, types.RoleEditor)

	// SigNoz tokenizer succeeds
	mockTok := &mockTokenizer{identity: identity}
	authN := NewAuthN([]string{"Authorization"}, &mockSharder{}, mockTok, testLogger())
	// No K-O11y SSO configured

	var capturedCtx context.Context
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCtx = r.Context()
		w.WriteHeader(http.StatusOK)
	})

	handler := authN.Wrap(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer signoz-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.NotNil(t, capturedCtx)

	// Claims should be set from SigNoz identity
	claims, claimsErr := authtypes.ClaimsFromContext(capturedCtx)
	require.NoError(t, claimsErr)
	assert.Equal(t, userID.String(), claims.UserID)
	assert.Equal(t, "test@example.com", claims.Email)

	// Should NOT be marked as K-O11y SSO
	assert.False(t, isKO11ySSO(capturedCtx))

	// Auth type should be tokenizer
	authType, ok := ctxtypes.AuthTypeFromContext(capturedCtx)
	require.True(t, ok)
	assert.Equal(t, ctxtypes.AuthTypeTokenizer, authType)
}

// TestWrap_KO11ySSO_SkipsTokenTracking verifies that K-O11y SSO requests
// do not trigger SetLastObservedAt on the tokenizer.
func TestWrap_KO11ySSO_SkipsTokenTracking(t *testing.T) {
	org := testOrg()
	orgGetter := &mockOrgGetterForAuthn{org: org}
	userMod := newMockUserModuleForAuthn()
	ko11ySSO := user.NewKO11ySSO(userMod, orgGetter, testLogger())
	validator := NewKO11yValidator(true, testPublicKey, "", "EDITOR", nil, testLogger())

	setLastObservedAtCalled := false
	trackingTokenizer := &trackingMockTokenizer{
		onSetLastObservedAt: func() { setLastObservedAtCalled = true },
	}

	authN := NewAuthN([]string{"Authorization"}, &mockSharder{}, trackingTokenizer, testLogger())
	authN.SetKO11ySSO(validator, ko11ySSO)

	tokenStr := makeKO11yTokenForAuthn(t, testPrivateKey, "usr_track_test")

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := authN.Wrap(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// SetLastObservedAt should NOT be called for K-O11y SSO tokens
	assert.False(t, setLastObservedAtCalled)
}

// TestWrap_KO11ySSO_BothFail tests that when both SigNoz and K-O11y JWT fail,
// the request passes through without authentication.
func TestWrap_KO11ySSO_BothFail(t *testing.T) {
	org := testOrg()
	orgGetter := &mockOrgGetterForAuthn{org: org}
	userMod := newMockUserModuleForAuthn()
	ko11ySSO := user.NewKO11ySSO(userMod, orgGetter, testLogger())

	// Validator uses a different key pair - will fail for tokens signed with testPrivateKey
	wrongKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	validator := NewKO11yValidator(true, &wrongKey.PublicKey, "", "EDITOR", nil, testLogger())

	mockTok := &mockTokenizer{err: assert.AnError}
	authN := NewAuthN([]string{"Authorization"}, &mockSharder{}, mockTok, testLogger())
	authN.SetKO11ySSO(validator, ko11ySSO)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := authN.Wrap(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Token signed with testPrivateKey but validator uses wrongKey.PublicKey
	tokenStr := makeKO11yTokenForAuthn(t, testPrivateKey, "usr_both_fail")
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// next is still called even when both authentication methods fail
	assert.True(t, called)
}

// TestWrap_SigNozToken_SetLastObservedAtError tests that SetLastObservedAt errors are handled gracefully.
func TestWrap_SigNozToken_SetLastObservedAtError(t *testing.T) {
	userID := valuer.GenerateUUID()
	orgID := valuer.GenerateUUID()
	email, err := valuer.NewEmail("test2@example.com")
	require.NoError(t, err)

	identity := authtypes.NewIdentity(userID, orgID, email, types.RoleEditor)

	// Tokenizer succeeds on GetIdentity but fails on SetLastObservedAt
	failingTokenizer := &failLastObservedAtTokenizer{
		identity: identity,
	}
	authN := NewAuthN([]string{"Authorization"}, &mockSharder{}, failingTokenizer, testLogger())

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := authN.Wrap(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer signoz-token-fail")
	rr := httptest.NewRecorder()

	// Should not panic; SetLastObservedAt error is logged and ignored
	handler.ServeHTTP(rr, req)
	assert.True(t, called)
}
