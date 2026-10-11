package reminder

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"git.woa.com/mooyliu/mooy_test/framework"
)

const defaultDrinkWaterMessage = "该喝水啦，休息一下，喝杯水吧。"

// DrinkWater generates reminder content independently of notification types.
type DrinkWater struct{}

// Register adds the example task without creating a scheduled reminder.
func Register(r *taskflow.Registry) error {
	return r.Tasks.Register(func() taskflow.TaskHandler {
		return DrinkWater{}
	})
}

type drinkWaterConfig struct {
	Message string `json:"message,omitempty"`
}

func (DrinkWater) Spec() taskflow.Spec {
	return taskflow.Spec{
		Type: "drink_water", Name: "喝水提醒",
		Description: "生成喝水提醒；请选择每次执行完成通知，" +
			"并绑定 Bark 或其他通知渠道。message 可自定义，留空使用默认文案。",
		Example: json.RawMessage(`{"message":"` +
			defaultDrinkWaterMessage + `"}`),
	}
}

func (DrinkWater) Validate(raw json.RawMessage) error {
	_, err := decodeDrinkWaterConfig(raw)
	return err
}

func decodeDrinkWaterConfig(raw json.RawMessage) (drinkWaterConfig, error) {
	var c drinkWaterConfig
	if err := taskflow.DecodeConfig(raw, &c); err != nil {
		return c, err
	}
	if c.Message == "" {
		c.Message = defaultDrinkWaterMessage
	}
	if strings.TrimSpace(c.Message) == "" {
		return c, fmt.Errorf("message 不能只包含空白字符")
	}
	return c, nil
}

func (DrinkWater) Run(ctx context.Context,
	raw json.RawMessage) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	c, err := decodeDrinkWaterConfig(raw)
	if err != nil {
		return "", err
	}
	return c.Message, nil
}
