package api

import (
	"fmt"

	"github.com/wxpusher/wxpusher-sdk-go"
	"github.com/wxpusher/wxpusher-sdk-go/model"
)

func Send(content string) error {
	msg := model.NewMessage("AT_1VTzqghshnQ2oDbwelBf1MXHnfoHXCLm").
		AddUId("UID_k9pEgaN3m3QXJDyC6fJqlrY0mQw7").
		AddUId("UID_ncVIWOOxLldoi0tbKeepVyRo3rhb").
		SetContent(content)
	msgArr, err := wxpusher.SendMessage(msg)
	fmt.Println(msgArr)
	return err
}
