package ssmrpc

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/yannh/arpicee/pkg/arpicee"
)

type SSMRPC struct {
	sess        *ssm.SSM
	name        string
	description string
	params      []arpicee.Parameter
}

type ssmDocParameter struct {
	Type        string
	Description string
	Default     *string
}

type ssmDocContent struct {
	Description string
	Parameters  map[string]ssmDocParameter
}

func (sr *SSMRPC) Name() string {
	return sr.name
}

func (sr *SSMRPC) Description() string {
	return sr.description
}

func (sr *SSMRPC) Params() []arpicee.Parameter {
	return sr.params
}

func New(s *ssm.SSM, name string) (*SSMRPC, error) {
	ssmDoc, err := s.GetDocument(&ssm.GetDocumentInput{
		DocumentFormat:  aws.String("JSON"),
		DocumentVersion: nil,
		Name:            aws.String(name),
		VersionName:     nil,
	})
	if err != nil {
		return nil, fmt.Errorf("failed retrieving ssm document \"%s\": %w", name, err)
	}

	ssmDocContentBytes := []byte(*ssmDoc.Content)
	// fmt.Printf("%s", ssmDocContentBytes)
	var sdc ssmDocContent
	json.Unmarshal(ssmDocContentBytes, &sdc)

	params := []arpicee.Parameter{}
	for paramName, param := range sdc.Parameters {
		switch strings.ToLower(param.Type) {
		case "string":
			required := false
			// Default only set on optional params in SSM
			if param.Default == nil {
				required = true
			}

			params = append(params, arpicee.Parameter{
				Name:        paramName,
				Type:        arpicee.TypeString,
				Description: param.Description,
				Required:    required,
			})
		case "bool":
			params = append(params, arpicee.Parameter{
				Name:        paramName,
				Type:        arpicee.TypeBool,
				Description: "",
				Required:    false,
			})
		}
	}

	return &SSMRPC{
		sess:        s,
		name:        name,
		description: "",
		params:      params,
	}, nil
}

func TagFilter(tagName, tagValue string) func(configuration *ssm.DocumentIdentifier) bool {
	return func(ssm *ssm.DocumentIdentifier) bool {
		if len(ssm.Tags) == 0 {
			return false
		}
		for _, t := range ssm.Tags {
			if *t.Key == tagName {
				if *t.Value == tagValue {
					return true
				}
			}
		}
		return false
	}
}

func (s *SSMRPC) Run(args []arpicee.Argument) (map[string]interface{}, error) {
	sInputParams := map[string][]*string{}
	for _, sa := range args {
		switch sat := sa.(type) {
		case *arpicee.ArgumentString:
			sInputParams[sat.Name] = []*string{aws.String(sat.Val)}
		}
	}
	sInput := ssm.StartAutomationExecutionInput{
		DocumentName: aws.String(s.name),
		Parameters:   sInputParams,
	}
	o, err := s.sess.StartAutomationExecution(&sInput)
	if err != nil {
		log.Fatalf("failed starting automation: %s", err)
	}

	id := o.AutomationExecutionId
	var execution *ssm.GetAutomationExecutionOutput
	for complete := false; complete == false; {
		execution, err = s.sess.GetAutomationExecution(&ssm.GetAutomationExecutionInput{AutomationExecutionId: id})
		if err != nil {
			log.Fatalf("error getting Automation execution: %s", err)
		}
		if *execution.AutomationExecution.AutomationExecutionStatus == "InProgress" {
			time.Sleep(1 * time.Second)
		} else {
			complete = true
		}
	}

	res := map[string]interface{}{}
	for _, v := range execution.AutomationExecution.Outputs {
		json.Unmarshal([]byte(*v[0]), &res)
	}
	return res, nil
}

func Discover(svc *ssm.SSM, filters []func(configuration *ssm.DocumentIdentifier) bool) ([]*SSMRPC, error) {
	var err error
	var ssmRPC []*SSMRPC
	result := &ssm.ListDocumentsOutput{}
	nCall := 1

	for ; result != nil && (nCall == 1 || result.NextToken != nil); nCall++ {
		input := &ssm.ListDocumentsInput{
			MaxResults: aws.Int64(50),
			NextToken:  result.NextToken,
		}
		if result, err = svc.ListDocuments(input); err != nil {
			return nil, err
		}

	OUTER:
		for _, doc := range result.DocumentIdentifiers {
			for _, filter := range filters {
				if !filter(doc) {
					continue OUTER
				}
			}
			d, err := New(svc, *doc.Name)
			if err != nil {
				return nil, fmt.Errorf("failed retrieving SSM Document: %w", err)
			}
			ssmRPC = append(ssmRPC, d)
		}
	}

	return ssmRPC, nil
}
