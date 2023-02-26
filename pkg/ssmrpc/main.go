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
	Sess        *ssm.SSM
	Name        string
	Description string
	Params      []arpicee.Parameter
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

func New(s *ssm.SSM, name string) (*SSMRPC, error) {
	ssmDoc, err := s.GetDocument(&ssm.GetDocumentInput{
		DocumentFormat:  aws.String("JSON"),
		DocumentVersion: nil,
		Name:            aws.String(name),
		VersionName:     nil,
	})
	if err != nil {
		return nil, fmt.Errorf("failed retrieving ssm document \"%s\": %s", err)
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
		Sess:        s,
		Name:        name,
		Description: "",
		Params:      params,
	}, nil
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
		DocumentName: aws.String(s.Name),
		Parameters:   sInputParams,
	}
	o, err := s.Sess.StartAutomationExecution(&sInput)
	if err != nil {
		log.Fatalf("failed starting automation: %s", err)
	}

	id := o.AutomationExecutionId
	var execution *ssm.GetAutomationExecutionOutput
	for complete := false; complete == false; {
		execution, err = s.Sess.GetAutomationExecution(&ssm.GetAutomationExecutionInput{AutomationExecutionId: id})
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
