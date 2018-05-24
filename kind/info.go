package kind

import (
	"reflect"
	"net/http"
)

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
func (k *Kind) Info() *KindInfo {
	if k.info == nil && k.HasUI() {
		fieldInfo := informizeType(k.Type)
		fieldInfo.Label = k.ui.Label
		fieldInfo.LabelMany = k.ui.Label
		fieldInfo.SearchIndex = k.SearchType.Name()
		fieldInfo.RelativePath = k.ui.relativePath
		for _, m := range k.ui.methods {
			switch m {
			case http.MethodGet:
				fieldInfo.HasGet = true
			case http.MethodPost:
				fieldInfo.HasPost = true
			case http.MethodPut:
				fieldInfo.HasPut = true
			case http.MethodDelete:
				fieldInfo.HasDelete = true
			}
		}

		k.info = fieldInfo
	}
	return k.info
}

type KindInfo struct {
	Name   string       `json:"name,omitempty"`
	Type   string       `json:"type,omitempty"`
	Fields []*FieldInfo `json:"fields,omitempty"`

	IsNested     bool   `json:"is_nested,omitempty"`
	Label        string `json:"label,omitempty"`
	LabelMany    string `json:"label_many,omitempty"`
	SearchIndex  string `json:"search_index,omitempty"`
	RelativePath string `json:"relative_path,omitempty"`
	HasGet       bool   `json:"get,omitempty"`
	HasPost      bool   `json:"post,omitempty"`
	HasPut       bool   `json:"put,omitempty"`
	HasDelete    bool   `json:"delete,omitempty"`
}

type FieldInfo struct {
	Label string `json:"label,omitempty"`
	Name  string `json:"name,omitempty"`

	Meta       string          `json:"meta,omitempty"`
	Hidden     bool            `json:"hidden,omitempty"` // only in on create window
	Attributes []InfoFieldAttr `json:"attributes,omitempty"`
	Type       string          `json:"type,omitempty"`
	IsInput    bool            `json:"is_input,omitempty"`
	IsSlice    bool            `json:"is_slice,omitempty"`
	IsStruct   bool            `json:"is_struct,omitempty"`
	IsSelect   bool            `json:"is_select,omitempty"`
	IsTextArea bool            `json:"is_text_area,omitempty"`
	InputType  string          `json:"input_type,omitempty"`

	Kind *KindInfo `json:"kind,omitempty"`
}

func informizeType(t reflect.Type) *KindInfo {
	kindInfo := new(KindInfo)

	kindInfo.Type = t.String()
	kindInfo.Name = t.Name()

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		fieldInfo := new(FieldInfo)
		fieldInfo.Type = f.Type.String()

		if m, ok := f.Tag.Lookup("json"); ok {
			fieldInfo.Name = m
		} else {
			fieldInfo.Name = f.Name
		}

		if m, ok := f.Tag.Lookup("label"); ok {
			fieldInfo.Label = m
		} else {
			fieldInfo.Label = fieldInfo.Name
		}

		if m, ok := f.Tag.Lookup("apis"); ok {
			fieldInfo.Meta = m
			fieldInfo.Hidden = true
			fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"readonly", "true"})
			fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"disabled", "true"})
		}

		switch fieldInfo.Type {
		case "*datastore.Key":
			fieldInfo.IsInput = true
			fieldInfo.InputType = "text"
			fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"type", fieldInfo.InputType})
		case "time.Time":
			fieldInfo.IsInput = true
			fieldInfo.InputType = "datetime-local"
			fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"type", fieldInfo.InputType})
		case "string":
			fieldInfo.IsInput = true
			fieldInfo.InputType = "text"
			fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"type", fieldInfo.InputType})
		case "float64":
			fieldInfo.IsInput = true
			fieldInfo.InputType = "number"
			fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"type", fieldInfo.InputType})
			fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"step", "any"})
			fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"pattern", `-?[0-9]*(\.[0-9]+)?`})
		case "float32":
			fieldInfo.IsInput = true
			fieldInfo.InputType = "number"
			fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"type", fieldInfo.InputType})
			fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"step", "any"})
			fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"pattern", `-?[0-9]*(\.[0-9]+)?`})
		case "int64":
			fieldInfo.IsInput = true
			fieldInfo.InputType = "number"
			fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"type", fieldInfo.InputType})
			fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"step", "1"})
		case "int32":
			fieldInfo.IsInput = true
			fieldInfo.InputType = "number"
			fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"type", fieldInfo.InputType})
			fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"step", "1"})
		case "int":
			fieldInfo.IsInput = true
			fieldInfo.InputType = "number"
			fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"type", fieldInfo.InputType})
			fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"step", "1"})
		default:
			switch f.Type.Kind() {
			case reflect.Slice, reflect.Array:
				fieldInfo.IsSlice = true
				fieldInfo.Kind = informizeType(f.Type.Elem())
				if m, ok := f.Tag.Lookup("label"); ok {
					fieldInfo.Kind.Label = m
				} else {
					fieldInfo.Kind.Label = fieldInfo.Name
				}
				fieldInfo.Kind.IsNested = true
			case reflect.Struct:
				fieldInfo.IsStruct = true
				fieldInfo.Kind = informizeType(f.Type)
				if m, ok := f.Tag.Lookup("label"); ok {
					fieldInfo.Kind.Label = m
				} else {
					fieldInfo.Kind.Label = fieldInfo.Name
				}
				fieldInfo.Kind.IsNested = true
			}
		}

		kindInfo.Fields = append(kindInfo.Fields, fieldInfo)

	}

	return kindInfo
}
