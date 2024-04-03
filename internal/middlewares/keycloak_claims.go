package middlewares

import (
	"errors"

	"github.com/golang-jwt/jwt"

	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

var (
	ErrNoAllowedResources = errors.New("no allowed resources")
	ErrSubjectNotDefined  = errors.New(`"sub" is not defined`)
)

type claims struct {
	jwt.StandardClaims
	ResourceAccess map[string]struct {
		Roles []string `json:"roles"`
	} `json:"resource_access"`
}

// Valid returns errors:
// - from StandardClaims validation;
// - ErrNoAllowedResources, if claims doesn't contain `resource_access` map, or it's empty;
// - ErrSubjectNotDefined, if claims doesn't contain `sub` field or subject is zero UUID.
func (c claims) Valid() error {
	if err := c.StandardClaims.Valid(); err != nil {
		return err
	}

	if c.UserID().IsZero() {
		return ErrSubjectNotDefined
	}

	if len(c.ResourceAccess) == 0 {
		return ErrNoAllowedResources
	}

	return nil
}

func (c claims) UserID() types.UserID {
	id, _ := types.Parse[types.UserID](c.Subject)
	return id
}
