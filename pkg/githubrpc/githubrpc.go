package githubrpc

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v50/github"
	"github.com/yannh/arpicee/pkg/arpicee"
	"gopkg.in/yaml.v3"
)

type WorkflowInput struct {
	Description string
	Required    bool
	InputType   string `yaml:"type"`
}

type WorkflowTriggers struct {
	Inputs map[string]WorkflowInput
}

type Workflow struct {
	On map[string]WorkflowTriggers
}

type GithubRPC struct {
	c           *github.Client
	ctx         context.Context
	owner       string
	repo        string
	name        string
	id          int64
	description string
	params      []arpicee.Parameter
}

func (gr *GithubRPC) Name() string {
	return gr.name
}

func (gr *GithubRPC) Description() string {
	return gr.description
}

func (gr *GithubRPC) Params() []arpicee.Parameter {
	return gr.params
}

func New(ctx context.Context, c *github.Client, owner string, repo string, workflowName string) (*GithubRPC, error) {
	ws, _, err := c.Actions.ListWorkflows(ctx, owner, repo, nil)
	if err != nil {
		return nil, err
	}

	var w *github.Workflow
	for _, iw := range ws.Workflows {
		if *iw.Name == workflowName {
			w = iw
		}
	}
	if w == nil {
		return nil, fmt.Errorf("failed finding workflow %s in %s", workflowName, repo)
	}

	workflowContent, _, _, err := c.Repositories.GetContents(ctx, owner, repo, *w.Path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed getting workflow file: %w", err)
	}
	var workflowContentBytes []byte
	switch *workflowContent.Encoding {
	case "base64":
		workflowContentBytes, err = base64.StdEncoding.DecodeString(*workflowContent.Content)
		if err != nil {
			return nil, fmt.Errorf("failed decoding content from file %s: %w", *w.Path, err)
		}
	}

	var workflow Workflow
	if err := yaml.Unmarshal(workflowContentBytes, &workflow); err != nil {
		return nil, fmt.Errorf("failed unmarshalling workflow file %s: %s", *w.Path, err)
	}

	if _, ok := workflow.On["workflow_dispatch"]; !ok {
		return nil, fmt.Errorf("workflow %+v contained in file %s does not contain a workflow_dispatch section", workflow, *w.Path)
	}

	var params []arpicee.Parameter
	for pName, p := range workflow.On["workflow_dispatch"].Inputs {
		// boolean, number of string
		// https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#onworkflow_callinputsinput_idtype
		pType := arpicee.TypeString
		switch strings.ToLower(p.InputType) {
		case "boolean":
			pType = arpicee.TypeBool
		case "number":
			pType = arpicee.TypeInt
		case "string":
		default:
		}
		params = append(params, arpicee.Parameter{
			Name:        pName,
			Type:        pType,
			Description: p.Description,
			Required:    p.Required,
		})
	}

	return &GithubRPC{
		c:           c,
		ctx:         ctx,
		owner:       owner,
		repo:        repo,
		name:        workflowName,
		id:          *w.ID,
		description: "",
		params:      params,
	}, nil
}

func Discover(ctx context.Context, c *github.Client, owner string, repo string) ([]*GithubRPC, error) {
	ws, _, err := c.Actions.ListWorkflows(ctx, owner, repo, nil)
	if err != nil {
		return nil, err
	}

	rpcs := []*GithubRPC{}
	for _, iw := range ws.Workflows {
		rpc, err := New(ctx, c, owner, repo, *iw.Name)
		if err != nil {
			return nil, fmt.Errorf("failed getting workflow %s", *iw.Name)
		}
		rpcs = append(rpcs, rpc)
	}
	return rpcs, nil
}

func output(wfName string, jobs []*github.WorkflowJob) map[string]interface{} {
	o := map[string]interface{}{}
	for _, wj := range jobs {
		o[""+*wj.Name] = *wj.Status
	}

	formatString := fmt.Sprintf("Workflow %s:\n", wfName)
	for k, v := range o {
		switch v {
		case "completed":
			formatString += "✓ " + k + "\n"
		default:
			formatString += fmt.Sprintf("%s: %s\n", k, v)
		}
	}
	o["formatString"] = formatString
	return o
}

func (gr *GithubRPC) Run(args []arpicee.Argument) (map[string]interface{}, error) {
	var payload github.CreateWorkflowDispatchEventRequest
	payload.Ref = "main"
	payload.Inputs = map[string]interface{}{}
	for _, arg := range args {
		switch a := arg.(type) {
		case *arpicee.ArgumentString:
			payload.Inputs[a.Name] = a.Val
		case *arpicee.ArgumentInt:
			payload.Inputs[a.Name] = a.Val
		}
	}
	t := time.Now()
	p := t.Format(time.RFC3339)
	w, _, err := gr.c.Actions.ListWorkflowRunsByID(gr.ctx, gr.owner, gr.repo, gr.id, &github.ListWorkflowRunsOptions{
		Event:   "workflow_dispatch",
		Created: fmt.Sprintf(">%s", p),
	})
	if err != nil {
		return nil, fmt.Errorf("failed listing workflow runs before invocation: %w", err)
	}
	countBefore := len(w.WorkflowRuns)

	_, err = gr.c.Actions.CreateWorkflowDispatchEventByID(gr.ctx, gr.owner, gr.repo, gr.id, payload)
	if err != nil {
		return nil, err
	}

	max_tries := 10
	tries := 0
	// Dispatching a workflow is asynhronous - we Wait until Workflow run
	// has actually been triggered
	for countAfter := countBefore; countAfter == countBefore; tries = tries + 1 {
		time.Sleep(1 * time.Second)
		if tries > max_tries {
			return nil, fmt.Errorf("failed getting dispatch run after %d tries", tries)
		}

		w, _, err = gr.c.Actions.ListWorkflowRunsByID(gr.ctx, gr.owner, gr.repo, gr.id, &github.ListWorkflowRunsOptions{
			Event:   "workflow_dispatch",
			Created: fmt.Sprintf(">%s", p),
		})
		if err != nil {
			return nil, fmt.Errorf("failed listing workflow runs before invocation: %w", err)
		}
		countAfter = len(w.WorkflowRuns)
	}

	// The workflow has finally started, we can now get the WorkflowRun object
	// There are subtle race conditions in all this, but is the best the GH API has to offer for now
	latestWorkflowRun := w.WorkflowRuns[0]
	for _, wr := range w.WorkflowRuns {
		if wr.CreatedAt.String() > latestWorkflowRun.CreatedAt.String() {
			latestWorkflowRun = wr
		}
	}

	max_tries = 1000
	tries = 0
	var run *github.WorkflowRun
	// We wait for our Github Workflow run to complete
	for run == nil || *run.Status == "queued" || *run.Status == "in_progress" {
		tries += 1
		if tries > max_tries {
			return nil, fmt.Errorf("failed waiting for workflow to complete, timeout")
		}

		run, _, err = gr.c.Actions.GetWorkflowRunByID(gr.ctx, gr.owner, gr.repo, *latestWorkflowRun.ID)
		if err != nil {
			return nil, fmt.Errorf("failed getting workflow run: %w", err)
		}
		time.Sleep(3 * time.Second)
	}

	if *run.Status != "completed" {
		return nil, fmt.Errorf("workflow run failed: got unexpected status %s", *run.Status)
	}

	wjs, _, err := gr.c.Actions.ListWorkflowJobs(gr.ctx, gr.owner, gr.repo, *latestWorkflowRun.ID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed listing workflow jobs: %w", err)
	}

	return output(*latestWorkflowRun.Name, wjs.Jobs), nil
}
