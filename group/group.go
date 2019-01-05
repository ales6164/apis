package group

import (
	"github.com/ales6164/apis/collection"
	"github.com/ales6164/apis/kind"
)

type Group struct {
	*collection.Collection
}

func New(name string, i interface{}) kind.Kind {
	return &Group{Collection: collection.New(name, i)}
}

func (g *Group) IsNamespace() bool {
	return true
}
