package slackbot

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"github.com/yannh/arpicee/pkg/arpicee"
	"github.com/yannh/arpicee/pkg/views"
)

type Slackbot struct {
	slackClient   *slack.Client
	socketClient  *socketmode.Client
	rpcs          []arpicee.RemoteCall
	discoverFuncs []func() ([]arpicee.RemoteCall, error)
}

func (s *Slackbot) AddDiscoveryFunction(f func() ([]arpicee.RemoteCall, error)) {
	s.discoverFuncs = append(s.discoverFuncs, f)
}

func (s *Slackbot) rpcByName(name string) arpicee.RemoteCall {
	for _, r := range s.rpcs {
		if r.Name() == name {
			return r
		}
	}
	return nil
}

func (s *Slackbot) reloadRPCs() error {
	rpcs := []arpicee.RemoteCall{}
	var wg sync.WaitGroup
	var errwg sync.WaitGroup
	var mu sync.Mutex

	errsChan := make(chan error)
	var resErr error = nil
	errwg.Add(1)
	go func() {
		for e := range errsChan {
			if resErr == nil {
				resErr = e
			} else {
				resErr = fmt.Errorf("%s, %s", resErr.Error(), e.Error())
			}
		}
		errwg.Done()
	}()

	for _, f := range s.discoverFuncs {
		wg.Add(1)
		go func(e chan<- error, df func() ([]arpicee.RemoteCall, error)) {
			r, err := df()
			if err != nil {
				e <- err
			}
			mu.Lock()
			rpcs = append(rpcs, r...)
			mu.Unlock()
			wg.Done()
		}(errsChan, f)
	}
	wg.Wait() // All discovery is complete
	close(errsChan)
	errwg.Wait() // All error reading is done

	s.rpcs = rpcs
	return resErr
}

func New(appToken, botToken string) (*Slackbot, error) {
	slackClient := slack.New(
		botToken,
		slack.OptionDebug(true),
		slack.OptionLog(log.New(os.Stdout, "api:  ", log.Lshortfile|log.LstdFlags)),
		slack.OptionAppLevelToken(appToken),
	)

	socketClient := socketmode.New(
		slackClient,
		socketmode.OptionDebug(true),
		socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)),
	)

	return &Slackbot{
		slackClient:  slackClient,
		socketClient: socketClient,
	}, nil
}

func argsFromView(params []arpicee.Parameter, state *slack.ViewState) ([]arpicee.Argument, error) {
	args := []arpicee.Argument{}
	for key, val := range state.Values {
		v := val[key].Value

		if val[key].Type == "checkboxes" {
			if len(val[key].SelectedOptions) > 0 {
				v = "true"
			} else {
				v = "false"
			}
		}

		for _, param := range params {
			if param.Name == key {
				switch param.Type {
				case arpicee.TypeString:
					if param.Required && v == "" {
						return nil, fmt.Errorf("parameter %s is required", param.Name)
					}
					args = append(args, &arpicee.ArgumentString{
						Name: key,
						Val:  v,
					})
				case arpicee.TypeInt:
					i, err := strconv.ParseInt(v, 10, 32)
					if err != nil {
						return nil, fmt.Errorf("parameter %s should be a number, could not parse given value: %s", param.Name, v)
					}
					args = append(args, &arpicee.ArgumentInt{
						Name: key,
						Val:  int(i),
					})
				case arpicee.TypeBool:
					b := v == "true"
					args = append(args, &arpicee.ArgumentBool{
						Name: key,
						Val:  b,
					})
				}
			}
		}
	}

	return args, nil
}

func getSlackIDFromCallback(externalID string) string {
	channelID := ""
	if externalID != "" {
		strArr := strings.Split(externalID, "_")
		channelID = strArr[0]
	}

	return channelID
}

func (sb *Slackbot) Run() error {
	if err := sb.reloadRPCs(); err != nil {
		// Exit early if we fail loading RPCs when starting up
		return fmt.Errorf("failed loading RPCs: %w", err)
	}

	go func() {
		for evt := range sb.socketClient.Events {
			switch evt.Type {
			case socketmode.EventTypeConnecting:
				log.Printf("Connecting to Slack with Socket Mode...")
			case socketmode.EventTypeConnectionError:
				log.Println("Connection failed. Retrying later...")
			case socketmode.EventTypeConnected:
				log.Println("Connected to Slack with Socket Mode.")
			case socketmode.EventTypeEventsAPI:
				eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
				if !ok {
					continue
				}
				sb.socketClient.Ack(*evt.Request)

				if eventsAPIEvent.Type == slackevents.CallbackEvent {
					switch ev := eventsAPIEvent.InnerEvent.Data.(type) {
					case *slackevents.AppMentionEvent:
						if _, _, err := sb.socketClient.PostMessage(ev.Channel, slack.MsgOptionText("hello!", false)); err != nil {
							log.Printf("failed posting message: %v", err)
						}
					case *slackevents.AppHomeOpenedEvent:
						res, err := sb.socketClient.PublishView(ev.User, *views.AppHome(sb.rpcs), "")
						if err != nil {
							log.Printf("failed posting message: %s %+v", err, res)
						}
					default:
					}
				}
			case socketmode.EventTypeInteractive:
				callback, ok := evt.Data.(slack.InteractionCallback)
				if !ok {
					log.Printf("Ignored %+v\n", evt)
					continue
				}

				switch callback.Type {
				case slack.InteractionTypeBlockActions:
					for _, action := range callback.ActionCallback.BlockActions {
						switch action.ActionID {
						// An RPC has been selected
						case views.SelectRPCActionID:
							// Open the invocation dialog for the selected RPC
							rpc := sb.rpcByName(action.SelectedOption.Text.Text)
							view := views.RunRPCDialog(callback.Channel.ID, rpc)
							v, err := sb.socketClient.OpenView(callback.TriggerID, view)
							if err != nil {
								log.Printf("Failed opening RPC view: %s, %+v", err, v)
								continue
							}

							// Delete the "Select RPC" view
							sb.socketClient.PostEphemeral(
								callback.Channel.ID,
								callback.User.ID,
								slack.MsgOptionText("", false),
								slack.MsgOptionReplaceOriginal(callback.ResponseURL),
								slack.MsgOptionDeleteOriginal(callback.ResponseURL),
							)

						default:
							log.Printf("unknown actionid %s", action.ActionID)
						}
					}

				case slack.InteractionTypeShortcut:
				case slack.InteractionTypeViewSubmission:
					if callback.View.CallbackID == views.RunRPCDialogCallbackID {
						if callback.View.CallbackID == "" {
							log.Printf("could not find ChannelID from view callback for view %s\n", callback.View.CallbackID)
							continue
						}

						rpc := sb.rpcByName(callback.View.PrivateMetadata)
						channelID := getSlackIDFromCallback(callback.View.ExternalID)

						go func() {
							args, err := argsFromView(rpc.Params(), callback.View.State)
							if err != nil {
								log.Printf("failed parsing args from view: %s", err)
								return
							}

							msg := fmt.Sprintf("RPC *%s* invoked by <@%s>, currently running...", rpc.Name(), callback.User.ID)
							_, ts, err := sb.socketClient.PostMessage(
								channelID,
								slack.MsgOptionAttachments(slack.Attachment{
									Pretext: msg,
								}),
								slack.MsgOptionAsUser(true),
							)
							if err != nil {
								log.Printf("RPC %s invoked by %s - error posting message to Slack: %s, not invoking RPC", rpc.Name(), callback.User.Name, err)
								return
							}

							rpcres, err := rpc.Run(args)
							if err != nil {
								log.Printf("failed invoking RPC %s: %s", rpc.Name(), err)
								return
							}
							payload, _ := views.RPCResult(rpc, callback.User, rpcres, err)
							_, _, _, err = sb.socketClient.UpdateMessage(
								channelID,
								ts,
								slack.MsgOptionAttachments(*payload),
								slack.MsgOptionAsUser(true),
							)
							if err != nil {
								log.Printf("RPC %s invoked by %s: error posting results to Slack: %s\n", rpc.Name(), callback.User.Name, err)
							}
						}()
					}
				}
				var payload interface{}
				sb.socketClient.Ack(*evt.Request, payload)

			case socketmode.EventTypeSlashCommand:
				cmd, _ := evt.Data.(slack.SlashCommand)
				sb.socketClient.Debugf("Slash command received: %+v", cmd)
				sb.socketClient.Ack(*evt.Request, map[string]interface{}{
					"blocks": views.SelectRPCDialog(sb.rpcs),
				})
			default:
				log.Printf("Unexpected event type received: %s\n", evt.Type)
			}
		}
	}()

	return sb.socketClient.Run()
}
