package lambdarpc

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	awsLambda "github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/yannh/arpicee/pkg/arpicee"
)

type mockLambdaClient struct {
	lambdaiface.LambdaAPI
	getFunction   func(input *awsLambda.GetFunctionInput) (*awsLambda.GetFunctionOutput, error)
	listFunctions func(input *awsLambda.ListFunctionsInput) (*awsLambda.ListFunctionsOutput, error)
	listTags      func(input *awsLambda.ListTagsInput) (*awsLambda.ListTagsOutput, error)
}

func (m *mockLambdaClient) GetFunction(input *awsLambda.GetFunctionInput) (*awsLambda.GetFunctionOutput, error) {
	return m.getFunction(input)
}

func (m *mockLambdaClient) ListFunctions(input *awsLambda.ListFunctionsInput) (*awsLambda.ListFunctionsOutput, error) {
	return m.listFunctions(input)
}

func (m *mockLambdaClient) ListTags(input *awsLambda.ListTagsInput) (*awsLambda.ListTagsOutput, error) {
	return m.listTags(input)
}

func TestNewLambdaRPC(t *testing.T) {
	for _, testCase := range []struct {
		f        func(input *awsLambda.GetFunctionInput) (*awsLambda.GetFunctionOutput, error)
		name     string
		expected *LambdaRPC
	}{
		{
			func(input *awsLambda.GetFunctionInput) (*awsLambda.GetFunctionOutput, error) {
				return &awsLambda.GetFunctionOutput{
					Configuration: &awsLambda.FunctionConfiguration{
						Description: aws.String("some description"),
					},
					Tags: map[string]*string{"param:param1:string/required": aws.String("some description")},
				}, nil
			},
			"foo",
			&LambdaRPC{
				name:        "foo",
				description: "bar",
				params: []arpicee.Parameter{
					{
						Name:        "param1",
						Type:        arpicee.TypeString,
						Required:    true,
						Description: "some description",
					},
				},
			},
		},
		{
			func(input *awsLambda.GetFunctionInput) (*awsLambda.GetFunctionOutput, error) {
				return &awsLambda.GetFunctionOutput{
					Configuration: &awsLambda.FunctionConfiguration{
						Description: aws.String("some description"),
					},
					Tags: map[string]*string{
						"param:foo:int":           aws.String("some description"),
						"someothertag":            aws.String("foobar"),
						"param:bar:bool/required": aws.String("something"),
					},
				}, nil
			},
			"foo",
			&LambdaRPC{
				name:        "foo",
				description: "bar",
				params: []arpicee.Parameter{
					{
						Name:        "bar",
						Type:        arpicee.TypeBool,
						Required:    true,
						Description: "something",
					},
					{
						Name:        "foo",
						Type:        arpicee.TypeInt,
						Required:    false,
						Description: "some description",
					},
				},
			},
		},
	} {
		c := &mockLambdaClient{
			getFunction: testCase.f,
		}
		rpc, err := New(c, testCase.name)
		if err != nil {
			t.Errorf("got error instanciating lambdarpc: %s", err)
		}
		if !arpicee.Equal(testCase.expected, rpc) {
			t.Errorf("expected %+v, got %+v", testCase.expected, rpc)
		}
	}
}

func TestDiscover(t *testing.T) {
	type lambda struct {
		name        string
		arn         string
		description string
		tags        map[string]*string
	}
	for _, testCase := range []struct {
		lambdas  []lambda
		filters  []func(configuration *awsLambda.ListTagsOutput) bool
		expected []*LambdaRPC
	}{
		{
			lambdas: []lambda{
				{
					name:        "lambda1",
					arn:         "foo:lambda1:bar",
					description: "some description",
					tags: map[string]*string{
						"param:foo:int":           aws.String("some description"),
						"someothertag":            aws.String("foobar"),
						"param:bar:bool/required": aws.String("something"),
					},
				},
			},
			filters:  []func(configuration *awsLambda.ListTagsOutput) bool{TagFilter("foo", "bar")},
			expected: []*LambdaRPC{ // The function is missing the tag foo with value bar, we should ignore it
			},
		},
		{
			lambdas: []lambda{
				{
					name:        "lambda1",
					arn:         "foo:lambda1:bar",
					description: "some description",
					tags: map[string]*string{
						"param:foo:int":           aws.String("some description"),
						"someothertag":            aws.String("foobar"),
						"param:bar:bool/required": aws.String("something"),
						"foo":                     aws.String("bar"),
					},
				},
			},
			filters: []func(configuration *awsLambda.ListTagsOutput) bool{TagFilter("foo", "bar")},
			expected: []*LambdaRPC{
				{
					name:        "lambda1",
					description: "some description",
					params: []arpicee.Parameter{
						{
							Name:        "foo",
							Type:        arpicee.TypeInt,
							Required:    false,
							Description: "some description",
						},
						{
							Name:        "bar",
							Type:        arpicee.TypeBool,
							Required:    true,
							Description: "something",
						},
					},
				},
			},
		},
	} {
		nget := 0
		nlistTags := 0
		c := &mockLambdaClient{
			getFunction: func(input *awsLambda.GetFunctionInput) (*awsLambda.GetFunctionOutput, error) {
				gfo := &awsLambda.GetFunctionOutput{
					Configuration: &awsLambda.FunctionConfiguration{
						Description: aws.String(testCase.lambdas[nget].description),
					},
					Tags: testCase.lambdas[nget].tags,
				}
				nget++
				return gfo, nil
			},
			listFunctions: func(input *awsLambda.ListFunctionsInput) (*awsLambda.ListFunctionsOutput, error) {
				list := []*awsLambda.FunctionConfiguration{}
				for _, f := range testCase.lambdas {
					list = append(list, &awsLambda.FunctionConfiguration{
						FunctionName: aws.String(f.name),
						Description:  aws.String(f.description),
						FunctionArn:  aws.String(f.arn),
					})
				}
				return &awsLambda.ListFunctionsOutput{Functions: list}, nil
			},
			listTags: func(input *awsLambda.ListTagsInput) (*awsLambda.ListTagsOutput, error) {
				lo := &awsLambda.ListTagsOutput{
					Tags: testCase.lambdas[nlistTags].tags,
				}
				nlistTags++
				return lo, nil
			},
		}

		rpcs, err := Discover(c, testCase.filters)
		if err != nil {
			t.Errorf("failed discovering lambdas: %s", err)
		}

		if len(rpcs) != len(testCase.expected) {
			t.Errorf("expected to discover %d lambdas, found %d", len(testCase.expected), len(rpcs))
		}

		if len(rpcs) != len(testCase.expected) {
			t.Errorf("expected to discover %d lambdas, found %d", len(testCase.expected), len(rpcs))
		}
	OUTER:
		for _, x := range testCase.expected {
			for _, y := range rpcs {
				if arpicee.Equal(x, y) {
					continue OUTER
				}
			}
			t.Errorf("expected to find rpc %+v, did not: %+v", x, rpcs)
		}
	}
}

func TestSerializeArguments(t *testing.T) {
	res, err := serializeArguments([]arpicee.Argument{
		&arpicee.ArgumentString{
			Name: "foo",
			Val:  "bar",
		},
		&arpicee.ArgumentInt{
			Name: "n",
			Val:  123,
		},
		&arpicee.ArgumentBool{
			Name: "tobeornottobe",
			Val:  true,
		},
	})
	expected := `{
  "foo": "bar",
  "n": 123,
  "tobeornottobe": true
}`

	if err != nil {
		t.Errorf("failed serializing arguments: %s", err.Error())
	}
	if string(res) != expected {
		t.Errorf("expected serialized arguments to be:\n%s\nGot:\n%s\n", expected, string(res))
	}
}
