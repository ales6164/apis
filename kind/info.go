package kind

import (
	"reflect"
	"net/http"
	"strings"
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

		kindInfo := new(KindInfo)
		kindInfo.Meta = new(MetaInfo)
		kindInfo.typ = k.Type
		kindInfo.Type = kindInfo.typ.String()
		kindInfo.Name = kindInfo.typ.Name()
		kindInfo.Label = k.ui.Label
		kindInfo.LabelMany = k.ui.Label
		kindInfo = kindInfo.informizeType(kindInfo, nil)
		kindInfo.SearchIndex = k.SearchType.Name()
		kindInfo.RelativePath = k.ui.relativePath
		for _, m := range k.ui.methods {
			switch m {
			case http.MethodGet:
				kindInfo.HasGet = true
			case http.MethodPost:
				kindInfo.HasPost = true
			case http.MethodPut:
				kindInfo.HasPut = true
			case http.MethodDelete:
				kindInfo.HasDelete = true
			}
		}

		k.info = kindInfo
	}
	return k.info
}

type KindInfo struct {
	Name         string       `json:"name,omitempty"`
	Type         string       `json:"type,omitempty"`
	Fields       []*FieldInfo `json:"fields,omitempty"`
	Meta         *MetaInfo    `json:"meta,omitempty"`
	TableColumns []*FieldInfo `json:"table_columns,omitempty"`

	typ reflect.Type

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

type MetaInfo struct {
	UpdatedAtField *FieldInfo `json:"updated_at_field,omitempty"`
}

type FieldInfo struct {
	Label string          `json:"label,omitempty"`
	Name  string          `json:"name,omitempty"`
	Table *InfoFieldTable `json:"table,omitempty"`

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

type InfoFieldTable struct {
	Display bool `json:"display,omitempty"`
	NoSort  bool `json:"no_sort,omitempty"`
}

func (mainKind *KindInfo) informizeType(kindInfo *KindInfo, parentField *FieldInfo) *KindInfo {
	for i := 0; i < kindInfo.typ.NumField(); i++ {
		f := kindInfo.typ.Field(i)

		fieldInfo := new(FieldInfo)
		fieldInfo.Type = f.Type.String()

		calcTable := parentField == nil || parentField.Table != nil && parentField.Table.Display

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

		if parentField != nil {
			fieldInfo.Label = kindInfo.Label + "." + fieldInfo.Label
		}

		if calcTable {
			if m, ok := f.Tag.Lookup("table"); ok {
				ms := strings.Split(m, ",")
			tableLoop:
				for i, msv := range ms {
					msv = strings.TrimSpace(msv)
					switch i {
					case 0:
						if msv == "-" {
							break tableLoop
						} else {
							fieldInfo.Table = new(InfoFieldTable)
							fieldInfo.Table.Display = true
						}
					case 1:
						fieldInfo.Table.NoSort = msv == "nosort"
					}
				}
			} else {
				fieldInfo.Table = new(InfoFieldTable)
				fieldInfo.Table.Display = true
			}
		}

		if m, ok := f.Tag.Lookup("apis"); ok {
			fieldInfo.Meta = m
			fieldInfo.Hidden = true
			fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"readonly", "true"})
			fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"disabled", "true"})

			for n, v := range strings.Split(m, ",") {
				v = strings.TrimSpace(v)
				v = strings.ToLower(v)
				switch n {
				case 0:
					if v == "updatedat" {
						mainKind.Meta.UpdatedAtField = fieldInfo
					}
				}
			}
		}

		if len(fieldInfo.Meta) == 0 && fieldInfo.Table != nil {
			mainKind.TableColumns = append(mainKind.TableColumns, fieldInfo)
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
				fieldInfo.Kind = new(KindInfo)
				fieldInfo.Kind.typ = f.Type.Elem()
				fieldInfo.Kind.Type = fieldInfo.Kind.typ.String()
				fieldInfo.Kind.Name = fieldInfo.Kind.typ.Name()
				if m, ok := f.Tag.Lookup("label"); ok {
					fieldInfo.Kind.Label = m
				} else {
					fieldInfo.Kind.Label = fieldInfo.Name
				}
				fieldInfo.Kind = mainKind.informizeType(fieldInfo.Kind, fieldInfo)
				fieldInfo.Kind.IsNested = true
			case reflect.Struct:
				fieldInfo.IsStruct = true
				fieldInfo.Kind = new(KindInfo)
				fieldInfo.Kind.typ = f.Type
				fieldInfo.Kind.Type = fieldInfo.Kind.typ.String()
				fieldInfo.Kind.Name = fieldInfo.Kind.typ.Name()
				if m, ok := f.Tag.Lookup("label"); ok {
					fieldInfo.Kind.Label = m
				} else {
					fieldInfo.Kind.Label = fieldInfo.Name
				}
				fieldInfo.Kind = mainKind.informizeType(fieldInfo.Kind, fieldInfo)
				fieldInfo.Kind.IsNested = true
			}
		}

		kindInfo.Fields = append(kindInfo.Fields, fieldInfo)

	}

	return kindInfo
}
