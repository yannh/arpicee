package lambdarpc

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	awsLambda "github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/yannh/arpicee/pkg/arpicee"
)

type LambdaRPC struct {
	svc         lambdaiface.LambdaAPI
	name        string
	description string
	params      []arpicee.Parameter
}

func (lr *LambdaRPC) Name() string {
	return lr.name
}

func (lr *LambdaRPC) Description() string {
	return lr.description
}

func (lr *LambdaRPC) Params() []arpicee.Parameter {
	return lr.params
}

func TagFilter(tagName, tagValue string) func(configuration *awsLambda.ListTagsOutput) bool {
	return func(lambda *awsLambda.ListTagsOutput) bool {
		return lambda.Tags != nil && lambda.Tags[tagName] != nil && *lambda.Tags[tagName] == tagValue
	}
}

func inArray(ar []string, el string) bool {
	for _, v := range ar {
		if v == el {
			return true
		}
	}
	return false
}

func getParamFlags(paramName string) []string {
	sep := ":"
	flagSeparator := "/"

	parts := strings.Split(paramName, sep)
	if len(parts) < 3 {
		return []string{}
	}

	return strings.Split(parts[2], flagSeparator)
}

func New(svc lambdaiface.LambdaAPI, name string) (*LambdaRPC, error) {
	input := awsLambda.GetFunctionInput{
		FunctionName: &name,
	}

	output, err := svc.GetFunction(&input)
	if err != nil {
		return nil, err
	}

	l := LambdaRPC{
		svc:         svc,
		name:        name,
		description: *output.Configuration.Description,
	}

	l.params = []arpicee.Parameter{}
	for tagName, tagValue := range output.Tags {
		if strings.HasPrefix(tagName, "param:") {
			parts := strings.Split(tagName, ":")
			if len(parts) == 3 {
				required := false
				t := arpicee.TypeString

				flags := getParamFlags(tagName)

				if inArray(flags, "required") {
					required = true
				}

				if inArray(flags, "int") {
					t = arpicee.TypeInt
				} else if inArray(flags, "bool") {
					t = arpicee.TypeBool
				}

				l.params = append(l.params, arpicee.Parameter{
					Name:        parts[1],
					Type:        t,
					Description: *tagValue,
					Required:    required,
				})
			}
		}
	}

	return &l, nil
}

func serializeArguments(args []arpicee.Argument) ([]byte, error) {
	m := map[string]interface{}{}
	for _, arg := range args {
		switch a := arg.(type) {
		case *arpicee.ArgumentString:
			m[a.Name] = a.Val
		case *arpicee.ArgumentInt:
			m[a.Name] = a.Val
		case *arpicee.ArgumentBool:
			m[a.Name] = a.Val
		}
	}

	return json.MarshalIndent(m, "", "  ")
}

func (l *LambdaRPC) Run(args []arpicee.Argument) (map[string]interface{}, error) {
	payload, err := serializeArguments(args)
	if err != nil {
		return nil, fmt.Errorf("failed serializing payload: %s", payload)
	}

	input := &awsLambda.InvokeInput{
		ClientContext:  nil,
		FunctionName:   aws.String(l.name),
		InvocationType: aws.String(awsLambda.InvocationTypeRequestResponse),
		LogType:        nil,
		Payload:        payload,
		Qualifier:      nil,
	}

	output, err := l.svc.Invoke(input)
	if err != nil {
		return nil, fmt.Errorf("failed invoking lambda %s: %s", l.Name(), err)
	}

	var res map[string]interface{}
	err = json.Unmarshal(output.Payload, &res)

	return res, nil
}

func Discover(svc lambdaiface.LambdaAPI, filters []func(configuration *awsLambda.ListTagsOutput) bool) ([]*LambdaRPC, error) {
	var err error
	var automationLambdas []*LambdaRPC

	result := &awsLambda.ListFunctionsOutput{}
	nCall := 1
	for ; result != nil && (nCall == 1 || result.NextMarker != nil); nCall++ {
		input := &awsLambda.ListFunctionsInput{
			MaxItems: aws.Int64(50),
			Marker:   result.NextMarker,
		}

		if result, err = svc.ListFunctions(input); err != nil {
			return nil, err
		}

	OUTER:
		for _, fn := range result.Functions {
			tagsOutput, err := svc.ListTags(&awsLambda.ListTagsInput{Resource: fn.FunctionArn})
			if err != nil {
				return nil, err
			}

			for _, filter := range filters {
				if !filter(tagsOutput) {
					continue OUTER
				}
			}

			f, _ := New(svc, *fn.FunctionName)
			automationLambdas = append(automationLambdas, f)
		}
	}

	return automationLambdas, nil
}
