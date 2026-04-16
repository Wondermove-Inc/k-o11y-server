package middleware

import (
	"context"
	"crypto/rsa"
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

// mockTokenizer implements tokenizer.Tokenizer for unit testing.
type mockTokenizer struct {
	identity *authtypes.Identity
	err      error
}

func (m *mockTokenizer) GetIdentity(_ context.Context, _ string) (*authtypes.Identity, error) {
	return m.identity, m.err
}
func (m *mockTokenizer) CreateToken(_ context.Context, _ *authtypes.Identity, _ map[string]string) (*authtypes.Token, error) {
	panic("not implemented")
}
func (m *mockTokenizer) RotateToken(_ context.Context, _, _ string) (*authtypes.Token, error) {
	panic("not implemented")
}
func (m *mockTokenizer) DeleteToken(_ context.Context, _ string) error {
	panic("not implemented")
}
func (m *mockTokenizer) DeleteTokensByUserID(_ context.Context, _ valuer.UUID) error {
	panic("not implemented")
}
func (m *mockTokenizer) DeleteIdentity(_ context.Context, _ valuer.UUID) error {
	panic("not implemented")
}
func (m *mockTokenizer) SetLastObservedAt(_ context.Context, _ string, _ time.Time) error {
	return nil
}
func (m *mockTokenizer) Config() tokenizer.Config {
	return tokenizer.Config{Provider: "mock"}
}
func (m *mockTokenizer) Start(_ context.Context) error { return nil }
func (m *mockTokenizer) Stop(_ context.Context) error  { return nil }
func (m *mockTokenizer) Collect(_ context.Context, _ valuer.UUID) (map[string]any, error) {
	return nil, nil
}
func (m *mockTokenizer) ListMaxLastObservedAtByOrgID(_ context.Context, _ valuer.UUID) (map[valuer.UUID]time.Time, error) {
	return nil, nil
}

// mockSharder implements sharder.Sharder for unit testing.
type mockSharder struct {
	err error
}

func (m *mockSharder) GetMyOwnedKeyRange(_ context.Context) (uint32, uint32, error) {
	return 0, ^uint32(0), nil
}

func (m *mockSharder) IsMyOwnedKey(_ context.Context, _ uint32) error {
	return m.err
}

// mockOrgGetterForAuthn implements organization.Getter for authn testing.
type mockOrgGetterForAuthn struct {
	org *types.Organization
	err error
}

func (m *mockOrgGetterForAuthn) Get(_ context.Context, _ valuer.UUID) (*types.Organization, error) {
	return m.org, m.err
}

func (m *mockOrgGetterForAuthn) ListByOwnedKeyRange(_ context.Context) ([]*types.Organization, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.org == nil {
		return []*types.Organization{}, nil
	}
	return []*types.Organization{m.org}, nil
}

// newMockUserModuleForAuthn returns a minimal Module mock for authn tests.
func newMockUserModuleForAuthn() *mockUserModuleForAuthn {
	return &mockUserModuleForAuthn{users: make(map[string]*types.User)}
}

// mockUserModuleForAuthn is a minimal user.Module mock for authn testing.
type mockUserModuleForAuthn struct {
	users map[string]*types.User
	err   error
}

func (m *mockUserModuleForAuthn) GetOrCreateUser(_ context.Context, u *types.User, _ ...user.CreateUserOption) (*types.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	key := u.Email.StringValue() + "|" + u.OrgID.StringValue()
	if existing, ok := m.users[key]; ok {
		return existing, nil
	}
	m.users[key] = u
	return u, nil
}

func (m *mockUserModuleForAuthn) CreateFirstUser(_ context.Context, _ *types.Organization, _ string, _ valuer.Email, _ string) (*types.User, error) {
	panic("not implemented")
}
func (m *mockUserModuleForAuthn) CreateUser(_ context.Context, _ *types.User, _ ...user.CreateUserOption) error {
	panic("not implemented")
}
func (m *mockUserModuleForAuthn) GetOrCreateResetPasswordToken(_ context.Context, _ valuer.UUID) (*types.ResetPasswordToken, error) {
	panic("not implemented")
}
func (m *mockUserModuleForAuthn) UpdatePasswordByResetPasswordToken(_ context.Context, _, _ string) error {
	panic("not implemented")
}
func (m *mockUserModuleForAuthn) UpdatePassword(_ context.Context, _ valuer.UUID, _, _ string) error {
	panic("not implemented")
}
func (m *mockUserModuleForAuthn) UpdateUser(_ context.Context, _ valuer.UUID, _ string, _ *types.User, _ string) (*types.User, error) {
	panic("not implemented")
}
func (m *mockUserModuleForAuthn) DeleteUser(_ context.Context, _ valuer.UUID, _, _ string) error {
	panic("not implemented")
}
func (m *mockUserModuleForAuthn) CreateBulkInvite(_ context.Context, _ valuer.UUID, _ valuer.UUID, _ *types.PostableBulkInviteRequest) ([]*types.Invite, error) {
	panic("not implemented")
}
func (m *mockUserModuleForAuthn) ListInvite(_ context.Context, _ string) ([]*types.Invite, error) {
	panic("not implemented")
}
func (m *mockUserModuleForAuthn) DeleteInvite(_ context.Context, _ string, _ valuer.UUID) error {
	panic("not implemented")
}
func (m *mockUserModuleForAuthn) AcceptInvite(_ context.Context, _, _ string) (*types.User, error) {
	panic("not implemented")
}
func (m *mockUserModuleForAuthn) GetInviteByToken(_ context.Context, _ string) (*types.Invite, error) {
	panic("not implemented")
}
func (m *mockUserModuleForAuthn) CreateAPIKey(_ context.Context, _ *types.StorableAPIKey) error {
	panic("not implemented")
}
func (m *mockUserModuleForAuthn) UpdateAPIKey(_ context.Context, _ valuer.UUID, _ *types.StorableAPIKey, _ valuer.UUID) error {
	panic("not implemented")
}
func (m *mockUserModuleForAuthn) ListAPIKeys(_ context.Context, _ valuer.UUID) ([]*types.StorableAPIKeyUser, error) {
	panic("not implemented")
}
func (m *mockUserModuleForAuthn) RevokeAPIKey(_ context.Context, _ valuer.UUID, _ valuer.UUID) error {
	panic("not implemented")
}
func (m *mockUserModuleForAuthn) GetAPIKey(_ context.Context, _ valuer.UUID, _ valuer.UUID) (*types.StorableAPIKeyUser, error) {
	panic("not implemented")
}
func (m *mockUserModuleForAuthn) Collect(_ context.Context, _ valuer.UUID) (map[string]any, error) {
	panic("not implemented")
}

// testOrg returns a default Organization for authn tests.
func testOrg() *types.Organization {
	return types.NewOrganization("Test Org")
}

// makeKO11yTokenForAuthn creates a signed RS256 JWT with the given userID for testing.
func makeKO11yTokenForAuthn(t *testing.T, key *rsa.PrivateKey, userID string) string {
	t.Helper()
	claims := KO11yClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID:   userID,
		TenantID: "test-tenant",
		Role:     "admin",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, &claims)
	signed, err := token.SignedString(key)
	require.NoError(t, err)
	return signed
}

// TestIsKO11ySSO verifies context key detection for K-O11y SSO authentication.
func TestIsKO11ySSO(t *testing.T) {
	ctx := context.Background()

	// When no key is set, should return false
	assert.False(t, isKO11ySSO(ctx))

	// When key is set to true, should return true
	ctx = context.WithValue(ctx, ko11ySSOKey{}, true)
	assert.True(t, isKO11ySSO(ctx))

	// When key is set to false, should return false
	ctx2 := context.WithValue(context.Background(), ko11ySSOKey{}, false)
	assert.False(t, isKO11ySSO(ctx2))
}

// TestParseBearerAuth verifies Bearer token extraction from Authorization header values.
func TestParseBearerAuth(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantTok string
		wantOK  bool
	}{
		{
			name:    "valid bearer prefix",
			input:   "Bearer mytoken123",
			wantTok: "mytoken123",
			wantOK:  true,
		},
		{
			name:    "case insensitive bearer prefix",
			input:   "bearer mytoken123",
			wantTok: "mytoken123",
			wantOK:  true,
		},
		{
			name:    "BEARER uppercase prefix",
			input:   "BEARER mytoken123",
			wantTok: "mytoken123",
			wantOK:  true,
		},
		{
			name:    "no bearer prefix returns empty",
			input:   "mytoken123",
			wantTok: "",
			wantOK:  false,
		},
		{
			name:    "empty string",
			input:   "",
			wantTok: "",
			wantOK:  false,
		},
		{
			name:    "bearer prefix only",
			input:   "Bearer ",
			wantTok: "",
			wantOK:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok, ok := parseBearerAuth(tt.input)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantTok, tok)
		})
	}
}

// TestSetKO11ySSO ensures SetKO11ySSO populates both ko11yValidator and ko11ySSO fields.
func TestSetKO11ySSO(t *testing.T) {
	mockTok := &mockTokenizer{err: assert.AnError}
	authN := NewAuthN([]string{"Authorization"}, &mockSharder{}, mockTok, testLogger())

	validator := NewKO11yValidator(true, testPublicKey, "", "EDITOR", nil, testLogger())
	orgGetter := &mockOrgGetterForAuthn{org: testOrg()}
	userMod := newMockUserModuleForAuthn()
	ko11ySSO := user.NewKO11ySSO(userMod, orgGetter, testLogger())

	// Before SetKO11ySSO, both fields are nil
	assert.Nil(t, authN.ko11yValidator)
	assert.Nil(t, authN.ko11ySSO)

	authN.SetKO11ySSO(validator, ko11ySSO)

	assert.NotNil(t, authN.ko11yValidator)
	assert.NotNil(t, authN.ko11ySSO)
}

// TestContextFromAccessToken_MissingToken verifies error when no auth header values are provided.
func TestContextFromAccessToken_MissingToken(t *testing.T) {
	mockTok := &mockTokenizer{err: assert.AnError}
	authN := NewAuthN([]string{"Authorization"}, &mockSharder{}, mockTok, testLogger())

	ctx, err := authN.contextFromAccessToken(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing authorization header")
	_ = ctx
}

// TestContextFromAccessToken_WithToken verifies raw token extraction from context.
func TestContextFromAccessToken_WithToken(t *testing.T) {
	mockTok := &mockTokenizer{err: assert.AnError}
	authN := NewAuthN([]string{"Authorization"}, &mockSharder{}, mockTok, testLogger())

	ctx, err := authN.contextFromAccessToken(context.Background(), "mytoken")
	require.NoError(t, err)

	tok, tokErr := authtypes.AccessTokenFromContext(ctx)
	require.NoError(t, tokErr)
	assert.Equal(t, "mytoken", tok)
}

// TestContextFromAccessToken_BearerToken verifies Bearer prefix is stripped from the token.
func TestContextFromAccessToken_BearerToken(t *testing.T) {
	mockTok := &mockTokenizer{err: assert.AnError}
	authN := NewAuthN([]string{"Authorization"}, &mockSharder{}, mockTok, testLogger())

	ctx, err := authN.contextFromAccessToken(context.Background(), "Bearer mytoken123")
	require.NoError(t, err)

	tok, tokErr := authtypes.AccessTokenFromContext(ctx)
	require.NoError(t, tokErr)
	assert.Equal(t, "mytoken123", tok)
}

// TestContextFromAccessToken_FirstNonEmpty verifies the first non-empty value is used.
func TestContextFromAccessToken_FirstNonEmpty(t *testing.T) {
	mockTok := &mockTokenizer{err: assert.AnError}
	authN := NewAuthN([]string{"Authorization"}, &mockSharder{}, mockTok, testLogger())

	ctx, err := authN.contextFromAccessToken(context.Background(), "", "secondtoken")
	require.NoError(t, err)

	tok, tokErr := authtypes.AccessTokenFromContext(ctx)
	require.NoError(t, tokErr)
	assert.Equal(t, "secondtoken", tok)
}

// TestContextFromKO11yToken_ValidToken verifies K-O11y JWT validation and user provisioning succeed.
func TestContextFromKO11yToken_ValidToken(t *testing.T) {
	org := testOrg()
	orgGetter := &mockOrgGetterForAuthn{org: org}
	userMod := newMockUserModuleForAuthn()
	ko11ySSO := user.NewKO11ySSO(userMod, orgGetter, testLogger())

	validator := NewKO11yValidator(true, testPublicKey, "", "EDITOR", nil, testLogger())
	mockTok := &mockTokenizer{err: assert.AnError}
	authN := NewAuthN([]string{"Authorization"}, &mockSharder{}, mockTok, testLogger())
	authN.SetKO11ySSO(validator, ko11ySSO)

	tokenStr := makeKO11yTokenForAuthn(t, testPrivateKey, "usr_abc123")
	ctx, err := authN.contextFromKO11yToken(context.Background(), tokenStr)
	require.NoError(t, err)

	// Context should have claims set from K-O11y user
	claims, claimsErr := authtypes.ClaimsFromContext(ctx)
	require.NoError(t, claimsErr)
	assert.Equal(t, "usr_abc123@ko11y.local", claims.Email)
	assert.Equal(t, types.RoleAdmin, claims.Role)

	// Context should be marked as K-O11y SSO authenticated
	assert.True(t, isKO11ySSO(ctx))
}

// TestContextFromKO11yToken_InvalidToken verifies an invalid JWT returns an error.
func TestContextFromKO11yToken_InvalidToken(t *testing.T) {
	mockTok := &mockTokenizer{err: assert.AnError}
	authN := NewAuthN([]string{"Authorization"}, &mockSharder{}, mockTok, testLogger())

	validator := NewKO11yValidator(true, testPublicKey, "", "EDITOR", nil, testLogger())
	org := testOrg()
	orgGetter := &mockOrgGetterForAuthn{org: org}
	userMod := newMockUserModuleForAuthn()
	ko11ySSO := user.NewKO11ySSO(userMod, orgGetter, testLogger())
	authN.SetKO11ySSO(validator, ko11ySSO)

	_, err := authN.contextFromKO11yToken(context.Background(), "invalid-token-string")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token")
}

// TestContextFromKO11yToken_SSOProvisioningError verifies user provisioning failure is wrapped.
func TestContextFromKO11yToken_SSOProvisioningError(t *testing.T) {
	// No organizations available - provisioning will fail
	orgGetter := &mockOrgGetterForAuthn{org: nil}
	userMod := newMockUserModuleForAuthn()
	ko11ySSO := user.NewKO11ySSO(userMod, orgGetter, testLogger())

	validator := NewKO11yValidator(true, testPublicKey, "", "EDITOR", nil, testLogger())
	mockTok := &mockTokenizer{err: assert.AnError}
	authN := NewAuthN([]string{"Authorization"}, &mockSharder{}, mockTok, testLogger())
	authN.SetKO11ySSO(validator, ko11ySSO)

	tokenStr := makeKO11yTokenForAuthn(t, testPrivateKey, "usr_test")
	_, err := authN.contextFromKO11yToken(context.Background(), tokenStr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SSO user provisioning failed")
}
