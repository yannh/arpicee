package views

import (
	"sort"

	"github.com/slack-go/slack"
	"github.com/yannh/arpicee/pkg/arpicee"
)

func AppHome(rpcs []arpicee.RemoteCall) *slack.HomeTabViewRequest {
	view := &slack.HomeTabViewRequest{
		Type: "home",
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{
				&slack.SectionBlock{
					Type: slack.MBTSection,
					Text: &slack.TextBlockObject{
						Type: slack.MarkdownType,
						Text: "*Arpicee* is your slackbot of choice to trigger remote jobs. AWS Lambdas, Github Actions, AWS SSM automations, you name it: you can trigger all of that without leaving Slack.",
					},
				},
				&slack.ContextBlock{
					Type: slack.MBTContext,
					ContextElements: slack.ContextElements{
						Elements: []slack.MixedElement{
							// &slack.TextBlockObject{
							//	Type: "mrkdwn",
							//	Text: "Uptime: 12d, 4h, 12m",
							// },
							&slack.TextBlockObject{
								Type: slack.MarkdownType,
								Text: "<https://github.com/yannh/arpicee|github.com/yannh/arpicee>",
							},
						},
					},
				},
				&slack.DividerBlock{
					Type: slack.MBTDivider,
				},
				&slack.HeaderBlock{
					Type: slack.MBTHeader,
					Text: &slack.TextBlockObject{
						Type: slack.PlainTextType,
						Text: "Discovered jobs",
					},
				},
				&slack.DividerBlock{
					Type: slack.MBTDivider,
				},
			},
		},
	}

	sort.Slice(rpcs, func(i, j int) bool {
		return rpcs[i].Name() < rpcs[j].Name()
	})
	for _, rpc := range rpcs {
		text := "•\t" + rpc.Name()
		if rpc.Description() != "" {
			text += ": " + rpc.Description()
		}
		view.Blocks.BlockSet = append(view.Blocks.BlockSet, &slack.SectionBlock{
			Type: slack.MBTSection,
			Text: &slack.TextBlockObject{
				Type: slack.PlainTextType,
				Text: text,
			},
		})
	}

	view.Blocks.BlockSet = append(view.Blocks.BlockSet, []slack.Block{
		&slack.ActionBlock{
			Type: slack.MBTAction,
			Elements: &slack.BlockElements{
				ElementSet: []slack.BlockElement{
					slack.ButtonBlockElement{
						Type: slack.METButton,
						Text: &slack.TextBlockObject{
							Type: slack.PlainTextType,
							Text: "Reload jobs",
						},
						ActionID: "reload_jobs_id",
						Value:    "reload-jobs",
					},
				},
			},
		},
	}...)

	return view
}
