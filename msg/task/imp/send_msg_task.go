package imp

import (
	"fmt"
	"github.com/caicaing/projects/msg/api"
	"time"
)

type SendMessageTask struct {
	id   string
	name string
}

func NewSendMessageTask() *SendMessageTask {
	return &SendMessageTask{
		id:   fmt.Sprintf("发送消息---[%+v]", time.Now().Unix()),
		name: "发送消息",
	}
}
func (s *SendMessageTask) Execute() error {
	html := "<h1>H1标题</h1><br/><p style=\"color:red;\">欢迎你使用WxPusher，推荐使用HTML发送</p>"
	api.Send(html)
	return nil
}

func (s *SendMessageTask) GetID() string {
	return s.id
}

func (s *SendMessageTask) GetName() string {
	return s.name
}
