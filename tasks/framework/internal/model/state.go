package model

import "time"

// Clone returns an independent snapshot. Time zone locations are immutable
// and may be shared; configurations, slices and optional timestamps are not.
func (s State) Clone() State {
	copy := s
	copy.Tasks = make([]Task, len(s.Tasks))
	for i, t := range s.Tasks {
		copy.Tasks[i] = t.Clone()
	}
	copy.Channels = make([]Channel, len(s.Channels))
	for i, c := range s.Channels {
		copy.Channels[i] = c.Clone()
	}
	copy.Runs = make([]Run, len(s.Runs))
	for i, r := range s.Runs {
		copy.Runs[i] = r.Clone()
	}
	return copy
}

func (t Task) Clone() Task {
	t.Config = append(t.Config[:0:0], t.Config...)
	t.NotifierIDs = append([]string{}, t.NotifierIDs...)
	t.Schedule.Weekdays = append([]int(nil), t.Schedule.Weekdays...)
	t.NextRun = cloneTime(t.NextRun)
	return t
}

func (c Channel) Clone() Channel {
	c.Config = append(c.Config[:0:0], c.Config...)
	return c
}

func (r Run) Clone() Run {
	r.Deliveries = append([]Delivery{}, r.Deliveries...)
	r.FinishedAt = cloneTime(r.FinishedAt)
	return r
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
