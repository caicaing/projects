package imp

import (
	"fmt"
	"time"
	
	"github.com/caicaing/send_msg/api"
)

// WaterReminderTask 定时提醒喝水任务
type WaterReminderTask struct {
	id   string
	name string
}

// NewWaterReminderTask 创建新的定时提醒喝水任务
func NewWaterReminderTask() *WaterReminderTask {
	return &WaterReminderTask{
		id:   fmt.Sprintf("喝水提醒---[%+v]", time.Now().Unix()),
		name: "喝水提醒",
	}
}

// Execute 执行任务
func (w *WaterReminderTask) Execute() error {
	w.run()
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			w.run()
		}
	}
}

func (w *WaterReminderTask) run() {
	// 根据时间段生成不同的问候语
	hour := time.Now().Hour()
	var greeting string
	switch {
	case hour < 12:
		greeting = "早上好"
	case hour < 18:
		greeting = "下午好"
	default:
		greeting = "晚上好"
	}

	reminderMsg := fmt.Sprintf(`
<h2>💧 %s，该喝水了！ 💧</h2>
<p>现在是 %s</p>
<p>保持充足的水分对健康非常重要，建议每天喝8杯水。</p>
<p>记得及时补充水分哦！</p>
<p>💡 小贴士：每隔一小时喝一杯水，有助于保持身体水分平衡。</p>
`, greeting, time.Now().Format("2006-01-02 15:04:05"))
	api.Send(reminderMsg)
}

// GetID 获取任务ID
func (w *WaterReminderTask) GetID() string {
	return w.id
}

// GetName 获取任务名称
func (w *WaterReminderTask) GetName() string {
	return w.name
}
