package main

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	awsLambda "github.com/aws/aws-sdk-go/service/lambda"
	"github.com/yannh/arpicee/pkg/arpicee"
	"github.com/yannh/arpicee/pkg/lambdarpc"
)

func realMain() error {
	progName := path.Base(os.Args[0])
	if fnName := os.Getenv("FN_NAME"); fnName != "" {
		progName = fnName
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
		return fmt.Errorf("error creating AWS session: %s\n", err)
	}

	svc := awsLambda.New(sess)
	l, err := lambdarpc.New(svc, progName)
	if err != nil {
		return err
	}
	cliArgs, o, err := arpicee.ArgsFromFlags(l.Params(), os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", o)
		return nil
	}
	lambdaOutput, err := l.Run(cliArgs)
	if err != nil {
		return err
	}

	output, err := arpicee.Output(lambdaOutput, arpicee.OutputFormat(cliArgs))
	if err != nil {
		return err
	}

	fmt.Printf("%s", output)
	return nil
}

func main() {
	if err := realMain(); err != nil {
		log.Fatal(err)
	}
}
