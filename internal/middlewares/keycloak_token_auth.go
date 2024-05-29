package middlewares

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	keycloakclient "github.com/pershin-daniil/ninja-chat-bank/internal/clients/keycloak"
	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/introspector_mock.gen.go -package=middlewaresmocks Introspector

const tokenCtxKey = "user-token"

var (
	ErrTokenNotActive         = errors.New("token not active")
	ErrNoRequiredResourceRole = errors.New("no required resource role")
)

type Introspector interface {
	IntrospectToken(ctx context.Context, token string) (*keycloakclient.IntrospectTokenResult, error)
}

// NewKeycloakTokenAuth returns a middleware that implements "active" authentication:
// each request is verified by the Keycloak server.
func NewKeycloakTokenAuth(introspector Introspector, resource, role, protocol string) echo.MiddlewareFunc {
	return middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		KeyLookup:  "header:" + echo.HeaderAuthorization + "," + "header:Sec-WebSocket-Protocol",
		AuthScheme: "Bearer",
		Validator: func(tokenStr string, eCtx echo.Context) (bool, error) {
			tokenStr = strings.TrimPrefix(tokenStr, protocol+", ")
			token, err := introspector.IntrospectToken(eCtx.Request().Context(), tokenStr)
			if err != nil {
				return false, fmt.Errorf("failed to introspect token: %w", err)
			}

			if !token.Active {
				return false, nil
			}

			parsedClaims := new(claims)
			parsedToken, _ := jwt.ParseWithClaims(tokenStr, parsedClaims, nil)
			if err = parsedClaims.Valid(); err != nil {
				return false, fmt.Errorf("failed to validate claims: %w", err)
			}

			roles, ok := parsedClaims.ResourceAccess[resource]
			if !ok {
				return false, ErrNoRequiredResourceRole
			}

			if !containsRole(role, roles.Roles) {
				return false, ErrNoRequiredResourceRole
			}

			eCtx.Set(tokenCtxKey, parsedToken)

			return true, nil
		},
	})
}

func containsRole(role string, roles []string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

func MustUserID(eCtx echo.Context) types.UserID {
	uid, ok := userID(eCtx)
	if !ok {
		panic("no user token in request context")
	}
	return uid
}

func userID(eCtx echo.Context) (types.UserID, bool) {
	t := eCtx.Get(tokenCtxKey)
	if t == nil {
		return types.UserIDNil, false
	}

	tt, ok := t.(*jwt.Token)
	if !ok {
		return types.UserIDNil, false
	}

	userIDProvider, ok := tt.Claims.(interface{ UserID() types.UserID })
	if !ok {
		return types.UserIDNil, false
	}
	return userIDProvider.UserID(), true
}
