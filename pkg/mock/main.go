package mock

import "fmt"

type Mock struct {
	name string
}

func Discover() []*Mock {
	return []*Mock{}
}

func New(name string) *Mock {
	return &Mock{name: name}
}

func (m *Mock) Run() error {
	fmt.Printf("Running %s\n", m.name)
	return nil
}
