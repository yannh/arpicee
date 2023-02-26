package arpicee

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"text/template"
)

type RemoteCall interface {
	Run(args []Argument) error
}

type ParamType int

const (
	TypeBool ParamType = iota
	TypeInt
	TypeString
)

type Parameter struct {
	Name        string
	Type        ParamType
	Description string
	Required    bool
}

type Argument interface {
	name() string
}
type ArgumentString struct {
	Name string
	Val  string
}

func GetArg(args []Argument, name string) Argument {
	for _, arg := range args {
		if arg.name() == name {
			return arg
		}
	}

	return nil
}

func (as *ArgumentString) name() string {
	return as.Name
}

type ArgumentBool struct {
	Name string
	Val  bool
}

func (as *ArgumentBool) name() string {
	return as.Name
}

type ArgumentInt struct {
	Name string
	Val  int
}

func (as *ArgumentInt) name() string {
	return as.Name
}

func ValidateArguments(args []Argument, params []Parameter) error {
	for _, param := range params {
		if !param.Required {
			continue
		}

		requiredArgPassed := false
		for _, arg := range os.Args {
			if arg == "-"+param.Name {
				requiredArgPassed = true
			}
		}

		if requiredArgPassed == false {
			return fmt.Errorf("parameter %s is required", param.Name)
		}
	}

	return nil
}

func ArgsFromFlags(params []Parameter) []Argument {
	cliArgs := map[string]interface{}{}
	for _, param := range params {
		switch param.Type {
		case TypeString:
			cliArgs[param.Name] = flag.String(param.Name, "", param.Description)
		case TypeBool:
			cliArgs[param.Name] = flag.Bool(param.Name, false, param.Description)
		case TypeInt:
			cliArgs[param.Name] = flag.Int(param.Name, 0, param.Description)
		}
	}

	// Parameters common to all Lambdas
	cliArgs["outputFormat"] = flag.String("output", "text", "output type: json or text")
	flag.Bool("debug", false, "set debug mode")
	flag.Parse()

	args := []Argument{}
	for k, v := range cliArgs {
		if k == "outputFormat" {
			continue
		}
		switch ca := v.(type) {
		case *string:
			args = append(args, &ArgumentString{
				Name: k,
				Val:  *ca,
			})
		case *int:
			args = append(args, &ArgumentInt{
				Name: k,
				Val:  *ca,
			})
		case *bool:
			args = append(args, &ArgumentBool{
				Name: k,
				Val:  *ca,
			})
		}
	}

	if err := ValidateArguments(args, params); err != nil {
		log.Fatalf("ERROR %s\n", err)
	}

	return args
}

func Output(res map[string]interface{}, outputFormat string) (string, error) {
	if outputFormat == "json" {
		var o []byte
		o, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			return "", err
		}
		return string(o) + "\n", nil
	} else {
		t := template.Must(template.New("").Parse(fmt.Sprintf("%s", res["formatString"])))
		b := bytes.Buffer{}
		if err := t.Execute(&b, res); err != nil {
			return "", err
		}
		return b.String(), nil
	}
}

func OutputFormat(args []Argument) string {
	outputFormat := ""
	arg := GetArg(args, "outputFormat")
	switch a := arg.(type) {
	case *ArgumentString:
		outputFormat = strings.ToLower(a.Val)
	}
	return outputFormat
}
