package apis

import (
	"google.golang.org/appengine/datastore"
	"strings"
	"golang.org/x/net/context"
)

type User struct {
	Email             string                 `json:"email"`
	Role              string                 `json:"role"`
	HasConfirmedEmail bool                   `json:"confirmedEmail"`
	Meta              map[string]interface{} `json:"meta"`
	//Profile           map[string]interface{} `json:"profile"`

	hash []byte
	//profile *datastore.Key
}

func GetUser(ctx context.Context, userKey *datastore.Key) (*User, error) {
	user := new(User)
	err := datastore.Get(ctx, userKey, user)
	return user, err
}

func (u *User) Language(ctx Context) string {
	if u.Meta == nil {
		return ""
	}
	if l, ok := u.Meta["lang"]; ok {
		if ls, ok := l.(string); ok {
			if _, ok := ctx.R.a.allowedTranslations[ls]; ok {
				return ls
			}
		}
	}
	return ""
}

func (u *User) SetMeta(name string, value interface{}) {
	if u.Meta == nil {
		u.Meta = map[string]interface{}{}
	}
	u.Meta[name] = value
}

func (u *User) Load(ps []datastore.Property) error {
	u.Meta = map[string]interface{}{}

	for _, prop := range ps {
		switch prop.Name {
		case "email":
			if k, ok := prop.Value.(string); ok {
				u.Email = k
			}
		case "role":
			if k, ok := prop.Value.(string); ok {
				u.Role = k
			}
			/*case "profile":
				if k, ok := prop.Value.(*datastore.Key); ok {
					u.profile = k
				}*/
		case "confirmedEmail":
			if k, ok := prop.Value.(bool); ok {
				u.HasConfirmedEmail = k
			}
		case "hash":
			if k, ok := prop.Value.([]byte); ok {
				u.hash = k
			}
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
	/*ps = append(ps, datastore.Property{
		Name:    "profile",
		Value:   u.profile,
		NoIndex: true,
	})*/
	ps = append(ps, datastore.Property{
		Name:    "hash",
		Value:   u.hash,
		NoIndex: true,
	})
	ps = append(ps, datastore.Property{
		Name:  "confirmedEmail",
		Value: u.HasConfirmedEmail,
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
