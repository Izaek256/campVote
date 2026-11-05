package access

// table maps "<METHOD> <path>" to its Requirement.
// This registry provides route access requirements for route explorer and documentation.
var table = map[string]Requirement{
	"GET /api/v1/auth/me": {
		Group:         "protected",
		Authenticated: true,
		Summary:       "signed in",
	},
}
