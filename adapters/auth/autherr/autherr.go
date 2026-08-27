// package: autherr / authn
// type:    errors
// job:     the auth port's rejection sentinel, held where every backend can return it
// limits:  one error value; auth.go re-exports it as auth.ErrUnauthenticated (-> auth.go)
//
// auth.go already imports each backend to dispatch New, so a backend importing auth back
// to return auth.ErrUnauthenticated would cycle. Backends return autherr.ErrUnauthenticated
// instead — the same value auth.ErrUnauthenticated re-exports — so errors.Is(err,
// auth.ErrUnauthenticated) still finds it from outside the port.
package autherr

import "errors"

// ErrUnauthenticated reports that a credential was required but missing or invalid.
var ErrUnauthenticated = errors.New("auth: unauthenticated")
