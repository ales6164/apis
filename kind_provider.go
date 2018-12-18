package apis

import "reflect"

type Provider struct {
	kinds map[string]*Kind
	types map[reflect.Type]*Kind
}

func NewProvider(kinds ...*Kind) *Provider {
	p := new(Provider)
	p.kinds = map[string]*Kind{}
	p.types = map[reflect.Type]*Kind{}
	for _, k := range kinds {
		p.kinds[k.Name] = k
		p.types[k.Type()] = k
	}
	return p
}

func (p *Provider) RegisterKind(k *Kind) {
	if p.kinds == nil {
		p.kinds = map[string]*Kind{}
	}
	if p.types == nil {
		p.types = map[reflect.Type]*Kind{}
	}
	p.kinds[k.Name] = k
	p.types[k.Type()] = k
}

func (p *Provider) GetNameKind(name string) *Kind {
	return p.kinds[name]
}

func (p *Provider) GetTypeKind(typ reflect.Type) *Kind {
	return p.types[typ]
}
