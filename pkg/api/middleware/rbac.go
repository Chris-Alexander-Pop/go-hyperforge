package middleware

import (
	"net/http"

	"github.com/chris-alexander-pop/system-design-library/pkg/api/rbac"
)

// RequirePermission returns middleware that allows the request only when one of
// the authenticated roles (from AuthMiddleware context) is permitted for
// resource/action by the given Enforcer.
func RequirePermission(enforcer rbac.Enforcer, resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roles := GetRoles(r.Context())
			if len(roles) == 0 {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			for _, role := range roles {
				ok, err := enforcer.Enforce(r.Context(), role, resource, action)
				if err != nil {
					http.Error(w, "internal error", http.StatusInternalServerError)
					return
				}
				if ok {
					next.ServeHTTP(w, r)
					return
				}
			}

			http.Error(w, "forbidden", http.StatusForbidden)
		})
	}
}
