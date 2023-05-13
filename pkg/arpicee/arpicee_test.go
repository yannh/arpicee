package arpicee

import (
	"flag"
	"fmt"
	"reflect"
	"testing"
)

func TestOutput(t *testing.T) {
	for _, testCase := range []struct {
		res          map[string]interface{}
		outputFormat string
		expect       string
		expectErr    error
	}{
		{
			map[string]interface{}{
				"foo": "bar",
			},
			"json",
			"{\n  \"foo\": \"bar\"\n}\n",
			nil,
		},
		{
			map[string]interface{}{
				"foo": "bar",
			},
			"text",
			"",
			ErrMissingFormatString,
		},
		{
			map[string]interface{}{
				"foo":          "bar",
				"formatString": "foo is {{ .foo }}",
			},
			"text",
			"foo is bar",
			nil,
		},
	} {
		r, err := Output(testCase.res, testCase.outputFormat)
		if r != testCase.expect {
			t.Errorf("output does not match, got %s, expected %s", r, testCase.expect)
		}
		if err != testCase.expectErr {
			t.Errorf("received error %s, expected %s", err, testCase.expectErr)
		}
	}
}

func TestArgsFromFlags(t *testing.T) {
	for i, testCase := range []struct {
		params       []Parameter
		flags        []string
		expect       []Argument
		expectOutput string
		expecterr    error
	}{
		{
			[]Parameter{},
			[]string{"cli"},
			[]Argument{},
			"",
			nil,
		},
		{
			[]Parameter{
				{
					"param1",
					TypeString,
					"",
					true,
				},
			},
			[]string{"cli", "-param1", "foo"},
			[]Argument{
				&ArgumentString{
					Name: "param1",
					Val:  "foo",
				},
			},
			"",
			nil,
		},
		{
			[]Parameter{},
			[]string{"cli", "-h"},
			[]Argument{
				&ArgumentString{
					Name: "param1",
					Val:  "foo",
				},
			},
			`Usage: cli [OPTION]... [FILE OR FOLDER]...
  -h	display help
  -output string
    	output type: json or text (default "text")
`,
			flag.ErrHelp,
		},
		{
			[]Parameter{
				{
					"param1",
					TypeString,
					"",
					true,
				},
			},
			[]string{"cli"},
			[]Argument{},
			`Usage: cli [OPTION]... [FILE OR FOLDER]...
  -h	display help
  -output string
    	output type: json or text (default "text")
  -param1 string
    	
`,
			fmt.Errorf("parameter param1 is required"),
		},
		{
			[]Parameter{},
			[]string{"cli", "-param1", "foo"},
			[]Argument{},
			`flag provided but not defined: -param1
Usage: cli [OPTION]... [FILE OR FOLDER]...
  -h	display help
  -output string
    	output type: json or text (default "text")
`,
			fmt.Errorf("flag provided but not defined: -param1"),
		},
	} {
		got, o, err := ArgsFromFlags(testCase.params, testCase.flags)
		if (err == nil || testCase.expecterr == nil) && err != testCase.expecterr {
			t.Errorf("test %d - expected err: %+v, got %+v", i, testCase.expecterr, err)
			continue
		} else if err != nil && err.Error() != testCase.expecterr.Error() {
			t.Errorf("test %d - expected err: %+v, got %+v", i, testCase.expecterr, err)
			continue
		}
		if o != testCase.expectOutput {
			t.Errorf("test %d, expected output to be:\n%s\nwas:\n%s\n", i, testCase.expectOutput, o)
		}
		if err == nil && !reflect.DeepEqual(testCase.expect, got) {
			t.Errorf("test %d, expected %+v, got %+v", i, testCase.expect, got)
		}
	}
}
