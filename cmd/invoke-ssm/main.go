package main

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/yannh/arpicee/pkg/arpicee"
	"github.com/yannh/arpicee/pkg/ssmrpc"
)

type ssmDocParameter struct {
	Type string
}

type ssmDocContent struct {
	Description string
	Parameters  map[string]ssmDocParameter
}

func realMain() error {
	docName := path.Base(os.Args[0])
	if fnName := os.Getenv("FN_NAME"); fnName != "" {
		docName = fnName
	}

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

	s := ssm.New(sess)
	doc, err := ssmrpc.New(s, docName)
	if err != nil {
		log.Fatalf("error creating aws session: %s\n", err)
	}

	cliArgs, _, err := arpicee.ArgsFromFlags(doc.Params(), os.Args)
	ssmOutput, err := doc.Run(cliArgs)
	if err != nil {
		log.Fatalf("error running ssm automation %s", err)
	}

	output, err := arpicee.Output(ssmOutput, arpicee.OutputFormat(cliArgs))
	if err != nil {
		log.Fatalf("failed generating output: %s", err)
	}

	fmt.Printf("%s", output)
	return nil
}

func main() {
	if err := realMain(); err != nil {
		log.Fatal(err)
	}
}
