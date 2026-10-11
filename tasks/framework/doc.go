// Package taskflow provides scheduling, execution records and notification
// routing for applications that inject storage and register their plugins.
// New initializes state; Start begins execution with the host's context.
// Close cancels and waits for the engine. Storage is owned by the host.
// Management, configuration files and business plugins belong to host apps.
package taskflow
