package arpicee

type RemoteCall interface {
	Run(args []Argument) error
}

type Argument interface{}
type ArgumentString struct {
	Name string
	Val  string
}
type ArgumentBool struct {
	Name string
	Val  bool
}
