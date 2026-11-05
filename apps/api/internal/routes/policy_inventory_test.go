package routes

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/policy"
	"campvote/apps/api/internal/testkit"
)

// TestEveryRouteHasAPolicy fails when a route is registered without a declared
// policy. That is the whole point of the central table: adding an endpoint
// without deciding who may call it breaks the build rather than shipping a
// default-allow route.
func TestEveryRouteHasAPolicy(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(models.Models()...); err != nil {
		t.Fatal(err)
	}
	cfg := testkit.Config()
	cfg.CORSOrigins = []string{"http://localhost:3000"}

	engine := Setup(db, cfg, &Services{})
	for _, route := range engine.Routes() {
		if policy.IsSkipped(route.Path) {
			continue
		}
		if _, ok := policy.Lookup(route.Method, route.Path); !ok {
			t.Errorf("route %s %s has no policy; add it to internal/policy/routes.go", route.Method, route.Path)
		}
	}
}
