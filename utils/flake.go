package utils

import (
	"github.com/bwmarrin/snowflake"
)

type Flake struct {
	Node *snowflake.Node
}

func NewFlake() *Flake {
	snowflake.Epoch = (2013 - 1970) * 31536000 * 1000

	node, err := snowflake.NewNode(45)
	if err != nil {
		panic(err)
	}
	flake := Flake{
		Node: node,
	}
	return &flake
}

func (f *Flake) Generate() int64 {
	return f.Node.Generate().Int64()
}
