package user

import (
	"context"
	"log/slog"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/modules/organization"
	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/valuer"
)

// KO11ySSO handles JIT (Just-In-Time) user provisioning for K-O11y SSO.
// It wraps the existing user.Module and organization.Getter to find or
// create users based on K-O11y JWT claims.
type KO11ySSO struct {
	userModule Module
	orgGetter  organization.Getter
	logger     *slog.Logger
}

// NewKO11ySSO creates a new KO11ySSO provisioner.
func NewKO11ySSO(userModule Module, orgGetter organization.Getter, logger *slog.Logger) *KO11ySSO {
	return &KO11ySSO{
		userModule: userModule,
		orgGetter:  orgGetter,
		logger:     logger,
	}
}

// FindOrCreateUser finds an existing user by email or creates a new one.
// It resolves the default organization and delegates to GetOrCreateUser
// for idempotent user provisioning.
func (s *KO11ySSO) FindOrCreateUser(ctx context.Context, email string, displayName string, role types.Role) (*types.User, error) {
	// Resolve default organization
	orgID, err := s.getDefaultOrgID(ctx)
	if err != nil {
		return nil, err
	}

	// Build validated email
	validEmail, err := valuer.NewEmail(email)
	if err != nil {
		return nil, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "invalid email for SSO user: %s", err.Error())
	}

	// Build user object
	newUser, err := types.NewUser(displayName, validEmail, role, orgID)
	if err != nil {
		return nil, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "failed to create SSO user: %s", err.Error())
	}

	// GetOrCreateUser is idempotent: returns existing user if email+orgID match
	user, err := s.userModule.GetOrCreateUser(ctx, newUser)
	if err != nil {
		return nil, err
	}

	s.logger.InfoContext(ctx, "SSO user provisioned",
		"user_id", user.ID.StringValue(),
		"email", email,
		"role", string(role),
		"org_id", orgID.StringValue(),
	)

	return user, nil
}

// getDefaultOrgID retrieves the first organization's ID.
func (s *KO11ySSO) getDefaultOrgID(ctx context.Context) (valuer.UUID, error) {
	orgs, err := s.orgGetter.ListByOwnedKeyRange(ctx)
	if err != nil {
		return valuer.UUID{}, errors.Newf(errors.TypeInternal, errors.CodeInternal, "failed to list organizations: %s", err.Error())
	}

	if len(orgs) == 0 {
		return valuer.UUID{}, errors.New(errors.TypeInternal, errors.CodeInternal, "no organizations found for SSO")
	}

	return orgs[0].ID, nil
}
