package kind

import (
	"google.golang.org/appengine/datastore"
	"time"
	"encoding/json"
	"github.com/imdario/mergo"
)

type Holder struct {
	Kind   *Kind
	user   *datastore.Key
	hasKey bool

	value interface{}
	meta  Meta

	hasInputData  bool // when updating
	hasLoadedData bool
}

type Meta struct {
	Id        *datastore.Key `json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	CreatedBy *datastore.Key `json:"createdBy"`
	Value     interface{}    `json:"value"`
}

func (h *Holder) Id() string {
	if h.hasKey {
		return h.meta.Id.Encode()
	}
	return ""
}

func (h *Holder) Value() interface{} {
	return h.value
}

func (h *Holder) Meta() Meta {
	return h.meta
}

func (h *Holder) Parse(body []byte) error {
	h.hasInputData = true
	h.value = h.Kind.New()
	return json.Unmarshal(body, &h.value)
}

func (h *Holder) Bytes(withMeta bool) ([]byte, error) {
	if withMeta {
		m := h.Meta()
		m.Value = h.Value()
		return json.Marshal(m)
	}
	return json.Marshal(h.Value())
}

func (h *Holder) Output() interface{} {
	m := h.Meta()
	m.Value = h.Value()
	return m
}

func (h *Holder) SetKey(k *datastore.Key) {
	if k != nil {
		h.meta.Id = k
		h.hasKey = true
	}
}

func (h *Holder) GetKey() *datastore.Key {
	return h.meta.Id
}

func (h *Holder) Load(ps []datastore.Property) error {
	h.hasLoadedData = true
	var ls []datastore.Property
	for _, prop := range ps {
		switch prop.Name {
		case "meta.createdBy":
			if v, ok := prop.Value.(*datastore.Key); ok {
				h.meta.CreatedBy = v
			}
		case "meta.createdAt":
			if v, ok := prop.Value.(time.Time); ok {
				h.meta.CreatedAt = v
			}
		case "meta.updatedAt":
			if v, ok := prop.Value.(time.Time); ok {
				h.meta.UpdatedAt = v
			}
		default:
			ls = append(ls, prop)
		}
	}

	if h.hasInputData {
		// replace only empty fields
		n := h.Kind.New()
		if err := datastore.LoadStruct(n, ls); err != nil {
			return err
		}
		return mergo.Merge(h.value, n)
	}

	return datastore.LoadStruct(h.value, ls)
}

func (h *Holder) Save() ([]datastore.Property, error) {
	ps, err := datastore.SaveStruct(h.value)
	var now = time.Now()
	if h.hasLoadedData {
		ps = append(ps, datastore.Property{
			Name:  "meta.createdBy",
			Value: h.meta.CreatedBy,
		})
		ps = append(ps, datastore.Property{
			Name:  "meta.createdAt",
			Value: h.meta.CreatedAt,
		})
		ps = append(ps, datastore.Property{
			Name:  "meta.updatedAt",
			Value: now,
		})
	} else {
		ps = append(ps, datastore.Property{
			Name:  "meta.createdBy",
			Value: h.user,
		})
		ps = append(ps, datastore.Property{
			Name:  "meta.createdAt",
			Value: now,
		})
		ps = append(ps, datastore.Property{
			Name:  "meta.updatedAt",
			Value: now,
		})
	}
	return ps, err
}
