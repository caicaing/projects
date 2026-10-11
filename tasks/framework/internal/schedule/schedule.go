package schedule

import (
	"fmt"

	"git.woa.com/mooyliu/mooy_test/framework/internal/model"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"
)

type Plan struct {
	interval time.Duration
	location *time.Location
	fields   [5]map[int]bool
	wildDay  bool
	wildWeek bool
}

// Preview returns upcoming times without modifying engine state.
func Preview(s model.Schedule, after time.Time) ([]time.Time, error) {
	p, err := Parse(s)
	if err != nil {
		return nil, err
	}
	times := make([]time.Time, 0, 5)
	for range 5 {
		after, err = p.Next(after)
		if err != nil {
			return nil, err
		}
		times = append(times, after)
	}
	return times, nil
}

func Parse(s model.Schedule) (Plan, error) {
	p := Plan{}
	s, err := Normalize(s)
	if err != nil {
		return p, err
	}
	if s.Kind == "interval" {
		d, err := time.ParseDuration(s.Expression)
		if err != nil || d < time.Second || d > 365*24*time.Hour {
			return p, fmt.Errorf("固定间隔须为 1s 到 8760h")
		}
		p.interval = d
		return p, nil
	}
	zone := s.Timezone
	if zone == "" {
		zone = "Asia/Shanghai"
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return p, fmt.Errorf("无效时区: %s", zone)
	}
	p.location = loc
	parts := strings.Fields(s.Expression)
	if len(parts) != 5 {
		return p, fmt.Errorf("Cron 需要五段：分 时 日 月 周")
	}
	limits := [5][2]int{{0, 59}, {0, 23}, {1, 31}, {1, 12}, {0, 7}}
	for i, part := range parts {
		p.fields[i], err = parseField(part, limits[i][0], limits[i][1])
		if err != nil {
			return p, fmt.Errorf("Cron 第 %d 段: %w", i+1, err)
		}
	}
	p.wildDay = strings.HasPrefix(parts[2], "*")
	p.wildWeek = strings.HasPrefix(parts[4], "*")
	if p.fields[4][7] {
		p.fields[4][0] = true
	}
	return p, nil
}

func parseField(raw string, min, max int) (map[int]bool, error) {
	values := make(map[int]bool)
	for _, item := range strings.Split(raw, ",") {
		step, base := 1, item
		hasStep := strings.Contains(item, "/")
		if hasStep {
			pair := strings.Split(item, "/")
			if len(pair) != 2 {
				return nil, fmt.Errorf("无效步长")
			}
			var err error
			step, err = strconv.Atoi(pair[1])
			if err != nil || step < 1 || step > max-min+1 {
				return nil, fmt.Errorf("无效步长")
			}
			base = pair[0]
		}
		lo, hi := min, max
		if base != "*" {
			pair := strings.Split(base, "-")
			var err error
			lo, err = strconv.Atoi(pair[0])
			if err != nil || len(pair) > 2 {
				return nil, fmt.Errorf("无效数字或范围")
			}
			hi = lo
			if len(pair) == 2 {
				hi, err = strconv.Atoi(pair[1])
			} else if hasStep {
				hi = max
			}
			if err != nil || lo < min || hi > max || lo > hi {
				return nil, fmt.Errorf("范围应在 %d 到 %d", min, max)
			}
		}
		for n := lo; n <= hi; n += step {
			values[n] = true
		}
	}
	return values, nil
}

func (p Plan) Next(after time.Time) (time.Time, error) {
	if p.interval > 0 {
		return after.Add(p.interval), nil
	}
	t := after.Truncate(time.Minute).Add(time.Minute)
	limit := t.AddDate(5, 0, 0)
	for ; t.Before(limit); t = t.Add(time.Minute) {
		local := t.In(p.location)
		if !p.fields[3][int(local.Month())] {
			continue
		}
		day := p.fields[2][local.Day()]
		week := p.fields[4][int(local.Weekday())]
		matchDay := day && week
		if !p.wildDay && !p.wildWeek {
			matchDay = day || week
		}
		if matchDay && p.fields[0][local.Minute()] &&
			p.fields[1][local.Hour()] {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("该 Cron 在未来五年没有执行时间")
}
