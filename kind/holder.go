package kind

import (
	"google.golang.org/appengine/datastore"
	"time"
	"errors"
	"encoding/json"
	"strings"
)

type Holder struct {
	Kind   *Kind `json:"entity"`
	user   *datastore.Key
	key    *datastore.Key
	hasKey bool

	ParsedInput       map[string]interface{}
	preparedInputData map[*Field][]datastore.Property // user input

	hasLoadedStoredData bool
	loadedStoredData    map[string][]datastore.Property // data already stored in datastore - if exists
	datastoreData       []datastore.Property            // list of properties stored in datastore - refreshed on Load or Save

	// populated on Load ... with HasAccess can check if current user is the same as createdBy
	CreatedBy *datastore.Key
	CreatedAt time.Time
}

func (h *Holder) Id() string {
	return h.key.Encode()
}

// strips fields from input
func (h *Holder) Strip(names ...string) {
	for _, name := range names {
		if f, ok := h.Kind.fields[name]; ok {
			if _, ok := h.preparedInputData[f]; ok {
				delete(h.preparedInputData, f)
			}
			if _, ok := h.ParsedInput[name]; ok {
				delete(h.ParsedInput, name)
			}
		}
	}
}

// strips all but
func (h *Holder) StripAllBut(names ...string) {
	var keepThese = map[string]bool{}
	for _, name := range names {
		keepThese[name] = true
	}
	for _, f := range h.Kind.Fields {
		if _, ok := h.preparedInputData[f]; ok {
			if _, ok := keepThese[f.Name]; !ok {
				delete(h.preparedInputData, f)
			}
		}
		if _, ok := h.ParsedInput[f.Name]; ok {
			if _, ok := keepThese[f.Name]; !ok {
				delete(h.ParsedInput, f.Name)
			}
		}
	}
}

func (h *Holder) GetValue(name string) interface{} {
	if h.hasLoadedStoredData {
		var output = map[string]interface{}{}
		if ps, ok := h.loadedStoredData[name]; ok {
			f := h.Kind.fields[name]
			for _, prop := range ps {
				output = h.appendPropertyValue(output, prop, f)
			}
		}
		for _, value := range output {
			return value
		}
	}
	var output = map[string]interface{}{}
	if f, ok := h.Kind.fields[name]; ok {
		for _, prop := range h.preparedInputData[f] {
			output = h.appendPropertyValue(output, prop, f)
		}
	}
	return output[name]
}

func (h *Holder) ParseInput(body []byte) error {
	return json.Unmarshal(body, &h.ParsedInput)
}

func (h *Holder) Prepare() error {
	for _, f := range h.Kind.Fields {
		if value, ok := h.ParsedInput[f.Name]; ok {
			if f.Kind != nil {
				// IF KIND SPECIFIED
				if f.Multiple {
					if arr, ok := value.([]interface{}); ok {
						for _, v := range arr {
							bs, _ := json.Marshal(v)
							h.preparedInputData[f] = append(h.preparedInputData[f], datastore.Property{
								Value:    string(bs),
								NoIndex:  true,
								Multiple: f.Multiple,
								Name:     f.Name,
							})
						}
					} else if arr, ok := value.([]string); ok {
						for _, v := range arr {
							bs, _ := json.Marshal(v)
							h.preparedInputData[f] = append(h.preparedInputData[f], datastore.Property{
								Value:    string(bs),
								NoIndex:  true,
								Multiple: f.Multiple,
								Name:     f.Name,
							})
						}
					}
				} else {
					bs, _ := json.Marshal(value)
					h.preparedInputData[f] = append(h.preparedInputData[f], datastore.Property{
						Value:    string(bs),
						NoIndex:  true,
						Multiple: f.Multiple,
						Name:     f.Name,
					})
				}
			} else {
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
	value = field.Output(value)

	if field != nil && field.Kind != nil {
		if body, ok := value.(string); ok {
			var fdst map[string]interface{}
			if err := json.Unmarshal([]byte(body), &fdst); err == nil {
				value = fdst
			}
		}
	}

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

func (h *Holder) SetKey(k *datastore.Key) {
	if k != nil {
		h.key = k
		h.hasKey = true
	}
}

func (h *Holder) GetKey() *datastore.Key {
	return h.key
}

func (h *Holder) Load(ps []datastore.Property) error {
	h.hasLoadedStoredData = true
	h.datastoreData = ps
	for _, prop := range ps {
		h.loadedStoredData[prop.Name] = append(h.loadedStoredData[prop.Name], prop)
		if prop.Name == "meta.createdBy" && prop.Value != nil {
			h.CreatedBy = prop.Value.(*datastore.Key)
		} else if prop.Name == "meta.createdAt" && prop.Value != nil {
			h.CreatedAt = prop.Value.(time.Time)
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
	}

	// set meta tags
	var now = time.Now()
	h.datastoreData = append(h.datastoreData, datastore.Property{
		Name:  "meta.updatedAt",
		Value: now,
	})
	if h.hasLoadedStoredData {
		if metaCreatedAt, ok := h.loadedStoredData["meta.createdAt"]; ok {
			h.CreatedAt = metaCreatedAt[0].Value.(time.Time)
			h.datastoreData = append(h.datastoreData, metaCreatedAt[0])
		}
		if metaCreatedBy, ok := h.loadedStoredData["meta.createdBy"]; ok {
			h.CreatedBy = metaCreatedBy[0].Value.(*datastore.Key)
			h.datastoreData = append(h.datastoreData, metaCreatedBy[0])
		}
	} else {
		h.datastoreData = append(h.datastoreData, datastore.Property{
			Name:  "meta.createdAt",
			Value: now,
		})
		h.CreatedAt = now
		h.datastoreData = append(h.datastoreData, datastore.Property{
			Name:  "meta.createdBy",
			Value: h.user,
		})
		h.CreatedBy = h.user
	}

	ps = h.datastoreData

	return ps, nil
}
