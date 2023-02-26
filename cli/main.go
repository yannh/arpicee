package main

import (
	"github.com/yannh/arpicee/pkg/arpicee"
	"github.com/yannh/arpicee/pkg/mock"
)

func main() {
	procs := []arpicee.RemoteCall{}
	procs = append(procs, mock.New("toto"))
	for _, p := range procs {
		p.Run([]arpicee.Argument{
			arpicee.ArgumentString{
				Name: "foo",
				Val:  "bar",
			},
		})
	}
}
