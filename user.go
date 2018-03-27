package apis

import (
	"google.golang.org/appengine/datastore"
	"strings"
	"golang.org/x/net/context"
	"github.com/ales6164/apis/kind"
)

type User struct {
	Email   string                 `json:"email"`
	Role    string                 `json:"role"`
	Meta    map[string]interface{} `json:"meta"`
	Profile map[string]interface{} `json:"profile"`

	hash    []byte
	profile *datastore.Key
}

func (u *User) SetMeta(name string, value interface{}) {
	if u.Meta == nil {
		u.Meta = map[string]interface{}{}
	}
	u.Meta[name] = value
}

func (u *User) LoadProfile(ctx context.Context, k *kind.Kind) (map[string]interface{}, error) {
	if u.profile != nil {
		h, err := k.Get(ctx, u.profile)
		if err != nil {
			return nil, err
		}
		u.Profile = h.Output()
		delete(u.Profile, "meta")
		return u.Profile, nil
	}
	return nil, ErrUserProfileDoesNotExist
}

func (u *User) Load(ps []datastore.Property) error {
	u.Meta = map[string]interface{}{}

	for _, prop := range ps {
		switch prop.Name {
		case "email":
			u.Email = prop.Value.(string)
		case "role":
			u.Role = prop.Value.(string)
		case "profile":
			if k, ok := prop.Value.(*datastore.Key); ok {
				u.profile = k
			}
		case "hash":
			u.hash = prop.Value.([]byte)
		default:
			spl := strings.Split(prop.Name, ".")
			if len(spl) > 1 {
				if spl[0] == "meta" {
					u.Meta[spl[1]] = prop.Value
				}
			}
		}
	}

	return nil
}

func (u *User) Save() ([]datastore.Property, error) {
	var ps []datastore.Property

	ps = append(ps, datastore.Property{
		Name:  "email",
		Value: u.Email,
	})
	ps = append(ps, datastore.Property{
		Name:  "role",
		Value: u.Role,
	})
	ps = append(ps, datastore.Property{
		Name:    "profile",
		Value:   u.profile,
		NoIndex: true,
	})
	ps = append(ps, datastore.Property{
		Name:    "hash",
		Value:   u.hash,
		NoIndex: true,
	})
	if u.Meta != nil {
		for k, v := range u.Meta {
			ps = append(ps, datastore.Property{
				Name:     "meta." + k,
				Value:    v,
				NoIndex:  true,
				Multiple: true,
			})
		}
	}

	return ps, nil
}
