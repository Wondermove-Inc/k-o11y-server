package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/modules/user"
	"github.com/SigNoz/signoz/pkg/tokenizer"
	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// callbackMockTokenizer extends mockTokenizer with configurable CreateToken for callback tests.
type callbackMockTokenizer struct {
	mockTokenizer
	createTokenFn func(ctx context.Context, identity *authtypes.Identity, meta map[string]string) (*authtypes.Token, error)
	rotationCfg   tokenizer.Config
}

func (m *callbackMockTokenizer) CreateToken(ctx context.Context, identity *authtypes.Identity, meta map[string]string) (*authtypes.Token, error) {
	return m.createTokenFn(ctx, identity, meta)
}

func (m *callbackMockTokenizer) Config() tokenizer.Config {
	return m.rotationCfg
}

var _ tokenizer.Tokenizer = (*callbackMockTokenizer)(nil)

// --- Test helpers ---

var (
	cbTestUserID = valuer.GenerateUUID()
	cbTestOrgID  = valuer.GenerateUUID()
)

// newTestCallbackHandler creates a KO11yCallbackHandler with controllable mocks.
func newTestCallbackHandler(
	userModFn func(context.Context, *types.User, ...user.CreateUserOption) (*types.User, error),
	tokenizerFn func(context.Context, *authtypes.Identity, map[string]string) (*authtypes.Token, error),
) *KO11yCallbackHandler {
	validator := NewKO11yValidator(true, testPublicKey, "ko11y", "EDITOR", nil, testLogger())

	org := testOrg()
	org.ID = cbTestOrgID
	orgGetter := &mockOrgGetterForAuthn{org: org}
	userMod := &mockUserModuleForAuthn{
		users: make(map[string]*types.User),
	}
	// Override GetOrCreateUser with custom function
	if userModFn != nil {
		userMod = &mockUserModuleForAuthn{users: make(map[string]*types.User)}
	}
	ko11ySSO := user.NewKO11ySSO(userMod, orgGetter, testLogger())

	// Use a wrapper that provides the custom userModFn behavior
	wrappedSSO := user.NewKO11ySSO(
		&callbackUserModule{fn: userModFn},
		orgGetter,
		testLogger(),
	)

	tok := &callbackMockTokenizer{
		createTokenFn: tokenizerFn,
		rotationCfg: tokenizer.Config{
			Rotation: tokenizer.RotationConfig{
				Interval: 30 * time.Minute,
			},
		},
	}
	_ = ko11ySSO // suppress unused

	return NewKO11yCallbackHandler(validator, wrappedSSO, tok, testLogger())
}

// callbackUserModule wraps a function to implement user.Module for callback tests.
type callbackUserModule struct {
	mockUserModuleForAuthn
	fn func(context.Context, *types.User, ...user.CreateUserOption) (*types.User, error)
}

func (m *callbackUserModule) GetOrCreateUser(ctx context.Context, u *types.User, opts ...user.CreateUserOption) (*types.User, error) {
	return m.fn(ctx, u, opts...)
}

// defaultCbUserModFn returns a provisioned user.
func defaultCbUserModFn() func(context.Context, *types.User, ...user.CreateUserOption) (*types.User, error) {
	return func(_ context.Context, u *types.User, _ ...user.CreateUserOption) (*types.User, error) {
		u.ID = cbTestUserID
		u.OrgID = cbTestOrgID
		return u, nil
	}
}

// defaultCbTokenizerFn returns a valid token.
func defaultCbTokenizerFn() func(context.Context, *authtypes.Identity, map[string]string) (*authtypes.Token, error) {
	return func(_ context.Context, _ *authtypes.Identity, _ map[string]string) (*authtypes.Token, error) {
		token, _ := authtypes.NewToken(map[string]string{}, cbTestUserID)
		return token, nil
	}
}

// --- Tests ---

func TestHandleCallback_ValidToken(t *testing.T) {
	handler := newTestCallbackHandler(defaultCbUserModFn(), defaultCbTokenizerFn())

	claims := validClaims()
	claims.Email = "test@company.com"
	jwtToken := makeKO11yToken(t, testPrivateKey, claims)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/ko11y/callback?token="+jwtToken, nil)
	rec := httptest.NewRecorder()

	handler.HandleCallback(rec, req)

	// Should redirect with 303 See Other
	assert.Equal(t, http.StatusSeeOther, rec.Code)

	location := rec.Header().Get("Location")
	require.NotEmpty(t, location)

	redirectURL, err := url.Parse(location)
	require.NoError(t, err)

	assert.Equal(t, "/login", redirectURL.Path)
	assert.NotEmpty(t, redirectURL.Query().Get("accessToken"))
	assert.NotEmpty(t, redirectURL.Query().Get("refreshToken"))
	assert.NotEmpty(t, redirectURL.Query().Get("expiresIn"))
	assert.Equal(t, "bearer", redirectURL.Query().Get("tokenType"))

	// Should NOT have error flag
	assert.Empty(t, redirectURL.Query().Get("callbackauthnerr"))
}

func TestHandleCallback_MissingToken(t *testing.T) {
	handler := newTestCallbackHandler(defaultCbUserModFn(), defaultCbTokenizerFn())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/ko11y/callback", nil)
	rec := httptest.NewRecorder()

	handler.HandleCallback(rec, req)

	assert.Equal(t, http.StatusSeeOther, rec.Code)

	location := rec.Header().Get("Location")
	redirectURL, err := url.Parse(location)
	require.NoError(t, err)

	assert.Equal(t, "/login", redirectURL.Path)
	assert.Equal(t, "true", redirectURL.Query().Get("callbackauthnerr"))
}

func TestHandleCallback_InvalidToken(t *testing.T) {
	handler := newTestCallbackHandler(defaultCbUserModFn(), defaultCbTokenizerFn())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/ko11y/callback?token=invalid-jwt", nil)
	rec := httptest.NewRecorder()

	handler.HandleCallback(rec, req)

	assert.Equal(t, http.StatusSeeOther, rec.Code)

	location := rec.Header().Get("Location")
	redirectURL, err := url.Parse(location)
	require.NoError(t, err)

	assert.Equal(t, "/login", redirectURL.Path)
	assert.Equal(t, "true", redirectURL.Query().Get("callbackauthnerr"))
}

func TestHandleCallback_ExpiredToken(t *testing.T) {
	handler := newTestCallbackHandler(defaultCbUserModFn(), defaultCbTokenizerFn())

	claims := validClaims()
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-1 * time.Hour))
	jwtToken := makeKO11yToken(t, testPrivateKey, claims)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/ko11y/callback?token="+jwtToken, nil)
	rec := httptest.NewRecorder()

	handler.HandleCallback(rec, req)

	assert.Equal(t, http.StatusSeeOther, rec.Code)

	location := rec.Header().Get("Location")
	redirectURL, err := url.Parse(location)
	require.NoError(t, err)

	assert.Equal(t, "/login", redirectURL.Path)
	assert.Equal(t, "true", redirectURL.Query().Get("callbackauthnerr"))
}

func TestHandleCallback_UserProvisioningError(t *testing.T) {
	failingUserMod := func(_ context.Context, _ *types.User, _ ...user.CreateUserOption) (*types.User, error) {
		return nil, fmt.Errorf("database connection error")
	}

	handler := newTestCallbackHandler(failingUserMod, defaultCbTokenizerFn())

	claims := validClaims()
	claims.Email = "test@company.com"
	jwtToken := makeKO11yToken(t, testPrivateKey, claims)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/ko11y/callback?token="+jwtToken, nil)
	rec := httptest.NewRecorder()

	handler.HandleCallback(rec, req)

	assert.Equal(t, http.StatusSeeOther, rec.Code)

	location := rec.Header().Get("Location")
	redirectURL, err := url.Parse(location)
	require.NoError(t, err)

	assert.Equal(t, "/login", redirectURL.Path)
	assert.Equal(t, "true", redirectURL.Query().Get("callbackauthnerr"))
}

func TestHandleCallback_TokenCreationError(t *testing.T) {
	failingTokenizer := func(_ context.Context, _ *authtypes.Identity, _ map[string]string) (*authtypes.Token, error) {
		return nil, fmt.Errorf("token store unavailable")
	}

	handler := newTestCallbackHandler(defaultCbUserModFn(), failingTokenizer)

	claims := validClaims()
	claims.Email = "test@company.com"
	jwtToken := makeKO11yToken(t, testPrivateKey, claims)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/ko11y/callback?token="+jwtToken, nil)
	rec := httptest.NewRecorder()

	handler.HandleCallback(rec, req)

	assert.Equal(t, http.StatusSeeOther, rec.Code)

	location := rec.Header().Get("Location")
	redirectURL, err := url.Parse(location)
	require.NoError(t, err)

	assert.Equal(t, "/login", redirectURL.Path)
	assert.Equal(t, "true", redirectURL.Query().Get("callbackauthnerr"))
}

func TestHandleCallback_FallbackEmail(t *testing.T) {
	// When JWT has no email, should use UserID@ko11y.local
	var capturedUser *types.User
	capturingUserMod := func(_ context.Context, u *types.User, _ ...user.CreateUserOption) (*types.User, error) {
		capturedUser = u
		u.ID = cbTestUserID
		u.OrgID = cbTestOrgID
		return u, nil
	}

	handler := newTestCallbackHandler(capturingUserMod, defaultCbTokenizerFn())

	claims := validClaims()
	claims.Email = "" // no email, should fallback
	jwtToken := makeKO11yToken(t, testPrivateKey, claims)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/ko11y/callback?token="+jwtToken, nil)
	rec := httptest.NewRecorder()

	handler.HandleCallback(rec, req)

	assert.Equal(t, http.StatusSeeOther, rec.Code)
	require.NotNil(t, capturedUser)
	assert.Equal(t, "usr_abc123@ko11y.local", capturedUser.Email.StringValue())
}
