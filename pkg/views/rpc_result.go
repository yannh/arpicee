package views

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/slack-go/slack"
	"github.com/yannh/arpicee/pkg/arpicee"
)

func RPCResult(rpc arpicee.RemoteCall, user slack.User, rpcOutput map[string]interface{}, err error) (*slack.Attachment, error) {
	if rpcOutput["formatString"] == "" {
		rpcOutput["formatString"] = "the remote procedure you are invoking is missing a formatString property in the return value"
	}

	var attachmentFields []slack.AttachmentField

	color := "00CB53"
	if err != nil {
		color = "e5345e"

		attachmentFields = append(
			attachmentFields,
			slack.AttachmentField{
				Title: "Error Message",
				Value: err.Error(),
			},
		)
	} else {
		t := template.Must(template.New("").Parse(fmt.Sprintf("%s", rpcOutput["formatString"])))
		b := bytes.Buffer{}
		if err := t.Execute(&b, rpcOutput); err != nil {
			return nil, err
		}
		output := b.String()

		attachmentFields = append(
			attachmentFields,
			slack.AttachmentField{
				Title: "Result",
				Value: fmt.Sprintf("%s", output),
			},
		)
	}

	return &slack.Attachment{
		Pretext: fmt.Sprintf("Remote procedure *%s* invoked by <@%s>", rpc.Name(), user.ID),
		Color:   color,
		Fields:  attachmentFields,
	}, nil
}
