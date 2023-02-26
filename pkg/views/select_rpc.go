package views

import (
	"github.com/slack-go/slack"
	"github.com/yannh/arpicee/pkg/arpicee"
)

const (
	SelectRPCActionID = "select_rpc"
)

func SelectRPCDialog(rpcs []arpicee.RemoteCall) []slack.Block {
	var opts []*slack.OptionBlockObject
	for _, rpc := range rpcs {
		opts = append(opts, &slack.OptionBlockObject{
			Text: &slack.TextBlockObject{
				Type: "plain_text",
				Text: rpc.Name(),
			},
			Value: rpc.Name(),
		})
	}

	return []slack.Block{
		slack.NewSectionBlock(
			&slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: "Select an automation",
			},
			nil,
			&slack.Accessory{SelectElement: &slack.SelectBlockElement{
				Type:     "static_select",
				ActionID: SelectRPCActionID,
				Options:  opts,
			}},
		),
	}
}
