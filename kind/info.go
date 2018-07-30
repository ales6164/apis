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

func (k *Kind) Info() *KindInfo {
	if k.info == nil {
		kindInfo := new(KindInfo)
		kindInfo.Meta = new(MetaInfo)
		kindInfo.typ = k.t
		kindInfo.Type = kindInfo.typ.String()
		kindInfo.parse()
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
	IsSlice bool            `json:"isSlice,omitempty"`

	typ reflect.Type

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

func (fieldInfo *FieldInfo) generateTag(name string) {
	// name label type
	fieldInfo.TagStart = "<" + name + " type=" + fieldInfo.InputType + " name=" + fieldInfo.JsonTag + " label='" + fieldInfo.Label + "'"
	for _, attr := range fieldInfo.Attributes {
		fieldInfo.TagStart += " " + attr.Name + "='" + attr.Value + "'"
	}
	fieldInfo.TagStart += ">"
	fieldInfo.TagEnd = "</" + name + ">"
}

func (info *KindInfo) parse() {
	for i := 0; i < info.typ.NumField(); i++ {
		f := info.typ.Field(i)
		info.parseField(f, nil)
	}
}

func (info *KindInfo) parseField(structField reflect.StructField, parentField *FieldInfo) {
	fieldInfo := new(FieldInfo)
	fieldInfo.typ = structField.Type
	fieldInfo.Type = structField.Type.String()
	fieldInfo.Name = structField.Name

	if m, ok := structField.Tag.Lookup("json"); ok {
		fieldInfo.JsonTag = strings.TrimSpace(strings.Split(m, ",")[0])
	} else {
		fieldInfo.JsonTag = structField.Name
	}

	if m, ok := structField.Tag.Lookup("label"); ok {
		fieldInfo.Label = m
	} else {
		fieldInfo.Label = structField.Name
	}

	var calcTable = parentField == nil || parentField.Table != nil && parentField.Table.Display
	if calcTable {
		if m, ok := structField.Tag.Lookup("table"); ok {
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

	if m, ok := structField.Tag.Lookup("apis"); ok {
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
					info.Meta.UpdatedAtField = fieldInfo
				} else if v == "id" {
					info.Meta.IdField = fieldInfo
				} else if v == "updatedby" {
					info.Meta.UpdatedByField = fieldInfo
				} else if v == "createdat" {
					info.Meta.CreatedAtField = fieldInfo
				} else if v == "createdby" {
					info.Meta.CreatedByField = fieldInfo
				}
			}
		}
	}

	if len(fieldInfo.Meta) == 0 && fieldInfo.Table != nil {
		info.TableColumns = append(info.TableColumns, fieldInfo)
	}

	fieldInfo.parseType()

	info.Fields = append(info.Fields, fieldInfo)
}

func (fieldInfo *FieldInfo) parseType() {
	switch fieldInfo.typ {
	case stringType, keyType:
		fieldInfo.InputType = "text"
		fieldInfo.generateTag("entry-field-textfield")
	case timeType:
		fieldInfo.InputType = "datetime-local"
		fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"transform", "date"})
		fieldInfo.generateTag("entry-field-textfield")
	case boolType:
		fieldInfo.InputType = "checkbox"
		fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"transform", "bool"})
		fieldInfo.generateTag("entry-field-toggle")
	case float64Type, float32Type:
		fieldInfo.InputType = "number"
		fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"transform", "double"})
		fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"step", "any"})
		fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"pattern", `-?[0-9]*(\.[0-9]+)?`})
		fieldInfo.generateTag("entry-field-textfield")
	case int64Type, intType, int32Type:
		fieldInfo.InputType = "number"
		fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"transform", "number"})
		fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"step", "1"})
		fieldInfo.generateTag("entry-field-textfield")
	default:
		switch fieldInfo.typ.Kind() {
		case reflect.Slice, reflect.Array:
			// is slice
			fieldInfo.IsSlice = true
			fieldInfo.Attributes = append(fieldInfo.Attributes, InfoFieldAttr{"slice", "true"})

			typ := fieldInfo.typ.Elem()
			if typ.Kind() == reflect.Struct {
				fieldInfo.Kind = &KindInfo{
					typ:      typ,
					Name:     typ.Name(),
					Type:     typ.String(),
					IsNested: true,
				}

				for i := 0; i < typ.NumField(); i++ {
					f := typ.Field(i)
					fieldInfo.Kind.parseField(f, fieldInfo)
				}
			} else {
				fieldInfo.typ = typ
				fieldInfo.parseType()
			}
		case reflect.Struct:
			fieldInfo.Kind = &KindInfo{
				typ:      fieldInfo.typ,
				Name:     fieldInfo.typ.Name(),
				Type:     fieldInfo.typ.String(),
				IsNested: true,
			}

			for i := 0; i < fieldInfo.typ.NumField(); i++ {
				f := fieldInfo.typ.Field(i)
				fieldInfo.Kind.parseField(f, fieldInfo)
			}

		default:
			//panic(errors.New("unrecognized field type"))
		}
	}
}
