package kind

import (
	"net/http"
	"reflect"
	"strings"
)

type InfoFieldAttr struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

func (k *Kind) Info() *KindInfo {
	if k.info == nil {
		kindInfo := new(KindInfo)
		kindInfo.Meta = new(MetaInfo)
		kindInfo.typ = k.Type
		kindInfo.Type = kindInfo.typ.String()
		kindInfo = kindInfo.informizeType(kindInfo, nil)
		kindInfo.Name = k.Name
		kindInfo.Label = k.Label
		kindInfo.RelativePath = k.path
		for _, m := range k.methods {
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
	Meta         *MetaInfo    `json:"meta"`
	TableColumns []*FieldInfo `json:"tableColumns,omitempty"`

	typ reflect.Type

	IsNested     bool   `json:"isNested,omitempty"`
	Label        string `json:"label,omitempty"`
	RelativePath string `json:"relativePath,omitempty"`
	HasGet       bool   `json:"get,omitempty"`
	HasPost      bool   `json:"post,omitempty"`
	HasPut       bool   `json:"put,omitempty"`
	HasDelete    bool   `json:"delete,omitempty"`
}

type MetaInfo struct {
	UpdatedAtField *FieldInfo `json:"updatedAt,omitempty"`
	CreatedAtField *FieldInfo `json:"createdAt,omitempty"`
	UpdatedByField *FieldInfo `json:"updatedBy,omitempty"`
	CreatedByField *FieldInfo `json:"createdBy,omitempty"`
	IdField        *FieldInfo `json:"id,omitempty"`
}

type FieldInfo struct {
	Label   string          `json:"label,omitempty"`
	Name    string          `json:"name,omitempty"`
	JsonTag string          `json:"jsonTag,omitempty"`
	Table   *InfoFieldTable `json:"table,omitempty"`

	Meta       string          `json:"meta,omitempty"`
	Hidden     bool            `json:"hidden,omitempty"` // only in on create window
	Attributes []InfoFieldAttr `json:"attributes,omitempty"`
	Type       string          `json:"type,omitempty"`
	InputType  string          `json:"inputType,omitempty"`
	TagStart   string          `json:"tagStart,omitempty"`
	TagEnd     string          `json:"tagEnd,omitempty"`

	Kind *KindInfo `json:"kind,omitempty"`
}

type InfoFieldTable struct {
	Display bool `json:"display,omitempty"`
	NoSort  bool `json:"noSort,omitempty"`
}

func (fi *FieldInfo) generateTag(name string) {
	// name label type
	fi.TagStart = "<" + name + " type=" + fi.InputType + " name=" + fi.JsonTag + " label='" + fi.Label + "'"
	for _, attr := range fi.Attributes {
		fi.TagStart += " " + attr.Name + "='" + attr.Value + "'"
	}
	fi.TagStart += ">"
	fi.TagEnd = "</" + name + ">"
}

func (mainKind *KindInfo) informizeType(kindInfo *KindInfo, parentField *FieldInfo) *KindInfo {
	for i := 0; i < kindInfo.typ.NumField(); i++ {
		f := kindInfo.typ.Field(i)

		fieldInfo := new(FieldInfo)
		fieldInfo.Type = f.Type.String()

		calcTable := parentField == nil || parentField.Table != nil && parentField.Table.Display

		fieldInfo.Name = f.Name

		if m, ok := f.Tag.Lookup("json"); ok {
			fieldInfo.JsonTag = strings.TrimSpace(strings.Split(m, ",")[0])
		} else {
			fieldInfo.JsonTag = f.Name
		}

		if m, ok := f.Tag.Lookup("label"); ok {
			fieldInfo.Label = m
		} else {
			fieldInfo.Label = f.Name
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
					} else if v == "id" {
						mainKind.Meta.IdField = fieldInfo
					} else if v == "updatedby" {
						mainKind.Meta.UpdatedByField = fieldInfo
					} else if v == "createdat" {
						mainKind.Meta.CreatedAtField = fieldInfo
					} else if v == "createdby" {
						mainKind.Meta.CreatedByField = fieldInfo
					}
				}
			}
		}

		if len(fieldInfo.Meta) == 0 && fieldInfo.Table != nil {
			mainKind.TableColumns = append(mainKind.TableColumns, fieldInfo)
		}

		switch fieldInfo.Type {
		case "string", "*datastore.Key":
			fieldInfo.InputType = "text"
			fieldInfo.generateTag("entry-field-textfield")
		case "time.Time":
			fieldInfo.InputType = "datetime-local"
			fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"transform", "date"})
			fieldInfo.generateTag("entry-field-textfield")
		case "bool":
			fieldInfo.InputType = "checkbox"
			fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"transform", "bool"})
			fieldInfo.generateTag("entry-field-toggle")
		case "float64", "float32":
			fieldInfo.InputType = "number"
			fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"transform", "double"})
			fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"step", "any"})
			fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"pattern", `-?[0-9]*(\.[0-9]+)?`})
			fieldInfo.generateTag("entry-field-textfield")
		case "int64", "int", "int32":
			fieldInfo.InputType = "number"
			fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"transform", "number"})
			fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"step", "1"})
			fieldInfo.generateTag("entry-field-textfield")
		default:
			switch f.Type.Kind() {
			case reflect.Slice, reflect.Array:
				fieldInfo.Kind = new(KindInfo)
				fieldInfo.Kind.typ = f.Type.Elem()
				fieldInfo.Kind.Type = fieldInfo.Kind.typ.String()
				fieldInfo.Kind.Name = fieldInfo.Kind.typ.Name()
				if m, ok := f.Tag.Lookup("label"); ok {
					fieldInfo.Kind.Label = m
				} else {
					fieldInfo.Kind.Label = fieldInfo.Kind.Name
				}
				if kindInfo.typ.Kind() != reflect.Struct {
					fieldInfo.Kind = mainKind.informizeType(fieldInfo.Kind, fieldInfo)
					fieldInfo.Kind.IsNested = true
				}
			case reflect.Struct:
				fieldInfo.Kind = new(KindInfo)
				fieldInfo.Kind.typ = f.Type
				fieldInfo.Kind.Type = fieldInfo.Kind.typ.String()
				fieldInfo.Kind.Name = fieldInfo.Kind.typ.Name()
				if m, ok := f.Tag.Lookup("label"); ok {
					fieldInfo.Kind.Label = m
				} else {
					fieldInfo.Kind.Label = fieldInfo.Kind.Name
				}
				fieldInfo.Kind = mainKind.informizeType(fieldInfo.Kind, fieldInfo)
				fieldInfo.Kind.IsNested = true
			}
		}

		kindInfo.Fields = append(kindInfo.Fields, fieldInfo)

	}

	return kindInfo
}
