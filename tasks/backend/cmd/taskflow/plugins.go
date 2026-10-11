package main

import (
	"git.woa.com/mooyliu/mooy_test/backend/plugins/bark"
	"git.woa.com/mooyliu/mooy_test/backend/plugins/builtin"
	"git.woa.com/mooyliu/mooy_test/backend/plugins/reminder"
	"git.woa.com/mooyliu/mooy_test/framework"
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
