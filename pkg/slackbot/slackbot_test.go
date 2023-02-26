package slackbot

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/slack-go/slack"
	"github.com/yannh/arpicee/pkg/arpicee"
	"github.com/yannh/arpicee/pkg/githubrpc"
)

func TestReloadRPCs(t *testing.T) {
	mockDiscoverNil := func() ([]arpicee.RemoteCall, error) {
		return nil, nil
	}
	mockDiscoverOne := func() ([]arpicee.RemoteCall, error) {
		return []arpicee.RemoteCall{&githubrpc.GithubRPC{}}, nil
	}
	mockDiscoverTwo := func() ([]arpicee.RemoteCall, error) {
		return []arpicee.RemoteCall{&githubrpc.GithubRPC{}, &githubrpc.GithubRPC{}}, nil
	}

	for i, testCase := range []struct {
		discoverFuncs []func() ([]arpicee.RemoteCall, error)
		expectRPCs    []arpicee.RemoteCall
		expectErr     error
	}{
		{
			[]func() ([]arpicee.RemoteCall, error){mockDiscoverNil},
			nil,
			nil,
		},
		{
			[]func() ([]arpicee.RemoteCall, error){mockDiscoverNil, mockDiscoverTwo, mockDiscoverOne},
			[]arpicee.RemoteCall{&githubrpc.GithubRPC{}, &githubrpc.GithubRPC{}, &githubrpc.GithubRPC{}},
			nil,
		},
	} {
		b := Slackbot{
			discoverFuncs: testCase.discoverFuncs,
		}
		err := b.reloadRPCs()
		if (err == nil && testCase.expectErr != nil) || (err != nil && testCase.expectErr == nil) {
			t.Errorf("test %d, expected err to be %s, was %s", i, testCase.expectErr, err)
		} else if err != nil && err.Error() != testCase.expectErr.Error() {
			t.Errorf("test %d, expected err to be %s, was %s", i, testCase.expectErr, err)
		}

		for i, _ := range testCase.expectRPCs {
			if !reflect.DeepEqual(b.rpcs[i], testCase.expectRPCs[i]) {
				t.Errorf("test %d, expected rpcs to be %s, was %s", i, testCase.expectRPCs, b.rpcs)
			}
		}
	}
}

func TestArgsFromView(t *testing.T) {
	for testN, testCase := range []struct {
		viewStateJSON string
		params        []arpicee.Parameter
		expectedArgs  []arpicee.Argument
		expectedErr   error
	}{
		{
			// These are dumped from the debug logs, on form submissions
			viewStateJSON: `{
				"values": {
					"name": {
						"name": {
							"type": "plain_text_input",
							"value": "foo"
						}
					},
					"dryrun": {
						"dryrun": {
							"type": "checkboxes",
							"selected_options": [
						{
							"text": {
							"type": "plain_text",
							"text": "Use -dryrun=false to actually purge. Default is true: do not purge assets.",
							"emoji": true
						},
							"value": "dryrun"
						}
						]
						}
					}
				}
			}`,
			params: []arpicee.Parameter{
				{
					Name:        "name",
					Type:        arpicee.TypeString,
					Description: "lorem ipsum",
					Required:    true,
				},
				{
					Name:        "dryrun",
					Type:        arpicee.TypeBool,
					Description: "lorem ipsum",
					Required:    true,
				},
			},
			expectedArgs: []arpicee.Argument{
				&arpicee.ArgumentString{
					Name: "name",
					Val:  "foo",
				},
				&arpicee.ArgumentBool{
					Name: "dryrun",
					Val:  true,
				},
			},
			expectedErr: nil,
		},
		{
			viewStateJSON: `{
        "values": {
          "name": {
            "name": {
              "type": "plain_text_input",
              "value": "foo"
            }
          },
          "dryrun": {
            "dryrun": {
              "type": "checkboxes",
              "selected_options": []
            }
          }
        }
    }`,
			params: []arpicee.Parameter{
				{
					Name:        "name",
					Type:        arpicee.TypeString,
					Description: "lorem ipsum",
					Required:    true,
				},
				{
					Name:        "dryrun",
					Type:        arpicee.TypeBool,
					Description: "lorem ipsum",
					Required:    true,
				},
			},
			expectedArgs: []arpicee.Argument{
				&arpicee.ArgumentString{
					Name: "name",
					Val:  "foo",
				},
				&arpicee.ArgumentBool{
					Name: "dryrun",
					Val:  false,
				},
			},
			expectedErr: nil,
		},
	} {
		var viewState slack.ViewState
		err := json.Unmarshal([]byte(testCase.viewStateJSON), &viewState)
		if err != nil {
			t.Errorf("failed setting up test: %s", err.Error())
		}

		args, err := argsFromView(testCase.params, &viewState)
		if err != nil {
			t.Errorf("failed getting args from view: %s", args)
		}

		if len(testCase.expectedArgs) != len(args) {
			t.Errorf("test %d - expected %d arguments, got %d", testN, len(testCase.expectedArgs), len(args))
		}
		for _, i := range testCase.expectedArgs {
			found := false
			for _, j := range args {
				if reflect.DeepEqual(i, j) {
					found = true
					break
				}
			}
			if found == false {
				t.Errorf("test %d - expected to find argument %+v", testN, i)
			}
		}
	}
}
