// Package api provides an optional net/http management handler for taskflow.
// Mount New on /api/ and /healthz without stripping their path prefixes.
// Applications own the HTTP server, authentication policy, and shutdown.
package api
