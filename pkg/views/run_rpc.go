package views

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"github.com/yannh/arpicee/pkg/arpicee"
)

const (
	RunRPCDialogCallbackID = "run_lambda_dialog"
)

func parameterTypeToBlock(param arpicee.Parameter) slack.Block {
	switch param.Type {
	case arpicee.TypeString,
		arpicee.TypeInt:
		return slack.InputBlock{
			Type:    slack.MBTInput,
			BlockID: param.Name,
			Label: &slack.TextBlockObject{
				Type: slack.PlainTextType,
				Text: param.Name,
			},
			Element: slack.PlainTextInputBlockElement{
				Type:     slack.METPlainTextInput,
				ActionID: param.Name,
				Placeholder: &slack.TextBlockObject{
					Type: slack.PlainTextType,
					Text: param.Description,
				},
			},
			Optional: !param.Required,
		}

	case arpicee.TypeBool:
		return slack.ActionBlock{
			Type:    "actions",
			BlockID: param.Name,
			Elements: &slack.BlockElements{
				ElementSet: []slack.BlockElement{
					slack.CheckboxGroupsBlockElement{
						Type:     slack.METCheckboxGroups,
						ActionID: param.Name,
						Options: []*slack.OptionBlockObject{
							{
								Text: &slack.TextBlockObject{
									Type: "plain_text",
									Text: param.Description,
								},
								Value: param.Name,
							},
						},
					},
				},
			},
		}
	}

	return nil
}

func RunRPCDialog(channelID string, rpc arpicee.RemoteCall) slack.ModalViewRequest {
	blocks := slack.Blocks{
		BlockSet: []slack.Block{},
	}

	if rpc.Description() != "" {
		blocks.BlockSet = append(blocks.BlockSet, slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.PlainTextType, rpc.Description(), false, false),
			nil,
			nil,
		))

		blocks.BlockSet = append(blocks.BlockSet, slack.DividerBlock{Type: "divider"})
	}

	params := rpc.Params()
	sort.Slice(params, func(i, j int) bool {
		return params[i].Name < params[j].Name
	})
	for _, p := range params {
		block := parameterTypeToBlock(p)
		if block != nil {
			blocks.BlockSet = append(
				blocks.BlockSet,
				block,
			)
		}
	}

	title := fmt.Sprintf("%s - %s", rpc.Name(), strings.Title(strings.ToLower("")))
	title_short := title
	if len(title) > 24 {
		title_short = title[:22] + ".." // [ERROR] must be less than 25 characters [json-pointer:/view/title/text]
	}

	return slack.ModalViewRequest{
		Type:            slack.VTModal,
		ExternalID:      strings.Join([]string{channelID, fmt.Sprintf("%d", time.Now().Nanosecond())}, "_"),
		PrivateMetadata: rpc.Name(),
		Title:           slack.NewTextBlockObject(slack.PlainTextType, title_short, false, false),
		Close:           slack.NewTextBlockObject(slack.PlainTextType, "Cancel", false, false),
		Submit:          slack.NewTextBlockObject(slack.PlainTextType, "Run", false, false),
		Blocks:          blocks,
		CallbackID:      RunRPCDialogCallbackID,
	}
}
