package policy

// The route policy table.
//
// Every route the router serves is listed here by method and full Gin path, and
// the route-inventory test fails when a registered route is missing. The
// requirements mirror what the route previously enforced inline; keeping the
// table explicit means a new route cannot inherit the wrong authority by
// accident, and a route guarded only in a handler is a visible gap rather than
// a quiet one.
func init() {
	// ── Public, no session ────────────────────────────────────────────────
	public("GET", "/llms.txt", "/llms-full.txt", "/.well-known/jwks.json",
		"/api/health", "/api/v1/health", "/api/v1/blogs", "/api/v1/blogs/:slug")
	public("GET", "/api/v1/auth/oauth/:provider", "/api/v1/auth/oauth/:provider/callback",
		"/api/v1/auth/sso/:slug", "/api/v1/auth/sso/:slug/callback",
		"/api/v1/auth/saml/:slug", "/api/v1/auth/saml/:slug/metadata")
	public("GET", "/api/v1/public/forms/:token")
	public("POST", "/api/v1/auth/register", "/api/v1/auth/login", "/api/v1/auth/refresh",
		"/api/v1/auth/forgot-password", "/api/v1/auth/reset-password",
		"/api/v1/auth/verify-email", "/api/v1/auth/magic-link",
		"/api/v1/auth/magic-link/consume",
		"/api/v1/auth/passkeys/login/begin", "/api/v1/auth/passkeys/login/finish",
		"/api/v1/auth/totp/verify", "/api/v1/auth/totp/backup-codes/verify",
		"/api/v1/auth/sso/discover", "/api/v1/auth/saml/:slug/acs")
	public("POST", "/api/v1/public/forms/:token/submit")
	// Provider signature verification is the boundary, not a session.
	public("POST", "/webhooks/:provider")
	// The WebSocket and SSE handshakes authenticate from the cookie or token in
	// the handler, because a browser cannot set an Authorization header on them.
	public("GET", "/api/ws", "/api/events")

	// ── Authenticated, self-service ───────────────────────────────────────
	authenticated("GET", "/api/v1/auth/me", "/api/v1/auth/permissions",
		"/api/v1/system/modules", "/api/v1/auth/sessions",
		"/api/v1/auth/magic-link/recent", "/api/v1/auth/totp/status",
		"/api/v1/auth/totp/trusted-devices", "/api/v1/auth/passkeys",
		"/api/v1/auth/security", "/api/v1/auth/recovery/email",
		"/api/v1/api-keys", "/api/v1/media/profiles",
		"/api/v1/uploads", "/api/v1/uploads/stats", "/api/v1/uploads/:id",
		"/api/v1/uploads/:id/download", "/api/v1/sync/pull", "/api/v1/sync/policy",
		"/api/v1/settings", "/api/v1/notifications", "/api/v1/dashboard-layout",
		"/api/v1/tickets", "/api/v1/tickets/:id", "/api/v1/imports/:id",
		"/api/v1/profile")
	authenticated("POST", "/api/v1/auth/logout", "/api/v1/auth/totp/setup",
		"/api/v1/auth/totp/enable", "/api/v1/auth/totp/email/send",
		"/api/v1/auth/totp/email/enable", "/api/v1/auth/totp/disable",
		"/api/v1/auth/totp/backup-codes", "/api/v1/auth/verify-email/send",
		"/api/v1/auth/passkeys/register/begin", "/api/v1/auth/passkeys/register/finish",
		"/api/v1/auth/recovery/email", "/api/v1/auth/recovery/email/verify",
		"/api/v1/auth/recovery/phone", "/api/v1/auth/recovery/phone/verify",
		"/api/v1/api-keys", "/api/v1/uploads", "/api/v1/uploads/presign",
		"/api/v1/uploads/complete", "/api/v1/sync/push",
		"/api/v1/notifications/:id/read", "/api/v1/notifications/read-all",
		"/api/v1/tickets", "/api/v1/tickets/:id/reply")
	authenticated("PUT", "/api/v1/profile", "/api/v1/dashboard-layout")
	authenticated("PATCH", "/api/v1/tickets/:id/close", "/api/v1/tickets/:id/reopen",
		"/api/v1/tickets/:id/assign", "/api/v1/auth/passkeys/:id")
	// The handler limits this to the caller's own record unless an ADMIN asks.
	authenticated("GET", "/api/v1/users/:id/gdpr-export")
	authenticated("POST", "/api/v1/auth/sessions/revoke-all")
	authenticated("DELETE", "/api/v1/auth/sessions/:id", "/api/v1/auth/passkeys/:id",
		"/api/v1/auth/totp/trusted-devices", "/api/v1/auth/totp/trusted-devices/:id",
		"/api/v1/auth/recovery/email", "/api/v1/auth/recovery/phone",
		"/api/v1/api-keys/:id", "/api/v1/uploads/:id", "/api/v1/profile")

	// Optional modules. Absent from the router when the module is off; declared
	// here so enabling it does not create an unlisted route.
	authenticated("POST", "/api/v1/ai/complete", "/api/v1/ai/chat", "/api/v1/ai/stream")

	// ── Staff: ADMIN or an explicit permission ────────────────────────────
	adminOr("users.view", "GET", "/api/v1/users", "/api/v1/users/:id")
	adminOr("users.create", "POST", "/api/v1/users")
	adminOr("users.edit", "PUT", "/api/v1/users/:id", "/api/v1/users/:id/roles")
	adminOr("users.edit", "POST", "/api/v1/users/:id/unlock")
	adminOr("users.delete", "DELETE", "/api/v1/users/:id")
	adminOr("users.delete", "POST", "/api/v1/users/:id/gdpr-erase")
	adminOr("audit.view", "GET", "/api/v1/gdpr/journal", "/api/v1/admin/activity",
		"/api/v1/admin/activity/integrity", "/api/v1/user-activity",
		"/api/v1/user-activity/stats", "/api/v1/audit/ocsf")
	adminOr("roles.view", "GET", "/api/v1/permissions", "/api/v1/roles",
		"/api/v1/roles/:id", "/api/v1/access-reviews", "/api/v1/access-reviews/:id")
	adminOr("roles.create", "POST", "/api/v1/roles")
	adminOr("roles.edit", "PUT", "/api/v1/roles/:id")
	adminOr("roles.delete", "DELETE", "/api/v1/roles/:id")
	adminOr("roles.edit", "POST", "/api/v1/access-reviews",
		"/api/v1/access-reviews/:id/items/:itemId/decision",
		"/api/v1/access-reviews/:id/complete")
	adminOr("system.view", "GET", "/api/v1/scale", "/api/v1/admin/mail/templates",
		"/api/v1/admin/mail/preview/:template", "/api/v1/admin/security/summary",
		"/api/v1/admin/observability/summary", "/api/v1/admin/trash",
		"/api/v1/admin/trash/:table", "/api/v1/admin/webhooks",
		"/api/v1/admin/flags", "/api/v1/admin/flags/:id/exposures")
	adminOr("jobs.view", "GET", "/api/v1/admin/jobs/stats", "/api/v1/admin/jobs/:status",
		"/api/v1/admin/sagas", "/api/v1/admin/sagas/definitions",
		"/api/v1/admin/sagas/:id", "/api/v1/admin/cron/tasks")
	adminOr("jobs.edit", "POST", "/api/v1/admin/jobs/:id/retry",
		"/api/v1/admin/sagas/:id/retry")
	adminOr("jobs.edit", "DELETE", "/api/v1/admin/jobs/queue/:queue")
	// The route explorer and settings are catalogued with a permission key the
	// scaffold's catalog does not define; ADMIN is the effective holder.
	adminOr("settings.view", "GET", "/api/v1/admin/routes")
	adminOr("blogs.view", "GET", "/api/v1/admin/blogs", "/api/v1/admin/blogs/:id")
	adminOr("blogs.create", "POST", "/api/v1/admin/blogs")
	adminOr("blogs.edit", "PUT", "/api/v1/admin/blogs/:id")
	adminOr("blogs.delete", "DELETE", "/api/v1/admin/blogs/:id")
	adminOr("backups.view", "GET", "/api/v1/backups", "/api/v1/backups/:id/download",
		"/api/v1/backup-settings")
	adminOr("backups.create", "POST", "/api/v1/backups/generate")
	adminOr("backups.edit", "PUT", "/api/v1/backup-settings")
	// The resource is in the URL; RequirePermissionFor checks the dynamic key.
	authenticated("GET", "/api/v1/admin/dashboard/resource-stats/:resource",
		"/api/v1/admin/dashboard/chart/:resource")

	// Restoration and permanent deletion are ADMIN and nothing less.
	adminOnly("POST", "/api/v1/admin/trash/:table/:id/restore")
	adminOnly("DELETE", "/api/v1/admin/trash/:table/:id", "/api/v1/admin/trash/:table")
	adminOnly("GET", "/api/v1/admin/deleted-accounts")
	adminOnly("POST", "/api/v1/admin/deleted-accounts/:id/restore")
	adminOnly("DELETE", "/api/v1/admin/deleted-accounts/:id")

	// ── Campus election access ────────────────────────────────────────────
	// The list is any signed-in caller's index of elections; the handler hides
	// other people's drafts.
	authenticated("GET", "/api/v1/elections")
	permitted("elections.create", "POST", "/api/v1/elections")
	permitted("elections.view", "GET", "/api/v1/elections/:id")
	permitted("elections.edit", "PUT", "/api/v1/elections/:id")
	permitted("elections.edit", "POST", "/api/v1/elections/:id/transitions")
	permitted("elections.roles", "GET", "/api/v1/elections/:id/roles")
	permitted("elections.roles", "PUT", "/api/v1/elections/:id/roles/:userId")
	permitted("elections.roles", "DELETE", "/api/v1/elections/:id/roles/:userId/:role")
	permitted("results.view", "GET", "/api/v1/elections/:id/results")
	permitted("elections.certify", "POST", "/api/v1/elections/:id/certify")

	// The voter's own eligibility and ballot. Any signed-in caller may ask;
	// the handler returns the ballot only when the server-side eligibility
	// check passes for this election.
	authenticated("GET", "/api/v1/elections/:id/eligibility", "/api/v1/elections/:id/ballot",
		"/api/v1/elections/:id/lifecycle", "/api/v1/elections/:id/my-vote-status")
	authenticated("POST", "/api/v1/elections/:id/ballot/start", "/api/v1/elections/:id/ballot/submit")

	// Roster evidence is officer-only and election-scoped in the handler.
	permitted("rosters.import", "POST", "/api/v1/elections/:id/roster/preview",
		"/api/v1/elections/:id/roster/import")
	permitted("rosters.edit", "POST", "/api/v1/elections/:id/roster/:rosterId/status",
		"/api/v1/elections/:id/eligibility/freeze")
	permitted("rosters.view", "GET", "/api/v1/elections/:id/eligibility/report")

	// ── ADMIN only ────────────────────────────────────────────────────────
	adminOnly("POST", "/api/v1/admin/activity/reseal", "/api/v1/admin/webhooks/:id/replay",
		"/api/v1/admin/flags", "/api/v1/sso/connections",
		"/api/v1/admin/form-shares")
	adminOnly("PUT", "/api/v1/admin/flags/:id", "/api/v1/sso/connections/:id",
		"/api/v1/settings")
	adminOnly("DELETE", "/api/v1/admin/flags/:id", "/api/v1/sso/connections/:id",
		"/api/v1/settings/:key")
	adminOnly("PATCH", "/api/v1/admin/form-shares/:id")
	adminOnly("DELETE", "/api/v1/admin/form-shares/:id")
	adminOnly("GET", "/api/v1/sso/connections", "/api/v1/sso/connections/:id/test",
		"/api/v1/admin/form-shares", "/api/v1/admin/form-shares/resources",
		"/api/v1/admin/form-shares/resources/:resource/fields",
		"/api/v1/admin/form-submissions")
}
