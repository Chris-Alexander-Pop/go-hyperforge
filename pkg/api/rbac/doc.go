/*
Package rbac provides an in-memory Role-Based Access Control enforcer with wildcard
permissions and SmartRWMutex-guarded policy updates.

Pair with middleware.RequirePermission and AuthMiddleware (or pkg/auth.MiddlewareVerifier)
for HTTP enforcement. This is not a Casbin/OPA replacement.
*/
package rbac
