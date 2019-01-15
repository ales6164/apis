package apis

import (
	"errors"
	"github.com/ales6164/apis/kind"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"net/http"
	"strings"
	"time"
)

type Collector struct {
	defaultContext context.Context
	ctx            Context
	collection     *collection
	collections    []*collection
	namespace      string
}

type collection struct {
	hasUncommittedChanges bool
	entryKey              *datastore.Key
	entry                 *Entry
	kind                  kind.Kind
}

// datastore entry descriptor
// only in default namespace
type Entry struct {
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	CreatedBy       *datastore.Key `json:"createdBy"`
	UpdatedBy       *datastore.Key `json:"updatedBy"`
	ParentNamespace string         `json:"-"`
	Namespace       string         `json:"-"` // every entry should have unique namespace --- or maybe auto generated if needed
}

func NewCollector(ctx Context) *Collector {
	return &Collector{
		defaultContext: appengine.NewContext(ctx.r),
		ctx:            ctx,
	}
}

func (c *Collector) NewEntryKey(kindName string, stringId string, intId int64) *datastore.Key {
	return datastore.NewKey(c.defaultContext, "_entry_"+kindName, stringId, intId, nil)
}

func (c *Collector) AppendCollection(k kind.Kind, id string) error {
	if c.collection != nil && c.collection.hasUncommittedChanges {
		return errors.New("uncommitted changes")
	}
	var key *datastore.Key
	if len(id) > 0 {
		var err error
		key, err = datastore.DecodeKey(id)
		if err == nil {
			// entry key from decoded key
			key = c.NewEntryKey(k.Name(), key.StringID(), key.IntID())
		} else {
			key = c.NewEntryKey(k.Name(), id, 0)
		}
	}
	col := &collection{
		kind:     k,
		entryKey: key,
	}
	c.collections = append(c.collections, col)
	c.collection = col
	return nil
}

// also check parent groupId inside entry
func (c *Collector) RetrieveEntry() error {
	if c.collection.entryKey != nil {
		c.collection.entry = new(Entry)
		err := datastore.Get(c.defaultContext, c.collection.entryKey, c.collection.entry)
		if err != nil {
			if err == datastore.ErrNoSuchEntity {
				// create entry value as the entry key is anonymous
				c.collection.hasUncommittedChanges = true
				c.collection.entry = &Entry{
				// TODO: add values
				}
			} else {
				return err
			}
		} else if len(c.collections) > 1 {
			// TODO: check parent namespace
			if c.collections[len(c.collections)-2].entry.Namespace == c.collection.entry.ParentNamespace {
				// current value namespace matches parent


			} else {
				return errors.New(http.StatusText(http.StatusNotFound))
			}
		}
	}
	return nil
}

// for every fetch retrieve document info and preload everything to then just get the entry when needed
func (c *Collector) Fetch(k kind.Kind, id string) (*Collector, error) {
	err := c.AppendCollection(k, id)
	if err != nil {
		return c, err
	}
	err = c.RetrieveEntry()

	return c, err
}

func (a *Apis) serve(w http.ResponseWriter, r *http.Request) {
	path := getPath(r.URL.Path)

	rules := a.Rules

	ctx := a.NewContext(w, r)
	collector := NewCollector(ctx)

	// analyse path in pairs
	for i := 0; i < len(path); i += 2 {
		// get collection kind and match it to rules
		if k, ok := a.kinds[path[i]]; ok {
			if rules, ok = rules.Match[k]; ok {
				// got latest rules
				var err error

				if (i + 1) < len(path) {
					collector, err = collector.Fetch(k, path[i+1])
				} else {
					collector, err = collector.Fetch(k, "")
				}

				if err != nil {
					ctx.PrintError(err.Error(), http.StatusBadRequest)
					return
				}
				continue
			}
		}
		ctx.PrintError(http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	var ok bool
	if ctx, ok = ctx.WithSession(); !ok {
		return
	}

	if collector.collection.entryKey != nil {
		switch r.Method {
		case http.MethodGet:
			if ok := ctx.HasRole(rules.ReadOnly, rules.ReadWrite, rules.FullControl); ok {
				doc, err := collector.collection.kind.Doc(ctx, collector.collection.entryKey).Get()
				if err != nil {
					ctx.PrintError(err.Error(), http.StatusInternalServerError)
					return
				}
				ctx.PrintJSON(collector.collection.kind.Data(doc), http.StatusOK)
			} else {
				ctx.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
			}
		}
	} else {
		switch r.Method {
		case http.MethodGet:
		case http.MethodPost:
			/*if ok := c.HasRole(rules.ReadWrite, rules.FullControl); ok {
				doc, err := lastKind.Doc(ctx, nil).Add(c.Body())
				if err != nil {
					c.PrintError(err.Error(), http.StatusInternalServerError)
					return
				}
				c.PrintJSON(lastKind.Data(doc), http.StatusOK)
			} else {
				c.PrintError(http.StatusText(http.StatusForbidden), http.StatusForbidden)
			}*/
		}
	}

}

func getPath(p string) []string {
	if p[:1] == "/" {
		p = p[1:]
	}
	return strings.Split(p, "/")
}
