package kind

import (
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"reflect"
	"strings"
	"github.com/ales6164/apis/errors"
	"google.golang.org/appengine/search"
	"net/http"
)

type Kind struct {
	Type        reflect.Type
	MetaFields  []MetaField
	MetaIdField MetaField
	*Options
	fields      []*Field

	info *Info
	ui   *UI

	searchFields map[string]SearchField // map of all fields
}

type Options struct {
	Name                 string // name used to represent kind on the backend
	EnableSearch         bool
	RetrieveByIDOnSearch bool
	SearchType           reflect.Type
}

type MetaField struct {
	Type      string
	FieldName string
}

type Field struct {
	Name       string
	DoStore    bool
	IsRequired bool // moving this somewhere else?
	Multiple   bool
	NoIndex    bool

	MetaField   string
	Label       string // json field name
	Json        string // json field name
	Type        string
	StructField reflect.StructField

	Kind *Kind
}

type SearchField struct {
	Name    string
	IsFacet bool
}

type Info struct {
	Name         string       `json:"name"`
	Label        string       `json:"label"`
	LabelMany    string       `json:"label_many"`
	SearchIndex  string       `json:"search_index"`
	Fields       []*InfoField `json:"fields"`
	RelativePath string       `json:"relative_path"`
	HasGet       bool         `json:"get"`
	HasPost      bool         `json:"post"`
	HasPut       bool         `json:"put"`
	HasDelete    bool         `json:"delete"`
}

type InfoField struct {
	Label      string          `json:"label,omitempty"`
	Name       string          `json:"name,omitempty"`
	Meta       string          `json:"meta,omitempty"`
	Hidden     bool            `json:"hidden,omitempty"` // only in on create window
	Attributes []InfoFieldAttr `json:"attributes,omitempty"`
	Type       string          `json:"type,omitempty"`
	IsInput    bool            `json:"is_input,omitempty"`
	IsSelect   bool            `json:"is_select,omitempty"`
	IsTextArea bool            `json:"is_text_area,omitempty"`
	InputType  string          `json:"input_type,omitempty"`
}

type InfoFieldAttr struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

type UI struct {
	Label        string
	LabelMany    string
	relativePath string
	methods      []string
}

func (k *Kind) UI() *UI {
	return k.ui
}
func (k *Kind) SetUI(ui *UI, relativePath string, methods []string) {
	ui.relativePath = relativePath
	ui.methods = methods
	k.ui = ui
}
func (k *Kind) HasUI() bool {
	return k.ui != nil
}
func (k *Kind) Info() *Info {
	if k.info == nil && k.HasUI() {
		info := &Info{
			Name:         k.Name,
			Label:        k.ui.Label,
			LabelMany:    k.ui.LabelMany,
			SearchIndex:  k.SearchType.Name(),
			RelativePath: k.ui.relativePath,
		}

		for _, m := range k.ui.methods {
			switch m {
			case http.MethodGet:
				info.HasGet = true
			case http.MethodPost:
				info.HasPost = true
			case http.MethodPut:
				info.HasPut = true
			case http.MethodDelete:
				info.HasDelete = true
			}
		}

		k.checkFields()

		for _, f := range k.fields {
			infoField := &InfoField{
				Label: f.Label,
				Name:  f.Json,
				Type:  f.Type,
			}
			switch f.Type {
			case "*datastore.Key":
				infoField.IsInput = true
				infoField.InputType = "text"
				infoField.Attributes = append(infoField.Attributes, InfoFieldAttr{"type", infoField.InputType})
			case "time.Time":
				infoField.IsInput = true
				infoField.InputType = "datetime-local"
				infoField.Attributes = append(infoField.Attributes, InfoFieldAttr{"type", infoField.InputType})
			case "string":
				infoField.IsInput = true
				infoField.InputType = "text"
				infoField.Attributes = append(infoField.Attributes, InfoFieldAttr{"type", infoField.InputType})
			case "float64":
				infoField.IsInput = true
				infoField.InputType = "number"
				infoField.Attributes = append(infoField.Attributes, InfoFieldAttr{"type", infoField.InputType})
				infoField.Attributes = append(infoField.Attributes, InfoFieldAttr{"step", "any"})
				infoField.Attributes = append(infoField.Attributes, InfoFieldAttr{"pattern", `-?[0-9]*(\.[0-9]+)?`})
			case "float32":
				infoField.IsInput = true
				infoField.InputType = "number"
				infoField.Attributes = append(infoField.Attributes, InfoFieldAttr{"type", infoField.InputType})
				infoField.Attributes = append(infoField.Attributes, InfoFieldAttr{"step", "any"})
				infoField.Attributes = append(infoField.Attributes, InfoFieldAttr{"pattern", `-?[0-9]*(\.[0-9]+)?`})
			case "int64":
				infoField.IsInput = true
				infoField.InputType = "number"
				infoField.Attributes = append(infoField.Attributes, InfoFieldAttr{"type", infoField.InputType})
				infoField.Attributes = append(infoField.Attributes, InfoFieldAttr{"step", "1"})
			case "int32":
				infoField.IsInput = true
				infoField.InputType = "number"
				infoField.Attributes = append(infoField.Attributes, InfoFieldAttr{"type", infoField.InputType})
				infoField.Attributes = append(infoField.Attributes, InfoFieldAttr{"step", "1"})
			case "int":
				infoField.IsInput = true
				infoField.InputType = "number"
				infoField.Attributes = append(infoField.Attributes, InfoFieldAttr{"type", infoField.InputType})
				infoField.Attributes = append(infoField.Attributes, InfoFieldAttr{"step", "1"})
			default:

			}

			if len(f.MetaField) > 0 {
				infoField.Attributes = append(infoField.Attributes, InfoFieldAttr{"readonly", "true"})
				infoField.Attributes = append(infoField.Attributes, InfoFieldAttr{"disabled", "true"})
				infoField.Hidden = true
				infoField.Meta = f.MetaField
			}

			info.Fields = append(info.Fields, infoField)
		}

		k.info = info
	}
	return k.info
}

func New(t reflect.Type, opt *Options) *Kind {
	if opt == nil {
		opt = new(Options)
	}

	if t.Kind() != reflect.Struct {
		panic(errors.New("type not of kind struct"))
	}

	k := &Kind{
		Type:         t,
		Options:      opt,
		searchFields: map[string]SearchField{},
	}

	k.checkFields()

	if len(k.Name) == 0 {
		k.Name = t.Name()
	}

	if k.SearchType == nil {
		k.SearchType = t
	}

	for i := 0; i < k.SearchType.NumField(); i++ {
		searchField := k.SearchType.Field(i)

		var field = SearchField{
			Name:    searchField.Name,
			IsFacet: false,
		}

		if val, ok := searchField.Tag.Lookup("search"); ok {

			for n, v := range strings.Split(val, ",") {
				v = strings.TrimSpace(v)
				switch n {
				case 0:
					if v == "-" {
						field.Name = ""
					} else {
						field.Name = v
					}
				case 1:
					field.IsFacet = v == "facet"
				}
			}
		}

		k.searchFields[field.Name] = field

	}

	return k
}

func (k *Kind) checkFields() {
	k.fields = []*Field{}
	for i := 0; i < k.Type.NumField(); i++ {
		structField := k.Type.Field(i)
		field := new(Field)
		field.StructField = structField
		field.Type = structField.Type.String()
		if val, ok := structField.Tag.Lookup("datastore"); ok {
			for n, v := range strings.Split(val, ",") {
				v = strings.TrimSpace(v)
				switch n {
				case 0:
					if v == "-" {
						field.DoStore = false
					} else {
						field.DoStore = true
					}
					field.Name = v
				case 1:
					field.NoIndex = v == "noindex"
				}
			}
		}
		if val, ok := structField.Tag.Lookup("json"); ok {
			for n, v := range strings.Split(val, ",") {
				v = strings.TrimSpace(v)
				switch n {
				case 0:
					field.Json = v
				}
			}
		}
		if val, ok := structField.Tag.Lookup("label"); ok {
			for n, v := range strings.Split(val, ",") {
				v = strings.TrimSpace(v)
				switch n {
				case 0:
					field.Label = v
				}
			}
		}
		if val, ok := structField.Tag.Lookup("apis"); ok {
			for n, v := range strings.Split(val, ",") {
				v = strings.TrimSpace(v)
				switch n {
				case 0:
					if v == "id" {
						k.MetaIdField = MetaField{
							Type:      v,
							FieldName: structField.Name,
						}
					} else {
						k.MetaFields = append(k.MetaFields, MetaField{
							Type:      v,
							FieldName: structField.Name,
						})
					}
					field.MetaField = v

				}
			}
		}

		if len(field.Name) == 0 {
			field.Name = structField.Name
		}

		if len(field.Json) == 0 {
			field.Json = field.Name
		}

		if structField.Type.Kind() == reflect.Slice {
			field.Multiple = true
		}
		k.fields = append(k.fields, field)
	}
}

func (k *Kind) New() interface{} {
	return reflect.New(k.Type).Interface()
}

func (k *Kind) NewHolder(user *datastore.Key) *Holder {
	return &Holder{
		Kind:  k,
		user:  user,
		value: k.New(),
	}
}

func (k *Kind) NewIncompleteKey(c context.Context, parent *datastore.Key) *datastore.Key {
	return datastore.NewIncompleteKey(c, k.Name, parent)
}

func (k *Kind) NewKey(c context.Context, nameId string, parent *datastore.Key) *datastore.Key {
	return datastore.NewKey(c, k.Name, nameId, 0, parent)
}

func (k *Kind) DecodeKey(encoded string) (*datastore.Key, error) {
	key, err := datastore.DecodeKey(encoded)
	if err != nil {
		return nil, err
	}
	if key.Kind() != k.Name {
		return nil, errors.New("key unathorized access")
	}
	return key, nil
}

func (k *Kind) RetrieveSearchParameter(parameterName string, value string, fields []search.Field, facets []search.Facet) ([]search.Field, []search.Facet) {
	if f, ok := k.searchFields[parameterName]; ok {
		if f.IsFacet {
			// todo: currently only supports facet type search.Atom
			facets = append(facets, search.Facet{Name: parameterName, Value: search.Atom(value)})
		} else {
			fields = append(fields, search.Field{Name: parameterName, Value: value})
		}
	}
	return fields, facets
}
