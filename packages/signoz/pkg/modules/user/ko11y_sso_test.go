package user

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockOrgGetter implements organization.Getter for testing.
type mockOrgGetter struct {
	orgs []*types.Organization
	err  error
}

func (m *mockOrgGetter) Get(_ context.Context, id valuer.UUID) (*types.Organization, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, org := range m.orgs {
		if org.ID == id {
			return org, nil
		}
	}
	return nil, nil
}

func (m *mockOrgGetter) ListByOwnedKeyRange(_ context.Context) ([]*types.Organization, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.orgs, nil
}

// mockUserModule is a minimal mock of user.Module for testing.
// It only implements GetOrCreateUser; other methods panic if called.
type mockUserModule struct {
	users       map[string]*types.User // keyed by email
	createErr   error
	createCount int
}

func newMockUserModule() *mockUserModule {
	return &mockUserModule{users: make(map[string]*types.User)}
}

func (m *mockUserModule) GetOrCreateUser(_ context.Context, user *types.User, _ ...CreateUserOption) (*types.User, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	key := user.Email.StringValue() + "|" + user.OrgID.StringValue()
	if existing, ok := m.users[key]; ok {
		return existing, nil
	}
	m.users[key] = user
	m.createCount++
	return user, nil
}

// Unused interface methods - panics to catch unintended calls.
func (m *mockUserModule) CreateFirstUser(context.Context, *types.Organization, string, valuer.Email, string) (*types.User, error) {
	panic("not implemented")
}
func (m *mockUserModule) CreateUser(context.Context, *types.User, ...CreateUserOption) error {
	panic("not implemented")
}
func (m *mockUserModule) GetOrCreateResetPasswordToken(context.Context, valuer.UUID) (*types.ResetPasswordToken, error) {
	panic("not implemented")
}
func (m *mockUserModule) UpdatePasswordByResetPasswordToken(context.Context, string, string) error {
	panic("not implemented")
}
func (m *mockUserModule) UpdatePassword(context.Context, valuer.UUID, string, string) error {
	panic("not implemented")
}
func (m *mockUserModule) UpdateUser(context.Context, valuer.UUID, string, *types.User, string) (*types.User, error) {
	panic("not implemented")
}
func (m *mockUserModule) DeleteUser(context.Context, valuer.UUID, string, string) error {
	panic("not implemented")
}
func (m *mockUserModule) CreateBulkInvite(context.Context, valuer.UUID, valuer.UUID, *types.PostableBulkInviteRequest) ([]*types.Invite, error) {
	panic("not implemented")
}
func (m *mockUserModule) ListInvite(context.Context, string) ([]*types.Invite, error) {
	panic("not implemented")
}
func (m *mockUserModule) DeleteInvite(context.Context, string, valuer.UUID) error {
	panic("not implemented")
}
func (m *mockUserModule) AcceptInvite(context.Context, string, string) (*types.User, error) {
	panic("not implemented")
}
func (m *mockUserModule) GetInviteByToken(context.Context, string) (*types.Invite, error) {
	panic("not implemented")
}
func (m *mockUserModule) CreateAPIKey(context.Context, *types.StorableAPIKey) error {
	panic("not implemented")
}
func (m *mockUserModule) UpdateAPIKey(context.Context, valuer.UUID, *types.StorableAPIKey, valuer.UUID) error {
	panic("not implemented")
}
func (m *mockUserModule) ListAPIKeys(context.Context, valuer.UUID) ([]*types.StorableAPIKeyUser, error) {
	panic("not implemented")
}
func (m *mockUserModule) RevokeAPIKey(context.Context, valuer.UUID, valuer.UUID) error {
	panic("not implemented")
}
func (m *mockUserModule) GetAPIKey(context.Context, valuer.UUID, valuer.UUID) (*types.StorableAPIKeyUser, error) {
	panic("not implemented")
}
func (m *mockUserModule) Collect(context.Context, valuer.UUID) (map[string]any, error) {
	panic("not implemented")
}

func ssoTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func defaultOrg() *types.Organization {
	org := types.NewOrganization("Default Org")
	return org
}

func TestFindOrCreateUser_NewUser(t *testing.T) {
	org := defaultOrg()
	orgGetter := &mockOrgGetter{orgs: []*types.Organization{org}}
	userMod := newMockUserModule()
	sso := NewKO11ySSO(userMod, orgGetter, ssoTestLogger())

	user, err := sso.FindOrCreateUser(context.Background(), "usr_abc@ko11y.local", "K-O11y User", types.RoleEditor)
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "usr_abc@ko11y.local", user.Email.StringValue())
	assert.Equal(t, types.RoleEditor, user.Role)
	assert.Equal(t, org.ID, user.OrgID)
	assert.Equal(t, 1, userMod.createCount)
}

func TestFindOrCreateUser_ExistingUser(t *testing.T) {
	org := defaultOrg()
	orgGetter := &mockOrgGetter{orgs: []*types.Organization{org}}
	userMod := newMockUserModule()
	sso := NewKO11ySSO(userMod, orgGetter, ssoTestLogger())

	// First call creates user
	user1, err := sso.FindOrCreateUser(context.Background(), "usr_abc@ko11y.local", "K-O11y User", types.RoleEditor)
	require.NoError(t, err)

	// Second call returns existing user
	user2, err := sso.FindOrCreateUser(context.Background(), "usr_abc@ko11y.local", "K-O11y User", types.RoleEditor)
	require.NoError(t, err)

	assert.Equal(t, user1.ID, user2.ID)
	assert.Equal(t, 1, userMod.createCount) // Only one creation
}

func TestFindOrCreateUser_DifferentOrg(t *testing.T) {
	org := defaultOrg()
	orgGetter := &mockOrgGetter{orgs: []*types.Organization{org}}
	userMod := newMockUserModule()
	sso := NewKO11ySSO(userMod, orgGetter, ssoTestLogger())

	user1, err := sso.FindOrCreateUser(context.Background(), "usr_a@ko11y.local", "User A", types.RoleEditor)
	require.NoError(t, err)

	user2, err := sso.FindOrCreateUser(context.Background(), "usr_b@ko11y.local", "User B", types.RoleAdmin)
	require.NoError(t, err)

	assert.NotEqual(t, user1.Email, user2.Email)
	assert.Equal(t, 2, userMod.createCount)
}

func TestFindOrCreateUser_AdminRole(t *testing.T) {
	org := defaultOrg()
	orgGetter := &mockOrgGetter{orgs: []*types.Organization{org}}
	userMod := newMockUserModule()
	sso := NewKO11ySSO(userMod, orgGetter, ssoTestLogger())

	user, err := sso.FindOrCreateUser(context.Background(), "admin_user@ko11y.local", "Admin User", types.RoleAdmin)
	require.NoError(t, err)
	assert.Equal(t, types.RoleAdmin, user.Role)
}

func TestFindOrCreateUser_NoOrganization(t *testing.T) {
	orgGetter := &mockOrgGetter{orgs: []*types.Organization{}}
	userMod := newMockUserModule()
	sso := NewKO11ySSO(userMod, orgGetter, ssoTestLogger())

	_, err := sso.FindOrCreateUser(context.Background(), "usr@ko11y.local", "User", types.RoleEditor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no organizations found")
}

func TestFindOrCreateUser_OrgError(t *testing.T) {
	orgGetter := &mockOrgGetter{err: assert.AnError}
	userMod := newMockUserModule()
	sso := NewKO11ySSO(userMod, orgGetter, ssoTestLogger())

	_, err := sso.FindOrCreateUser(context.Background(), "usr@ko11y.local", "User", types.RoleEditor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list organizations")
}

func TestFindOrCreateUser_InvalidEmail(t *testing.T) {
	org := defaultOrg()
	orgGetter := &mockOrgGetter{orgs: []*types.Organization{org}}
	userMod := newMockUserModule()
	sso := NewKO11ySSO(userMod, orgGetter, ssoTestLogger())

	_, err := sso.FindOrCreateUser(context.Background(), "", "User", types.RoleEditor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid email")
}

func TestFindOrCreateUser_DBError(t *testing.T) {
	org := defaultOrg()
	orgGetter := &mockOrgGetter{orgs: []*types.Organization{org}}
	userMod := newMockUserModule()
	userMod.createErr = assert.AnError
	sso := NewKO11ySSO(userMod, orgGetter, ssoTestLogger())

	_, err := sso.FindOrCreateUser(context.Background(), "usr@ko11y.local", "User", types.RoleEditor)
	require.Error(t, err)
}

func TestFindOrCreateUser_InvalidRole(t *testing.T) {
	// types.NewUser fails when role is empty string, covering the error branch
	// after valuer.NewEmail succeeds.
	org := defaultOrg()
	orgGetter := &mockOrgGetter{orgs: []*types.Organization{org}}
	userMod := newMockUserModule()
	sso := NewKO11ySSO(userMod, orgGetter, ssoTestLogger())

	_, err := sso.FindOrCreateUser(context.Background(), "usr@ko11y.local", "User", types.Role(""))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create SSO user")
}
