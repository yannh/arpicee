package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	awsLambda "github.com/aws/aws-sdk-go/service/lambda"
	awsSSM "github.com/aws/aws-sdk-go/service/ssm"
	"github.com/google/go-github/v50/github"
	"github.com/yannh/arpicee/pkg/arpicee"
	"github.com/yannh/arpicee/pkg/githubrpc"
	"github.com/yannh/arpicee/pkg/lambdarpc"
	"github.com/yannh/arpicee/pkg/slackbot"
	"github.com/yannh/arpicee/pkg/ssmrpc"
	"golang.org/x/oauth2"
)

type lambdaDiscovery struct {
	Region    string
	TagFilter map[string]string
}
type ssmDiscovery struct {
	Region    string
	TagFilter map[string]string
}
type githubDiscovery struct {
	Repo     string
	Workflow string
}

type config struct {
	Lambda []lambdaDiscovery
	Ssm    []ssmDiscovery
	Github []githubDiscovery
}

func realMain() error {
	var appToken, botToken string

	if appToken = os.Getenv("SLACK_APP_TOKEN"); appToken == "" {
		return fmt.Errorf("SLACK_APP_TOKEN must be set.")
	}

	if !strings.HasPrefix(appToken, "xapp-") {
		return fmt.Errorf("SLACK_APP_TOKEN must have the prefix \"xapp-\".")
	}

	if botToken = os.Getenv("SLACK_BOT_TOKEN"); botToken == "" {
		return fmt.Errorf("SLACK_BOT_TOKEN must be set.")
	}

	if !strings.HasPrefix(botToken, "xoxb-") {
		return fmt.Errorf("SLACK_BOT_TOKEN must have the prefix \"xoxb-\".")
	}

	s, err := slackbot.New(appToken, botToken)
	if err != nil {
		return fmt.Errorf("failed initialising Slackbot: %w", err)
	}

	cfgFileName := "config.json"
	f, err := os.ReadFile(cfgFileName)
	if err != nil {
		log.Fatalf("failed reading from config file %s: %s", cfgFileName, err)
	}
	var c config
	if err = json.Unmarshal(f, &c); err != nil {
		log.Fatalf("failed parsing config file %s: %s", cfgFileName, err)
	}

	if len(c.Lambda) > 0 {
		region, found := os.LookupEnv("AWS_REGION")
		if !found {
			region = "us-east-1"
			// warn for environments where lambdas might not be deployed in this region
			log.Printf("WARN: AWS_REGION env var not found; using %s as default\n", region)
		}

		awsProfile, found := os.LookupEnv("AWS_PROFILE")
		if !found {
			log.Println("WARN: AWS_PROFILE env var not found; invoking lambda might not work as expected")
		}

		sess, err := session.NewSessionWithOptions(session.Options{
			Config: aws.Config{
				Region: aws.String(region),
			},
			SharedConfigState: session.SharedConfigEnable,
			Profile:           awsProfile,
		})
		if err != nil {
			log.Fatalf("error creating AWS session: %s\n", err)
		}
		lambdaSvc := awsLambda.New(sess)
		for _, d := range c.Lambda {
			var filters []func(configuration *awsLambda.ListTagsOutput) bool
			for k, v := range d.TagFilter {
				filters = append(filters, lambdarpc.TagFilter(k, v))
			}
			s.AddDiscoveryFunction(func() ([]arpicee.RemoteCall, error) {
				r, err := lambdarpc.Discover(lambdaSvc, filters)
				var rpcs []arpicee.RemoteCall
				for _, ra := range r {
					rpcs = append(rpcs, ra)
				}
				return rpcs, err
			})
		}
	}

	if len(c.Ssm) > 0 {
		region, found := os.LookupEnv("AWS_REGION")
		if !found {
			region = "us-east-1"
			// warn for environments where lambdas might not be deployed in this region
			log.Printf("WARN: AWS_REGION env var not found; using %s as default\n", region)
		}

		awsProfile, found := os.LookupEnv("AWS_PROFILE")
		if !found {
			log.Println("WARN: AWS_PROFILE env var not found; invoking lambda might not work as expected")
		}

		sess, err := session.NewSessionWithOptions(session.Options{
			Config: aws.Config{
				Region: aws.String(region),
			},
			SharedConfigState: session.SharedConfigEnable,
			Profile:           awsProfile,
		})
		if err != nil {
			log.Fatalf("error creating AWS session: %s\n", err)
		}
		ssmSvc := awsSSM.New(sess)
		for _, ssm := range c.Ssm {
			var filters []func(configuration *awsSSM.DocumentIdentifier) bool
			for k, v := range ssm.TagFilter {
				filters = append(filters, ssmrpc.TagFilter(k, v))
			}
			s.AddDiscoveryFunction(func() ([]arpicee.RemoteCall, error) {
				r, err := ssmrpc.Discover(ssmSvc, filters)
				var rpcs []arpicee.RemoteCall
				for _, ra := range r {
					rpcs = append(rpcs, ra)
				}
				return rpcs, err
			})
		}
	}
	if len(c.Github) > 0 {
		githubToken := os.Getenv("GITHUB_TOKEN")
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: githubToken},
		)
		gc := github.NewClient(oauth2.NewClient(ctx, ts))
		for _, g := range c.Github {
			s.AddDiscoveryFunction(func() ([]arpicee.RemoteCall, error) {
				b := strings.Split(g.Repo, "/")
				owner, repo := b[0], b[1]
				r, err := githubrpc.Discover(ctx, gc, owner, repo)
				var rpcs []arpicee.RemoteCall
				for _, ra := range r {
					rpcs = append(rpcs, ra)
				}
				return rpcs, err
			})
		}
	}

	return s.Run()
}

func main() {
	if err := realMain(); err != nil {
		log.Fatalf("%s\n", err)
	}
}
