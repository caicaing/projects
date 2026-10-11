package main

import (
	"github.com/caicaing/projects/tasks/backend/plugins/bark"
	"github.com/caicaing/projects/tasks/backend/plugins/builtin"
	"github.com/caicaing/projects/tasks/backend/plugins/reminder"
	"github.com/caicaing/projects/tasks/framework"
)

// Register constructors with r.Tasks.Register and r.Notifiers.Register.
// Factories create isolated instances; plugins appear in the console.
func registerPlugins(r *taskflow.Registry) error {
	if err := builtin.Register(r); err != nil {
		return err
	}
	if err := reminder.Register(r); err != nil {
		return err
	}
	return bark.Register(r)
}
