package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/troubleman96/ziadapos/internal/security"
)

// context keys
const (
	ctxUserID    = "auth_user_id"
	ctxRole      = "auth_role"
	ctxOrgID     = "auth_org_id"
	ctxStoreID   = "auth_store_id"
	ctxClaims    = "auth_claims"
)

// Auth protects routes with a valid access JWT. Claims are stored in the
// gin context so handlers/services can scope queries to the user.
func Auth(m *security.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			abortUnauthorized(c, "missing authorization header")
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer"))
		if token == "" {
			abortUnauthorized(c, "invalid authorization header")
			return
		}

		claims, err := m.ParseAccess(token)
		if err != nil {
			abortUnauthorized(c, "invalid or expired token")
			return
		}

		c.Set(ctxUserID, claims.UserID.String())
		c.Set(ctxRole, claims.Role)
		c.Set(ctxOrgID, claims.OrgID.String())
		c.Set(ctxStoreID, claims.StoreID.String())
		c.Set(ctxClaims, claims)
		c.Next()
	}
}

func abortUnauthorized(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"success": false,
		"message": msg,
	})
}

// UserID returns the authenticated user's UUID as a string.
func UserID(c *gin.Context) string {
	v, _ := c.Get(ctxUserID)
	s, _ := v.(string)
	return s
}

// Role returns the authenticated user's role.
func Role(c *gin.Context) string {
	v, _ := c.Get(ctxRole)
	s, _ := v.(string)
	return s
}

// OrgID returns the authenticated user's organisation id ("" if unset).
func OrgID(c *gin.Context) string {
	v, _ := c.Get(ctxOrgID)
	s, _ := v.(string)
	return s
}

// StoreID returns the authenticated user's active store id ("" if unset).
func StoreID(c *gin.Context) string {
	v, _ := c.Get(ctxStoreID)
	s, _ := v.(string)
	return s
}

// Claims returns the parsed JWT claims.
func Claims(c *gin.Context) *security.Claims {
	v, _ := c.Get(ctxClaims)
	cl, _ := v.(*security.Claims)
	return cl
}

// RequireRoles allows only the listed roles through.
func RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool)
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		if !allowed[Role(c)] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "you do not have permission to perform this action",
			})
			return
		}
		c.Next()
	}
}

// RequireOwner allows admin and owner roles (owners manage org + stores).
func RequireOwner() gin.HandlerFunc {
	return RequireRoles("admin", "owner")
}

// RequireAdmin allows only platform admins.
func RequireAdmin() gin.HandlerFunc {
	return RequireRoles("admin")
}
