package arpicee

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"strings"
	"text/template"
)

type RemoteCall interface {
	Name() string
	Description() string
	Params() []Parameter

	Run(args []Argument) (map[string]interface{}, error)
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
		for _, arg := range args {
			if arg.name() == param.Name {
				requiredArgPassed = true
			}
		}

		if requiredArgPassed == false {
			return fmt.Errorf("parameter %s is required", param.Name)
		}
	}

	return nil
}

type flagsUsageError struct {
	err   error
	usage string
}

func ArgsFromFlags(params []Parameter, flags []string) ([]Argument, string, error) {
	if len(flags) == 0 {
		return nil, "", fmt.Errorf("fatal error: flags array is empty")
	}
	fset := flag.NewFlagSet(flags[0], flag.ContinueOnError)
	var buf bytes.Buffer
	var err error
	fset.SetOutput(&buf)

	cliArgs := map[string]interface{}{}
	for _, param := range params {
		switch param.Type {
		case TypeString:
			cliArgs[param.Name] = fset.String(param.Name, "", param.Description)
		case TypeBool:
			cliArgs[param.Name] = fset.Bool(param.Name, false, param.Description+" (Default: false)")
		case TypeInt:
			cliArgs[param.Name] = fset.Int(param.Name, 0, param.Description)
		}
	}

	// Parameters common to all Lambdas
	cliArgs["outputFormat"] = fset.String("output", "text", "output type: json or text")
	cliArgs["help"] = fset.Bool("h", false, "display help")
	// cliArgs["debug"] = fset.Bool("debug", false, "set debug mode")
	fset.Usage = func() {
		fmt.Fprintf(&buf, "Usage: %s [OPTION]... [FILE OR FOLDER]...\n", flags[0])
		fset.PrintDefaults()
	}
	if len(flags) > 1 {
		err := fset.Parse(flags[1:])
		if err != nil {
			return nil, buf.String(), err
		}
	}

	if *(cliArgs["help"].(*bool)) {
		fset.Usage()
		return nil, buf.String(), flag.ErrHelp
	}

	args := []Argument{}
	for k, v := range cliArgs {
		if k == "outputFormat" {
			continue
		}

		// Ensure we do not set the argument if the parameter was not explicitly passed
		found := false
		fset.Visit(func(f *flag.Flag) {
			if f.Name == k {
				found = true
			}
		})
		if found == false {
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

	err = ValidateArguments(args, params)
	if err != nil {
		fset.Usage()
	}

	return args, buf.String(), err
}

var ErrMissingFormatString = errors.New("failed to format response: object missing a formatString")

func Output(res map[string]interface{}, outputFormat string) (string, error) {
	if outputFormat == "json" {
		var o []byte
		o, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			return "", err
		}
		return string(o) + "\n", nil
	} else {
		if _, ok := res["formatString"]; !ok {
			return "", ErrMissingFormatString
		}
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

func Equal(expected, is RemoteCall) bool {
	if expected.Name() != is.Name() {
		return false
	}
	if len(expected.Params()) != len(is.Params()) {
		return false
	}

OUTER:
	for _, expP := range expected.Params() {
		for _, isP := range is.Params() {
			if expP.Name != isP.Name {
				continue
			}
			if expP.Description != isP.Description ||
				expP.Type != isP.Type ||
				expP.Required != isP.Required {
				return false
			}
			continue OUTER // We found the parameter
		}
		return false
	}

	return true
}
