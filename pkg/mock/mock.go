package mock

import (
	"fmt"

	"github.com/yannh/arpicee/pkg/arpicee"
)

type Mock struct {
	name string
}

func Discover() []*Mock {
	return []*Mock{}
}

func New(name string) *Mock {
	return &Mock{name: name}
}

func (m *Mock) Run(args []arpicee.Argument) error {
	output := fmt.Sprintf("Running %s", m.name)
	for _, arg := range args {
		switch arg := arg.(type) {
		case *arpicee.ArgumentString:
			output = fmt.Sprintf("%s --%s %s", output, arg.Name, arg.Val)
			// if arg.Val {
			//	output = fmt.Sprintf("%s --%s true", output, arg.Name)
			// } else {
			//	output = fmt.Sprintf("%s --%s false", output, arg.Name)
			// }
		}
	}

	fmt.Println(output)

	return nil
}
