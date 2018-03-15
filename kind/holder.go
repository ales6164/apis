package kind

import (
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"time"
	"errors"
	"encoding/json"
	"strings"
)

type Holder struct {
	Kind    *Kind `json:"entity"`
	user    *datastore.Key
	context context.Context
	key     *datastore.Key

	preparedInputData   map[*Field][]datastore.Property // user input
	hasLoadedStoredData bool
	loadedStoredData    map[string][]datastore.Property // data already stored in datastore - if exists
	datastoreData       []datastore.Property            // list of properties stored in datastore - refreshed on Load or Save

	// populated on Load ... with HasAccess can check if current user is the same as createdBy
	createdBy *datastore.Key

	isOldVersion bool // when updating entity we want to also update old entry meta.
}

func (h *Holder) SetContext(ctx context.Context) {
	h.context = ctx
}

func (h *Holder) ParseInput(body []byte) error {
	var m map[string]interface{}
	err := json.Unmarshal(body, &m)
	if err != nil {
		return err
	}

	for _, f := range h.Kind.Fields {

		// check for input
		if value, ok := m[f.Name]; ok {

			props, err := f.Parse(value)
			if err != nil {
				return err
			}

			h.preparedInputData[f] = props
		} else {
			names := strings.Split(f.Name, ".")
			if _, ok := m[names[0]]; ok {

				var endValue = m[names[0]]
				for i := 1; i < len(names); i++ {
					if nestedMap, ok := endValue.(map[string]interface{}); ok {
						endValue = nestedMap[names[i]]
					} else {
						continue
					}
				}
				props, err := f.Parse(value)
				if err != nil {
					return err
				}
				h.preparedInputData[f] = props
			}
		}
	}
	return nil
}

// used in parsing function with field that has Kind specified
func (h *Holder) Parse(m map[string]interface{}) error {
	for _, f := range h.Kind.Fields {

		// check for input
		if value, ok := m[f.Name]; ok {

			props, err := f.Parse(value)
			if err != nil {
				return err
			}

			h.preparedInputData[f] = props
		} else {
			names := strings.Split(f.Name, ".")
			if _, ok := m[names[0]]; ok {

				var endValue = m[names[0]]
				for i := 1; i < len(names); i++ {
					if nestedMap, ok := endValue.(map[string]interface{}); ok {
						endValue = nestedMap[names[i]]
					} else {
						continue
					}
				}
				props, err := f.Parse(value)
				if err != nil {
					return err
				}
				h.preparedInputData[f] = props
			}
		}
	}
	return nil
}

// appends value
func (h *Holder) appendValue(dst interface{}, field *Field, value interface{}, multiple bool) interface{} {
	value = field.Output(h.context, value)
	if multiple {
		if dst == nil {
			dst = []interface{}{}
		}

		dst = append(dst.([]interface{}), value)
	} else {
		dst = value
	}
	return dst
}

// appends property to dst; it can return a flat object or structured
func (h *Holder) appendPropertyValue(dst map[string]interface{}, prop datastore.Property, field *Field) map[string]interface{} {

	names := strings.Split(prop.Name, ".")
	if len(names) > 1 {
		prop.Name = strings.Join(names[1:], ".")
		if _, ok := dst[names[0]].(map[string]interface{}); !ok {
			dst[names[0]] = map[string]interface{}{}
		}
		dst[names[0]] = h.appendPropertyValue(dst[names[0]].(map[string]interface{}), prop, field)
	} else {
		dst[names[0]] = h.appendValue(dst[names[0]], field, prop.Value, prop.Multiple)
	}

	return dst
}

func (h *Holder) Output() map[string]interface{} {
	var output = map[string]interface{}{}

	// range over data. Value can be single value or if the field it Multiple then it's an array
	for _, prop := range h.datastoreData {
		output = h.appendPropertyValue(output, prop, h.Kind.fields[prop.Name])
	}

	if h.key != nil {
		output["id"] = h.key.Encode()
	}

	return output
}

func (h *Holder) Load(ps []datastore.Property) error {
	h.hasLoadedStoredData = true
	h.datastoreData = ps
	for _, prop := range ps {
		h.loadedStoredData[prop.Name] = append(h.loadedStoredData[prop.Name], prop)
		if prop.Name == "meta.createdBy" && prop.Value != nil {
			h.createdBy = prop.Value.(*datastore.Key)
		}
	}
	return nil
}

func (h *Holder) Save() ([]datastore.Property, error) {
	var ps []datastore.Property

	h.datastoreData = []datastore.Property{}

	// check if required field are there
	for _, f := range h.Kind.Fields {

		var inputProperties = h.preparedInputData[f]
		var loadedProperties = h.loadedStoredData[f.Name]

		var toSaveProps []datastore.Property

		if len(inputProperties) != 0 {
			toSaveProps = append(toSaveProps, inputProperties...)
		} else if len(loadedProperties) != 0 {
			toSaveProps = append(toSaveProps, loadedProperties...)
		} else if f.IsRequired {
			return nil, errors.New("field " + f.Name + " required")
		}

		h.datastoreData = append(h.datastoreData, toSaveProps...)

		// save search field
		// TODO:
		/*if field.toSearchFieldConvertFunc != nil {
			if err := field.toSearchFieldConvertFunc(holder, toSaveProps...); err != nil {
				return nil, err
			}
		}*/
	}

	// set meta tags
	var now = time.Now()
	h.datastoreData = append(h.datastoreData, datastore.Property{
		Name:  "meta.updatedAt",
		Value: now,
	})
	h.datastoreData = append(h.datastoreData, datastore.Property{
		Name:  "meta.status",
		Value: "active",
	})
	if h.hasLoadedStoredData {
		if metaCreatedAt, ok := h.loadedStoredData["meta.createdAt"]; ok {
			h.datastoreData = append(h.datastoreData, metaCreatedAt[0])
		}
		if metaCreatedBy, ok := h.loadedStoredData["meta.createdBy"]; ok {
			h.datastoreData = append(h.datastoreData, metaCreatedBy[0])
		}
		if metaVersion, ok := h.loadedStoredData["meta.version"]; ok {
			var version = metaVersion[0]
			version.Value = version.Value.(int64) + 1
			h.datastoreData = append(h.datastoreData, version)
		}
	} else {
		h.datastoreData = append(h.datastoreData, datastore.Property{
			Name:  "meta.createdAt",
			Value: now,
		})
		h.datastoreData = append(h.datastoreData, datastore.Property{
			Name:  "meta.createdBy",
			Value: h.user,
		})
		h.datastoreData = append(h.datastoreData, datastore.Property{
			Name:  "meta.version",
			Value: int64(0),
		})
	}

	h.datastoreData = append(h.datastoreData, datastore.Property{
		Name:  "meta.updatedBy",
		Value: h.user,
	})

	ps = h.datastoreData

	return ps, nil
}

type HolderOld struct {
	data *Holder
	key  *datastore.Key
}

func (h *Holder) OldHolder(replacementKey *datastore.Key) *HolderOld {
	var old = &HolderOld{
		data: h,
		key:  replacementKey,
	}
	return old
}

func (h *HolderOld) Load(ps []datastore.Property) error {
	return nil
}

func (h *HolderOld) Save() ([]datastore.Property, error) {
	var ps []datastore.Property

	h.data.loadedStoredData["meta.status"] = []datastore.Property{{
		Name:  "meta.status",
		Value: nil,
	}}

	for _, props := range h.data.loadedStoredData {
		ps = append(ps, props...)
	}

	return ps, nil
}
