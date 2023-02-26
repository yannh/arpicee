package githubrpc

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/google/go-github/v50/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	"github.com/yannh/arpicee/pkg/arpicee"
)

func TestNewGithubRPC(t *testing.T) {
	for _, testCase := range []struct {
		workflowData   []byte
		expectedParams []arpicee.Parameter
	}{
		{
			workflowData: []byte(`name: my_workflow
on:
  workflow_dispatch:
    inputs:
jobs:
  sayhello:
    runs-on: ubuntu-latest
    steps:
      - name: checkout
        uses: actions/checkout@v2

      - name: 'say hi'
        run: "echo \"Hello\""
`),
			expectedParams: []arpicee.Parameter{},
		},
		{
			workflowData: []byte(`name: my_workflow
on:
  workflow_dispatch:
    inputs:
      name:
        description: 'Hello who?'
        required: true
jobs:
  sayhello:
    runs-on: ubuntu-latest
    steps:
      - name: checkout
        uses: actions/checkout@v2

      - name: 'say hi'
        run: "echo \"Hello ${{ github.event.inputs.name }}\""
`),
			expectedParams: []arpicee.Parameter{
				{
					Name:        "name",
					Type:        arpicee.TypeString,
					Required:    true,
					Description: "Hello who?",
				},
			},
		},
		{
			workflowData: []byte(`name: my_workflow
on:
  workflow_dispatch:
    inputs:
      count:
        description: 'How many'
        required: true
        type: number
      benice:
        description: 'Be nice?'
        required: false
        type: boolean
jobs:
  sayhello:
    runs-on: ubuntu-latest
    steps:
      - name: checkout
        uses: actions/checkout@v2

      - name: 'say hi'
        run: "echo \"Count ${{ github.event.inputs.count }}, ${{ github.event.inputs.benice }}\""
`),
			expectedParams: []arpicee.Parameter{
				{
					Name:        "count",
					Type:        arpicee.TypeInt,
					Required:    true,
					Description: "How many",
				},
				{
					Name:        "benice",
					Type:        arpicee.TypeBool,
					Required:    false,
					Description: "Be nice?",
				},
			},
		},
		{
			// Testing defaults
			workflowData: []byte(`name: my_workflow
on:
  workflow_dispatch:
    inputs:
      name:
jobs:
  sayhello:
    runs-on: ubuntu-latest
    steps:
      - name: checkout
        uses: actions/checkout@v2

      - name: 'say hi'
        run: "echo \"Hello ${{ github.event.inputs.name }}!\""
`),
			expectedParams: []arpicee.Parameter{
				{
					Name:        "name",
					Type:        arpicee.TypeString,
					Required:    false,
					Description: "",
				},
			},
		},
	} {
		encoded := base64.StdEncoding.EncodeToString(testCase.workflowData)

		mockedHTTPClient := mock.NewMockedHTTPClient(
			mock.WithRequestMatch(
				mock.GetReposActionsWorkflowsByOwnerByRepo,
				github.Workflows{
					github.Int(1),
					[]*github.Workflow{
						{
							ID:   github.Int64(123),
							Name: github.String("my_workflow"),
							Path: github.String("my/workflow/path"),
						},
					},
				},
			),
			mock.WithRequestMatch(
				mock.GetReposContentsByOwnerByRepoByPath,
				github.RepositoryContent{
					Encoding: github.String("base64"),
					Content:  github.String(encoded),
				},
			),
		)

		c := github.NewClient(mockedHTTPClient)
		rpc, err := New(context.Background(), c, "yannh", "arpicee", "my_workflow")
		if err != nil {
			t.Errorf("failed creating new githubrpc: %s", err)
			continue
		}

		expectedName := "my_workflow"
		if rpc.Name() != expectedName {
			t.Errorf("githubrpc name is %s, expected %s", rpc.Name(), expectedName)
		}

		if len(testCase.expectedParams) != len(rpc.params) {
			t.Errorf("expected %d parameters, got %d", len(testCase.expectedParams), len(rpc.params))
		} else {
			for _, p := range testCase.expectedParams {
				found := false
				for _, q := range rpc.params {
					if p.Name != q.Name {
						continue
					}
					found = true
					if p.Description != q.Description {
						t.Errorf("expected parameter %s description to be %s, is %s", p.Name, p.Description, q.Description)
					}
					if p.Type != q.Type {
						t.Errorf("expected parameter %s type to be %d, is %d", p.Name, p.Type, q.Type)
					}
					if p.Required != q.Required {
						t.Errorf("expected parameter %s required to be %t, is %t", p.Name, p.Required, q.Required)
					}
				}
				if found == false {
					t.Errorf("expected to find a parameter named %s", p.Name)
				}
			}
		}
	}
}
