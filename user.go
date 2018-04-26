package apis

import (
	"google.golang.org/appengine/datastore"
	"strings"
	"golang.org/x/net/context"
)

type User struct {
	Id                *datastore.Key         `datastore:"-" json:"id"`
	Email             string                 `datastore:"email" json:"email"`
	Role              string                 `datastore:"role" json:"role"`
	IsAnonymous       bool                   `datastore:"anonymous" json:"anonymous"`
	HasConfirmedEmail bool                   `datastore:"confirmedEmail" json:"confirmedEmail"`
	Meta              map[string]interface{} `datastore:"meta" json:"meta"`
	Hash              []byte                 `datastore:"hash" json:"-"`
}

func GetUser(ctx context.Context, userKey *datastore.Key) (*User, error) {
	user := new(User)
	err := datastore.Get(ctx, userKey, user)
	return user, err
}

func (u *User) Language(ctx Context, defaultLang string) string {
	if u.Meta == nil {
		return defaultLang
	}
	if l, ok := u.Meta["lang"]; ok {
		if ls, ok := l.(string); ok {
			if _, ok := ctx.R.a.allowedTranslations[ls]; ok {
				return ls
			}
		}
	}
	return defaultLang
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
		case "confirmedEmail":
			if k, ok := prop.Value.(bool); ok {
				u.HasConfirmedEmail = k
			}
		case "hash":
			if k, ok := prop.Value.([]byte); ok {
				u.Hash = k
			}
		case "anonymous":
			if k, ok := prop.Value.(bool); ok {
				u.IsAnonymous = k
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
	ps = append(ps, datastore.Property{
		Name:    "hash",
		Value:   u.Hash,
		NoIndex: true,
	})
	ps = append(ps, datastore.Property{
		Name:  "confirmedEmail",
		Value: u.HasConfirmedEmail,
	})
	ps = append(ps, datastore.Property{
		Name:  "anonymous",
		Value: u.IsAnonymous,
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
