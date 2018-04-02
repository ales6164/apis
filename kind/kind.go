package kind

import (
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
)

type Kind struct {
	*Options
	fields map[string]*Field
	inited bool
}

type Options struct {
	Name   string
	Fields []*Field
}

type Field struct {
	Name       string
	IsRequired bool
	Multiple   bool
	NoIndex    bool

	Kind *Kind
}

func New(opt *Options) *Kind {
	k := &Kind{
		Options: opt,
	}
	for _, f := range k.Fields {
		if k.fields == nil {
			k.fields = map[string]*Field{}
		}
		k.fields[f.Name] = f
	}
	return k
}

func (k *Kind) NewHolder(user *datastore.Key) *Holder {
	return &Holder{
		Kind:              k,
		user:              user,
		preparedInputData: map[*Field][]datastore.Property{},
		loadedStoredData:  map[string][]datastore.Property{},
	}
}

func (k *Kind) NewIncompleteKey(c context.Context, parent *datastore.Key) *datastore.Key {
	return datastore.NewIncompleteKey(c, k.Name, parent)
}

func (k *Kind) NewKey(c context.Context, nameId string, parent *datastore.Key) *datastore.Key {
	return datastore.NewKey(c, k.Name, nameId, 0, parent)
}
