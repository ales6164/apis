package kind

import "reflect"

type KindProvider struct {
	kinds map[string]*Kind
	types map[reflect.Type]*Kind
}

func NewKindProvider(kinds ...*Kind) *KindProvider {
	p := new(KindProvider)
	p.kinds = map[string]*Kind{}
	p.types = map[reflect.Type]*Kind{}
	for _, k := range kinds {
		p.kinds[k.Name] = k
		p.types[k.Type()] = k
	}
	return p
}

func (p *KindProvider) RegisterKind(k *Kind) {
	if p.kinds == nil {
		p.kinds = map[string]*Kind{}
	}
	if p.types == nil {
		p.types = map[reflect.Type]*Kind{}
	}
	p.kinds[k.Name] = k
	p.types[k.Type()] = k
}

func (p *KindProvider) GetNameKind(name string) *Kind {
	return p.kinds[name]
}

func (p *KindProvider) GetTypeKind(typ reflect.Type) *Kind {
	return p.types[typ]
}
