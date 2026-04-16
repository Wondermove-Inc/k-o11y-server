package middleware

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testPrivateKey *rsa.PrivateKey
	testPublicKey  *rsa.PublicKey
)

func init() {
	var err error
	testPrivateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic("failed to generate test RSA key: " + err.Error())
	}
	testPublicKey = &testPrivateKey.PublicKey
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func makeKO11yToken(t *testing.T, key *rsa.PrivateKey, claims KO11yClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, &claims)
	signed, err := token.SignedString(key)
	require.NoError(t, err)
	return signed
}

func validClaims() KO11yClaims {
	return KO11yClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "ko11y",
		},
		UserID:   "usr_abc123",
		TenantID: "tenant_xyz",
		Role:     "admin",
	}
}

func TestNewKO11yValidator_Disabled(t *testing.T) {
	v := NewKO11yValidator(false, testPublicKey, "", "EDITOR", nil, testLogger())
	assert.Nil(t, v)
}

func TestNewKO11yValidator_NilPublicKey(t *testing.T) {
	v := NewKO11yValidator(true, nil, "", "EDITOR", nil, testLogger())
	assert.Nil(t, v)
}

func TestNewKO11yValidator_Enabled(t *testing.T) {
	v := NewKO11yValidator(true, testPublicKey, "ko11y", "EDITOR", nil, testLogger())
	require.NotNil(t, v)
	assert.True(t, v.IsEnabled())
	assert.Equal(t, types.RoleEditor, v.defaultRole)
}

func TestNewKO11yValidator_InvalidDefaultRole(t *testing.T) {
	v := NewKO11yValidator(true, testPublicKey, "", "INVALID_ROLE", nil, testLogger())
	require.NotNil(t, v)
	// Falls back to EDITOR when role is invalid
	assert.Equal(t, types.RoleEditor, v.defaultRole)
}

func TestIsEnabled_NilValidator(t *testing.T) {
	var v *KO11yValidator
	assert.False(t, v.IsEnabled())
}

func TestValidate_ValidToken(t *testing.T) {
	v := NewKO11yValidator(true, testPublicKey, "ko11y", "EDITOR", nil, testLogger())

	claims := validClaims()
	token := makeKO11yToken(t, testPrivateKey, claims)

	result, err := v.Validate(token)
	require.NoError(t, err)
	assert.Equal(t, "usr_abc123", result.UserID)
	assert.Equal(t, "tenant_xyz", result.TenantID)
	assert.Equal(t, "admin", result.Role)
}

func TestValidate_ExpiredToken(t *testing.T) {
	v := NewKO11yValidator(true, testPublicKey, "", "EDITOR", nil, testLogger())

	claims := validClaims()
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-1 * time.Hour))
	token := makeKO11yToken(t, testPrivateKey, claims)

	_, err := v.Validate(token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token")
}

func TestValidate_InvalidSignature(t *testing.T) {
	v := NewKO11yValidator(true, testPublicKey, "", "EDITOR", nil, testLogger())

	// Sign with a different private key
	wrongKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	claims := validClaims()
	token := makeKO11yToken(t, wrongKey, claims)

	_, err = v.Validate(token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token")
}

func TestValidate_InvalidIssuer(t *testing.T) {
	v := NewKO11yValidator(true, testPublicKey, "expected-issuer", "EDITOR", nil, testLogger())

	claims := validClaims()
	claims.Issuer = "wrong-issuer"
	token := makeKO11yToken(t, testPrivateKey, claims)

	_, err := v.Validate(token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid issuer")
}

func TestValidate_NoIssuerCheck(t *testing.T) {
	v := NewKO11yValidator(true, testPublicKey, "", "EDITOR", nil, testLogger())

	claims := validClaims()
	claims.Issuer = "anything"
	token := makeKO11yToken(t, testPrivateKey, claims)

	result, err := v.Validate(token)
	require.NoError(t, err)
	assert.Equal(t, "usr_abc123", result.UserID)
}

func TestValidate_MissingUserID(t *testing.T) {
	v := NewKO11yValidator(true, testPublicKey, "", "EDITOR", nil, testLogger())

	claims := validClaims()
	claims.UserID = ""
	token := makeKO11yToken(t, testPrivateKey, claims)

	_, err := v.Validate(token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing user_id")
}

func TestValidate_MalformedToken(t *testing.T) {
	v := NewKO11yValidator(true, testPublicKey, "", "EDITOR", nil, testLogger())

	_, err := v.Validate("not-a-jwt-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token")
}

func TestValidate_DisabledValidator(t *testing.T) {
	v := &KO11yValidator{enabled: false}
	_, err := v.Validate("any-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not enabled")
}

func TestValidate_UnsupportedSigningMethod_HS256(t *testing.T) {
	v := NewKO11yValidator(true, testPublicKey, "", "EDITOR", nil, testLogger())

	// Create a token signed with HS256 (should be rejected by RS256 validator)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &KO11yClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
		UserID: "usr_test",
	})
	signed, err := token.SignedString([]byte("some-secret"))
	require.NoError(t, err)

	_, err = v.Validate(signed)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token")
}

func TestMapRole(t *testing.T) {
	v := NewKO11yValidator(true, testPublicKey, "", "EDITOR", nil, testLogger())

	tests := []struct {
		input    string
		expected types.Role
	}{
		{"admin", types.RoleAdmin},
		{"tenant_admin", types.RoleAdmin},
		{"user", types.RoleEditor},
		{"viewer", types.RoleEditor},
		{"", types.RoleEditor},
		{"unknown", types.RoleEditor},
	}

	for _, tt := range tests {
		t.Run("role_"+tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, v.MapRole(tt.input))
		})
	}
}

func TestMapRole_ViewerDefault(t *testing.T) {
	v := NewKO11yValidator(true, testPublicKey, "", "VIEWER", nil, testLogger())

	assert.Equal(t, types.RoleAdmin, v.MapRole("admin"))
	assert.Equal(t, types.RoleViewer, v.MapRole("user"))
	assert.Equal(t, types.RoleViewer, v.MapRole(""))
}

func TestGetEmail_Fallback(t *testing.T) {
	claims := &KO11yClaims{UserID: "usr_abc123"}
	assert.Equal(t, "usr_abc123@ko11y.local", claims.GetEmail())
}

func TestGetEmail_Fallback_SpecialChars(t *testing.T) {
	claims := &KO11yClaims{UserID: "user.name+tag"}
	assert.Equal(t, "user.name+tag@ko11y.local", claims.GetEmail())
}

func TestGetEmail_WithEmail(t *testing.T) {
	claims := &KO11yClaims{UserID: "usr_abc123", Email: "real@company.com"}
	assert.Equal(t, "real@company.com", claims.GetEmail())
}

func TestGetEmail_EmptyEmail_Fallback(t *testing.T) {
	claims := &KO11yClaims{UserID: "usr_abc123", Email: ""}
	assert.Equal(t, "usr_abc123@ko11y.local", claims.GetEmail())
}

func TestToIdentity(t *testing.T) {
	v := NewKO11yValidator(true, testPublicKey, "", "EDITOR", nil, testLogger())

	claims := &KO11yClaims{
		UserID:   "usr_abc123",
		TenantID: "tenant_xyz",
		Role:     "admin",
	}
	userID := valuer.GenerateUUID()
	orgID := valuer.GenerateUUID()

	identity := v.ToIdentity(claims, userID, orgID)

	require.NotNil(t, identity)
	assert.Equal(t, userID, identity.UserID)
	assert.Equal(t, orgID, identity.OrgID)
	assert.Equal(t, types.RoleAdmin, identity.Role)
	assert.Equal(t, "usr_abc123@ko11y.local", identity.Email.StringValue())
}

func TestToIdentity_DefaultRole(t *testing.T) {
	v := NewKO11yValidator(true, testPublicKey, "", "EDITOR", nil, testLogger())

	claims := &KO11yClaims{
		UserID: "usr_def456",
		Role:   "viewer", // maps to default (EDITOR)
	}
	userID := valuer.GenerateUUID()
	orgID := valuer.GenerateUUID()

	identity := v.ToIdentity(claims, userID, orgID)

	require.NotNil(t, identity)
	assert.Equal(t, types.RoleEditor, identity.Role)
}

// --- CheckTenant tests ---

// mockTenantStore implements TenantStore for testing.
type mockTenantStore struct {
	tenants  []SSOTenant
	inserted []SSOTenant
	err      error
}

func (m *mockTenantStore) GetSSOAllowedTenants(_ context.Context) ([]SSOTenant, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.tenants, nil
}

func (m *mockTenantStore) InsertSSOAllowedTenant(_ context.Context, tenantID, lockedBy string) error {
	m.inserted = append(m.inserted, SSOTenant{TenantID: tenantID})
	m.tenants = append(m.tenants, SSOTenant{TenantID: tenantID})
	return nil
}

func TestCheckTenant_ExplicitList_Allowed(t *testing.T) {
	v := NewKO11yValidator(true, testPublicKey, "", "EDITOR", []string{"tenant_xyz", "tenant_abc"}, testLogger())
	claims := validClaims()

	err := v.CheckTenant(context.Background(), &claims)
	require.NoError(t, err)
}

func TestCheckTenant_ExplicitList_Denied(t *testing.T) {
	v := NewKO11yValidator(true, testPublicKey, "", "EDITOR", []string{"tenant_other"}, testLogger())
	claims := validClaims()

	err := v.CheckTenant(context.Background(), &claims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tenant tenant_xyz is not allowed")
}

func TestCheckTenant_ExplicitList_MissingTenantID(t *testing.T) {
	v := NewKO11yValidator(true, testPublicKey, "", "EDITOR", []string{"tenant_xyz"}, testLogger())
	claims := validClaims()
	claims.TenantID = ""

	err := v.CheckTenant(context.Background(), &claims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing tenant_id")
}

func TestCheckTenant_AllowAll(t *testing.T) {
	v := NewKO11yValidator(true, testPublicKey, "", "EDITOR", []string{"*"}, testLogger())
	claims := validClaims()

	err := v.CheckTenant(context.Background(), &claims)
	require.NoError(t, err)
}

func TestCheckTenant_AllowAll_AnyTenant(t *testing.T) {
	v := NewKO11yValidator(true, testPublicKey, "", "EDITOR", []string{"*"}, testLogger())
	claims := validClaims()
	claims.TenantID = "completely_random_tenant"

	err := v.CheckTenant(context.Background(), &claims)
	require.NoError(t, err)
}

func TestCheckTenant_AutoLock_FirstLogin(t *testing.T) {
	store := &mockTenantStore{tenants: nil}
	v := NewKO11yValidator(true, testPublicKey, "", "EDITOR", nil, testLogger())
	v.SetTenantStore(store)

	claims := validClaims()
	err := v.CheckTenant(context.Background(), &claims)
	require.NoError(t, err)

	// Should have inserted the tenant
	require.Len(t, store.inserted, 1)
	assert.Equal(t, "tenant_xyz", store.inserted[0].TenantID)
}

func TestCheckTenant_AutoLock_SameTenant(t *testing.T) {
	store := &mockTenantStore{
		tenants: []SSOTenant{{TenantID: "tenant_xyz"}},
	}
	v := NewKO11yValidator(true, testPublicKey, "", "EDITOR", nil, testLogger())
	v.SetTenantStore(store)

	claims := validClaims()
	err := v.CheckTenant(context.Background(), &claims)
	require.NoError(t, err)
	assert.Empty(t, store.inserted) // no new insert
}

func TestCheckTenant_AutoLock_DifferentTenant_Rejected(t *testing.T) {
	store := &mockTenantStore{
		tenants: []SSOTenant{{TenantID: "tenant_locked"}},
	}
	v := NewKO11yValidator(true, testPublicKey, "", "EDITOR", nil, testLogger())
	v.SetTenantStore(store)

	claims := validClaims() // tenant_xyz ≠ tenant_locked
	err := v.CheckTenant(context.Background(), &claims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tenant tenant_xyz is not allowed")
}

func TestCheckTenant_AutoLock_MultipleTenants(t *testing.T) {
	store := &mockTenantStore{
		tenants: []SSOTenant{{TenantID: "tenant_a"}, {TenantID: "tenant_xyz"}},
	}
	v := NewKO11yValidator(true, testPublicKey, "", "EDITOR", nil, testLogger())
	v.SetTenantStore(store)

	claims := validClaims()
	err := v.CheckTenant(context.Background(), &claims)
	require.NoError(t, err)
}

func TestCheckTenant_AutoLock_NoStore_GracefulDegradation(t *testing.T) {
	v := NewKO11yValidator(true, testPublicKey, "", "EDITOR", nil, testLogger())
	// No SetTenantStore called — store is nil

	claims := validClaims()
	err := v.CheckTenant(context.Background(), &claims)
	require.NoError(t, err) // should allow (graceful degradation)
}

func TestCheckTenant_AutoLock_MissingTenantID(t *testing.T) {
	store := &mockTenantStore{tenants: nil}
	v := NewKO11yValidator(true, testPublicKey, "", "EDITOR", nil, testLogger())
	v.SetTenantStore(store)

	claims := validClaims()
	claims.TenantID = ""

	err := v.CheckTenant(context.Background(), &claims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing tenant_id")
}
