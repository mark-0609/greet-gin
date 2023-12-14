package middleware

import (
	"greet_gin/tools"
	//"greet-gin/tools"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const (
	RequestTokenName = "X-Request-Token"
	BearerLength     = len("Bearer ")
)

func JwtMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		token := c.GetHeader(RequestTokenName)

		logrus.Infof("token: %v", token)
		if token == "" {
			hToken := c.GetHeader("Authorization")
			if len(hToken) < BearerLength {
				c.AbortWithStatusJSON(http.StatusPreconditionFailed, gin.H{"msg": "header Authorization has not Bearer token", "code": -1, "data": nil})
				return
			}
			token = strings.TrimSpace(hToken[BearerLength:])
		}

		jwtRes := new(tools.JWT)
		jwtRes.SigningKey = []byte("test")
		claims, err := jwtRes.ParseToken(token)
		if err != nil {
			logrus.Errorf("auth middleware err: %v", err)
			c.AbortWithStatusJSON(http.StatusPreconditionFailed, gin.H{"msg": err.Error(), "code": -1, "data": nil})
			return
		}
		logrus.Infof("claims: %v", claims)

		// Set XToken header
		c.Writer.Header().Set("token", "tokenValue")
		// c.Set("test", "tesval")
		c.Next()
		return
	}
}
