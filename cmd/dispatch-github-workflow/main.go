package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/go-github/v50/github"
	"github.com/yannh/arpicee/pkg/arpicee"
	"github.com/yannh/arpicee/pkg/githubrpc"
	"golang.org/x/oauth2"
)

type WorkflowInput struct {
	Description string
	Required    bool
}

type WorkflowTriggers struct {
	Inputs map[string]WorkflowInput
}

type Workflow struct {
	On map[string]WorkflowTriggers
}

func realMain() error {

	githubToken := os.Getenv("GITHUB_TOKEN")

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)

	tc := oauth2.NewClient(ctx, ts)
	c := github.NewClient(tc)

	r, err := githubrpc.New(ctx, c, "yannh", "arpicee-dispatch-workflow", "hello")
	if err != nil {
		fmt.Errorf("failed initialising Github Workflow: %s", err.Error())
	}
	cliArgs, _, _ := arpicee.ArgsFromFlags(r.Params(), os.Args)
	workflowOutput, err := r.Run(cliArgs)
	if err != nil {
		fmt.Errorf("failed running Github Workflow: %s", err.Error())
	}

	output, err := arpicee.Output(workflowOutput, arpicee.OutputFormat(cliArgs))
	if err != nil {
		log.Fatalf("failed generating output: %s, %s", err, output)
	}

	fmt.Printf("%s", output)

	return nil
}

func main() {
	if err := realMain(); err != nil {
		log.Fatal(err)
	}
}
