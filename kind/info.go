package kind

import (
	"reflect"
	"net/http"
)

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
	IsSlice    bool            `json:"is_slice,omitempty"`
	IsStruct   bool            `json:"is_struct,omitempty"`
	IsSelect   bool            `json:"is_select,omitempty"`
	IsTextArea bool            `json:"is_text_area,omitempty"`
	InputType  string          `json:"input_type,omitempty"`
	Struct     interface{}     `json:"struct,omitempty"`
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
func (k *Kind) Info() *KindInfo {
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

		//k.checkFields()

		fieldInfo := informizeType(k.Type)

		k.info = fieldInfo
	}
	return k.info
}

type KindInfo struct {
	Name   string       `json:"name,omitempty"`
	Type   string       `json:"type,omitempty"`
	Fields []*FieldInfo `json:"fields,omitempty"`
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
		fieldInfo.Label = f.Tag.Get("label")
		fieldInfo.Name = f.Tag.Get("json")

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
			case reflect.Struct:
				fieldInfo.IsStruct = true
				fieldInfo.Kind = informizeType(f.Type)
			}
		}
		/*
					if len(f.MetaField) > 0 {
						infoField.Attributes = append(infoField.Attributes, InfoFieldAttr{"readonly", "true"})
						infoField.Attributes = append(infoField.Attributes, InfoFieldAttr{"disabled", "true"})
						infoField.Hidden = true
						infoField.Meta = f.MetaField
					}*/

		kindInfo.Fields = append(kindInfo.Fields, fieldInfo)

	}

	return kindInfo
}
