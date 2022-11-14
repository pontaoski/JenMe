package main

import (
	"io/ioutil"
	"os"

	"github.com/dave/jennifer/jen"
)

func main() {
	data, err := ioutil.ReadFile(os.Args[2])
	if err != nil {
		panic(err)
	}

	n, err := Unmarshal(data)
	if err != nil {
		panic(err)
	}

	f := jen.NewFile(os.Args[1])
	for _, node := range n {
		Generate(f, node, n)
	}

	os.Stdout.WriteString(f.GoString())
}
