package schedule

import (
	"fmt"

	"git.woa.com/mooyliu/mooy_test/framework/internal/model"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Normalize converts calendar selections to a scheduler expression.
// Structured fields are authoritative; legacy cron/interval remain supported.
func Normalize(s model.Schedule) (model.Schedule, error) {
	s.Expression = strings.TrimSpace(s.Expression)
	if s.Timezone == "" {
		s.Timezone = "Asia/Shanghai"
	}
	if s.Kind == "cron" {
		return s, nil
	}
	if s.Kind == "interval" {
		return normalizeInterval(s)
	}
	if s.Kind == "hourly" {
		if s.Minute < 0 || s.Minute > 59 {
			return s, fmt.Errorf("执行分钟须介于 0 和 59")
		}
		s.Expression = fmt.Sprintf("%d * * * *", s.Minute)
		return s, nil
	}
	if s.Kind != "daily" && s.Kind != "weekly" && s.Kind != "monthly" {
		return s, fmt.Errorf("不支持的执行周期: %s", s.Kind)
	}
	clock, err := time.Parse("15:04", s.Time)
	if err != nil || len(s.Time) != 5 {
		return s, fmt.Errorf("执行时间须为 HH:mm 格式")
	}
	day, weekday := "*", "*"
	switch s.Kind {
	case "daily":
	case "weekly":
		s.Weekdays, weekday, err = normalizeWeekdays(s.Weekdays)
		if err != nil {
			return s, err
		}
	case "monthly":
		if s.Day < 1 || s.Day > 31 {
			return s, fmt.Errorf("每月执行日须介于 1 和 31")
		}
		day = strconv.Itoa(s.Day)
	default:
		return s, fmt.Errorf("不支持的执行周期: %s", s.Kind)
	}
	s.Expression = fmt.Sprintf("%d %d %s * %s", clock.Minute(),
		clock.Hour(), day, weekday)
	return s, nil
}

func normalizeWeekdays(days []int) ([]int, string, error) {
	if len(days) == 0 || len(days) > 7 {
		return nil, "", fmt.Errorf("请至少选择一个执行星期")
	}
	days = append([]int{}, days...)
	sort.Ints(days)
	parts := make([]string, len(days))
	for i, day := range days {
		if day < 0 || day > 6 || (i > 0 && day == days[i-1]) {
			return nil, "", fmt.Errorf("星期须为 0 到 6 且不能重复")
		}
		parts[i] = strconv.Itoa(day)
	}
	return days, strings.Join(parts, ","), nil
}

func normalizeInterval(s model.Schedule) (model.Schedule, error) {
	if s.Every == 0 && s.Unit == "" {
		return s, nil
	}
	units := map[string]time.Duration{
		"s": time.Second, "m": time.Minute,
		"h": time.Hour, "d": 24 * time.Hour,
	}
	unit, ok := units[s.Unit]
	limit := 365 * 24 * time.Hour
	if !ok || s.Every < 1 || s.Every > int(limit/unit) {
		return s, fmt.Errorf("间隔须介于 1 秒和 365 天，单位为 s/m/h/d")
	}
	s.Expression = (time.Duration(s.Every) * unit).String()
	return s, nil
}
