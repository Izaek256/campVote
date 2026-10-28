// Package policy is the server-side authorization layer for campus elections.
//
// It has two jobs. RoutePolicy answers "may this role touch this endpoint at
// all", which is every route in the API and is declared in routes.go. Election
// answers "may this person touch this election", which is object-level and
// cannot be known from the URL alone.
//
// The important default is denial. A matched route with no declaration is
// refused, and the route-inventory test in internal/routes fails before such a
// route can ship. Adding a route therefore means adding a policy, not
// remembering one.
package policy

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"campvote/apps/api/internal/authz"
	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/respond"
	"campvote/apps/api/internal/services"
)

// Kind is the shape of a route requirement.
type Kind int

const (
	// Public routes have no signed-in user: sign-in itself, public results,
	// webhooks whose signature is the boundary, and health checks.
	Public Kind = iota
	// Authenticated routes need a live session but no particular grant. The
	// handler still scopes rows it returns.
	Authenticated
	// Permission routes need one permission key from the catalog.
	Permission
	// Role routes need one legacy role name.
	Role
	// RoleOrPermission is the scaffold's existing OR guard: ADMIN, or a holder
	// of the named permission. It is mapped 1:1 from RequireRole calls.
	RoleOrPermission
)

// Requirement is one route's declared authorization.
type Requirement struct {
	Kind       Kind
	Role       string
	Permission string
}

// Declared routes, keyed by "METHOD /full/gin/path".
var routeTable = map[string]Requirement{}

func key(method, path string) string { return method + " " + path }

func declare(req Requirement, method string, paths ...string) {
	for _, path := range paths {
		k := key(method, path)
		if _, exists := routeTable[k]; exists {
			panic("policy: duplicate route policy for " + k)
		}
		routeTable[k] = req
	}
}

func public(method string, paths ...string) { declare(Requirement{Kind: Public}, method, paths...) }
func authenticated(method string, paths ...string) {
	declare(Requirement{Kind: Authenticated}, method, paths...)
}
func permitted(permission, method string, paths ...string) {
	declare(Requirement{Kind: Permission, Permission: permission}, method, paths...)
}
func adminOnly(method string, paths ...string) {
	declare(Requirement{Kind: Role, Role: models.RoleAdmin}, method, paths...)
}
func adminOr(permission, method string, paths ...string) {
	declare(Requirement{Kind: RoleOrPermission, Role: models.RoleAdmin, Permission: permission}, method, paths...)
}

// Lookup returns the policy for a matched route.
func Lookup(method, path string) (Requirement, bool) {
	req, ok := routeTable[key(method, path)]
	return req, ok
}

// skipPrefixes are mounted by libraries and are not this API's routes. They
// carry their own credentials and are excluded from the inventory.
var skipPrefixes = []string{"/docs", "/pulse", "/sentinel", "/studio", "/files/"}

// IsSkipped reports whether a path belongs to a mounted library rather than
// this API's own route table.
func IsSkipped(path string) bool { return skipped(path) }

func skipped(path string) bool {
	for _, p := range skipPrefixes {
		if len(path) >= len(p) && path[:len(p)] == p {
			return true
		}
	}
	return false
}

// Declared refuses any matched route that has no declaration. It runs before
// authentication, so it is the last net for a route somebody mounted without a
// policy, whichever group it landed in.
func Declared() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.FullPath()
		if path == "" || skipped(path) {
			c.Next()
			return
		}
		if _, ok := Lookup(c.Request.Method, path); !ok {
			log.Printf("policy: no declaration for %s %s", c.Request.Method, path)
			respond.Fail(c, respond.CodeForbidden, "This endpoint has no authorization policy and is denied")
			c.Abort()
			return
		}
		c.Next()
	}
}

// Guard enforces the route's declared requirement. Mount it after Auth on
// every authenticated group; a route absent from the table is denied.
func Guard(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		req, ok := Lookup(c.Request.Method, c.FullPath())
		if !ok {
			deny(c, db, "", "no declared policy", "")
			return
		}

		if req.Kind == Public {
			c.Next()
			return
		}
		if c.GetString("user_id") == "" {
			respond.Fail(c, respond.CodeUnauthorized, "Authentication required")
			c.Abort()
			return
		}
		if req.Kind == Authenticated {
			c.Next()
			return
		}

		if req.Kind == RoleOrPermission && c.GetString("user_role") == req.Role {
			c.Next()
			return
		}
		if req.Kind == Role && c.GetString("user_role") == req.Role {
			c.Next()
			return
		}
		if req.Permission != "" && hasPermission(c, req.Permission) {
			c.Next()
			return
		}

		required := req.Permission
		if req.Role != "" {
			required = req.Role
			if req.Permission != "" {
				required += " or " + req.Permission
			}
		}
		deny(c, db, c.GetString("user_id"), "route requires "+required, "")
	}
}

func hasPermission(c *gin.Context, permission string) bool {
	value, ok := c.Get("user_grants")
	if !ok {
		return false
	}
	grants, ok := value.([]string)
	return ok && authz.Granted(grants, permission)
}

// deny writes the 403 and records who was refused and what they asked for. It
// never records request bodies or ballot content.
func deny(c *gin.Context, db *gorm.DB, userID, reason, resourceID string) {
	if db != nil {
		services.LogActivity(db, c, services.ActivityArgs{
			UserID:       userID,
			Action:       "authz.denied",
			Severity:     "warn",
			Summary:      "Denied " + c.Request.Method + " " + c.FullPath() + ": " + reason,
			ResourceType: "route",
			ResourceID:   resourceID,
		})
	}
	respond.Fail(c, respond.CodeForbidden, "You do not have permission to perform this action")
	c.Abort()
}

// StatusIsDenied is a convenience for tests: it reports whether status is one
// of the deliberate refusals this layer writes.
func StatusIsDenied(status int) bool {
	return status == http.StatusUnauthorized || status == http.StatusForbidden || status == http.StatusNotFound
}
