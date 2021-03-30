package apis

import (
	"cloud.google.com/go/datastore"
	"golang.org/x/net/context"
)

type Kind struct {
	*KindOptions
	fields map[string]*Field
	inited bool
}

type KindOptions struct {
	Name      string
	ProjectID string
	Fields    []*Field
}

type Field struct {
	Name       string
	IsRequired bool
	Multiple   bool
	NoIndex    bool

	Kind *Kind
}

func NewKind(opt *KindOptions) *Kind {
	k := &Kind{
		KindOptions: opt,
	}
	for _, f := range k.Fields {
		if k.fields == nil {
			k.fields = map[string]*Field{}
		}
		k.fields[f.Name] = f
	}
	return k
}

func (k *Kind) NewHolder(projectId string, user *datastore.Key) *Holder {
	k.ProjectID = projectId
	return &Holder{
		Kind:              k,
		user:              user,
		ProjectID:         projectId,
		preparedInputData: map[*Field][]datastore.Property{},
		loadedStoredData:  map[string][]datastore.Property{},
	}
}

func (k *Kind) NewIncompleteKey(c context.Context, parent *datastore.Key) *datastore.Key {
	return datastore.IncompleteKey(k.Name, parent)
}

func (k *Kind) NewKey(c context.Context, nameId string, parent *datastore.Key) *datastore.Key {
	return datastore.NameKey(k.Name, nameId, parent)
}
