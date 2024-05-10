package middlewares

import (
	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"

	"github.com/pershin-daniil/ninja-chat-bank/internal/types"
)

func SetToken(c echo.Context, uid types.UserID) {
	c.Set(tokenCtxKey, &jwt.Token{
		Claims: &claims{
			StandardClaims: jwt.StandardClaims{
				Subject: uid.String(),
			},
		},
		Valid: true,
	})
}
