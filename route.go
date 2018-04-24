package apis

import (
	"github.com/ales6164/apis/kind"
	"net/http"
	"google.golang.org/appengine/datastore"
	"strconv"
	"github.com/asaskevich/govalidator"
	"strings"
	"encoding/json"
	"golang.org/x/net/context"
	"github.com/ales6164/apis/errors"
	"bytes"
	"google.golang.org/appengine/search"
	"reflect"
)

type Route struct {
	a    *Apis
	kind *kind.Kind
	path string

	listeners map[string]Listener
	searchListener func(ctx Context, query string) ([]interface{}, error)
	roles map[Role]bool

	methods []string

	get    http.HandlerFunc
	post   http.HandlerFunc
	put    http.HandlerFunc
	delete http.HandlerFunc
}

type Listener func(ctx Context, h *kind.Holder) error

const (
	BeforeRead   = "beforeGet"
	BeforeCreate = "beforeCreate"
	BeforeUpdate = "beforeUpdate"
	BeforeDelete = "beforeDelete"

	AfterRead   = "afterRead"
	AfterCreate = "afterCreate"
	AfterUpdate = "afterUpdate"
	AfterDelete = "afterDelete"

	Search = "search"
)

// adds event listener
func (R *Route) On(event string, listener Listener) *Route {
	if R.listeners == nil {
		R.listeners = map[string]Listener{}
	}
	R.listeners[event] = listener
	return R
}
func (R *Route) trigger(e string, ctx Context, h *kind.Holder) error {
	if R.listeners != nil {
		if ls, ok := R.listeners[e]; ok {
			return ls(ctx, h)
		}
	}
	return nil
}

// custom search
func (R *Route) Search(searchListener func(ctx Context, query string) ([]interface{}, error)) *Route {
	R.searchListener = searchListener
	return R
}

func (R *Route) Roles(rs ...Role) *Route {
	R.roles = map[Role]bool{}
	for _, r := range rs {
		R.roles[r] = true
	}
	return R
}

func (R *Route) Methods(ms ...string) *Route {
	R.methods = ms
	return R
}

func (R *Route) Get(x http.HandlerFunc) *Route {
	R.get = x
	return R
}
func (R *Route) Post(x http.HandlerFunc) *Route {
	R.post = x
	return R
}
func (R *Route) Put(x http.HandlerFunc) *Route {
	R.put = x
	return R
}
func (R *Route) Delete(x http.HandlerFunc) *Route {
	R.delete = x
	return R
}

func (R *Route) getHandler() http.HandlerFunc {
	if R.get != nil {
		return R.get
	}
	if R.kind == nil {
		return func(w http.ResponseWriter, r *http.Request) {}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		_, ctx := R.NewContext(r).Authenticate()

		if ok := ctx.HasPermission(R.kind, READ); !ok {
			ctx.PrintError(w, errors.ErrForbidden)
			return
		}

		q, name, id, sort, limit, offset, ancestor := r.FormValue("q"), r.FormValue("name"), r.FormValue("id"), r.FormValue("sort"), r.FormValue("limit"), r.FormValue("offset"), r.FormValue("ancestor")

		if err := R.trigger(BeforeRead, ctx, nil); err != nil {
			ctx.PrintError(w, err)
			return
		}

		if len(q) > 0 && R.kind.EnableSearch {
			if q == "*" {
				q = ""
			}
			index, err := OpenIndex(R.kind.Name)
			if err != nil {
				ctx.PrintError(w, err)
				return
			}

			// search refinements

			var itDiscovery = index.Search(ctx, q, &search.SearchOptions{
				Facets: []search.FacetSearchOption{
					search.AutoFacetDiscovery(0, 0),
				},
			})

			facetsResult, _ := itDiscovery.Facets()
			var facsOutput = map[string]interface{}{}
			for _, f := range facetsResult {
				for _, v := range f {
					facsOutput[v.Name] = f
				}
			}

			var facets []search.Facet
			for key, val := range r.URL.Query() {
				if key == "filter" {
					for _, v := range val {
						filter := strings.Split(v, ":")
						if len(filter) == 2 {
							// todo: currently only supports facet type search.Atom
							facets = append(facets, search.Facet{Name: filter[0], Value: search.Atom(filter[1])})
						}

					}
				}
			}

			var sortExpr []search.SortExpression
			if len(sort) > 0 {
				var desc bool
				if sort[:1] == "-" {
					sort = sort[1:]
					desc = true
				}
				sortExpr = append(sortExpr, search.SortExpression{Expr: sort, Reverse: !desc})
			}

			var results []interface{}
			var docKeys []*datastore.Key

			for t := index.Search(ctx, q, &search.SearchOptions{
				Refinements: facets,
				Sort: &search.SortOptions{
					Expressions: sortExpr,
				}}); ; {
				var doc = reflect.New(R.kind.SearchType).Interface()
				docKey, err := t.Next(doc)
				if err == search.Done {
					break
				}
				if err != nil {
					ctx.PrintError(w, err)
					return
				}

				if key, err := datastore.DecodeKey(docKey); err == nil {
					docKeys = append(docKeys, key)
				}

				results = append(results, doc)
			}

			if R.kind.RetrieveByIDOnSearch && len(docKeys) == len(results) {
				hs, err := kind.GetMulti(ctx, R.kind, docKeys...)
				if err != nil {
					ctx.PrintError(w, err)
					return
				}

				for k, h := range hs {
					results[k] = h.Value()
				}
			}

			ctx.PrintResult(w, map[string]interface{}{
				"count":   len(results),
				"results": results,
				"filters": facsOutput,
			})

			return
		} else if len(id) > 0 {
			// ordinary get
			key, err := datastore.DecodeKey(id)
			if err != nil {
				ctx.PrintError(w, err)
				return
			}
			h := R.kind.NewHolder(ctx.UserKey)
			err = h.Get(ctx, key)
			if err != nil {
				ctx.PrintError(w, err)
				return
			}
			ctx.Print(w, h.Value())
			return
		} else if len(name) > 0 {
			// ordinary get
			var parent *datastore.Key
			if ancestor != "false" {
				parent = ctx.UserKey
			}

			key := R.kind.NewKey(ctx, name, parent)
			h := R.kind.NewHolder(ctx.UserKey)
			err := h.Get(ctx, key)
			if err != nil {
				ctx.PrintError(w, err)
				return
			}
			ctx.Print(w, h.Value())
			return
		} else {
			// query
			limitInt, _ := strconv.Atoi(limit)
			offsetInt, _ := strconv.Atoi(offset)

			var hs []*kind.Holder
			var err error
			if ancestor == "false" && ctx.Role == AdminRole {
				hs, err = R.kind.Query(ctx, sort, limitInt, offsetInt, nil, nil)
				if err != nil {
					ctx.PrintError(w, err)
					return
				}
			} else {
				hs, err = R.kind.Query(ctx, sort, limitInt, offsetInt, nil, ctx.UserKey)
				if err != nil {
					ctx.PrintError(w, err)
					return
				}
			}

			var out []interface{}
			for _, h := range hs {
				if err := R.trigger(AfterRead, ctx, h); err != nil {
					ctx.PrintError(w, err)
					return
				}
				out = append(out, h.Value())
			}
			ctx.PrintResult(w, map[string]interface{}{
				"count":   len(out),
				"results": out,
			})
			return
		}
	}
}

func (R *Route) postHandler() http.HandlerFunc {
	if R.post != nil {
		return R.post
	}
	if R.kind == nil {
		return func(w http.ResponseWriter, r *http.Request) {}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		_, ctx := R.NewContext(r).Authenticate()

		if ok := ctx.HasPermission(R.kind, CREATE); !ok {
			ctx.PrintError(w, errors.ErrForbidden)
			return
		}

		h := R.kind.NewHolder(ctx.UserKey)
		err := h.Parse(ctx.Body())
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		if err := R.trigger(BeforeCreate, ctx, h); err != nil {
			ctx.PrintError(w, err)
			return
		}

		err = h.Add(ctx)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		if err := R.trigger(AfterCreate, ctx, h); err != nil {
			ctx.PrintError(w, err)
			return
		}

		if R.kind.EnableSearch {
			// put to search
			index, err := OpenIndex(R.kind.Name)
			if err != nil {
				ctx.PrintError(w, err)
				return
			}

			v := reflect.ValueOf(h.Value()).Elem()

			doc := reflect.New(R.kind.SearchType)

			for i := 0; i < R.kind.SearchType.NumField(); i++ {
				docFieldName := R.kind.SearchType.Field(i).Name

				valField := v.FieldByName(docFieldName)
				if !valField.IsValid() {
					continue
				}

				docField := doc.Elem().FieldByName(docFieldName)
				if docField.CanSet() {
					if docField.Kind() == reflect.Slice {

						// make slice to get value type
						sliceValTyp := reflect.MakeSlice(docField.Type(), 1, 1).Index(0).Type()

						if valField.Kind() == reflect.Slice {
							for j := 0; j < valField.Len(); j++ {
								docField.Set(reflect.Append(docField, valField.Index(j).Convert(sliceValTyp)))
							}
						}
					} else {
						docField.Set(valField.Convert(docField.Type()))
					}

				}
			}

			if _, err := index.Put(ctx, h.Id(), doc.Interface()); err != nil {
				ctx.PrintError(w, err)
				return
			}
		}

		ctx.Print(w, h.Value())
	}
}

func (R *Route) putHandler() http.HandlerFunc {
	if R.put != nil {
		return R.put
	}
	if R.kind == nil {
		return func(w http.ResponseWriter, r *http.Request) {}
	}
	type UpdateVal struct {
		Id    string      `json:"id"`
		Name  string      `json:"name"`
		Value interface{} `json:"value"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		_, ctx := R.NewContext(r).Authenticate()

		if ok := ctx.HasPermission(R.kind, UPDATE); !ok {
			ctx.PrintError(w, errors.ErrForbidden)
			return
		}

		var data = UpdateVal{}
		json.Unmarshal(ctx.Body(), &data)

		h := R.kind.NewHolder(ctx.UserKey)
		err := h.Parse(ctx.Body())
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		if len(data.Id) == 0 && len(data.Name) == 0 {
			ctx.PrintError(w, errors.New("must provide id or name"))
			return
		}

		var buf bytes.Buffer
		err = json.NewEncoder(&buf).Encode(data.Value)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		if data.Value == nil {
			ctx.PrintError(w, errors.New("must provide value"))
			return
		}

		var key *datastore.Key
		if len(data.Id) > 0 {
			key, err = datastore.DecodeKey(data.Id)
			if err != nil {
				ctx.PrintError(w, err)
				return
			}
		} else {
			key = R.kind.NewKey(ctx, data.Name, ctx.UserKey)
		}

		if err := R.trigger(BeforeUpdate, ctx, h); err != nil {
			ctx.PrintError(w, err)
			return
		}

		h.Parse(buf.Bytes())

		/*ctx.Print(w, h.Output())
		return*/

		err = h.Update(ctx, key)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		if err := R.trigger(AfterUpdate, ctx, h); err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.Print(w, h.Value())
	}
}

// todo
func (R *Route) deleteHandler() http.HandlerFunc {
	if R.delete != nil {
		return R.delete
	}
	return func(w http.ResponseWriter, r *http.Request) {}
}

var (
	ErrEmailUndefined    = errors.New("email undefined")
	ErrPasswordUndefined = errors.New("password undefined")
	ErrInvalidCallback   = errors.New("callback is not a valid URL")
	ErrInvalidEmail      = errors.New("email is not valid")
	ErrPasswordTooLong   = errors.New("password must be exactly or less than 128 characters long")
	ErrPasswordTooShort  = errors.New("password must be at least 6 characters long")
)

func checkEmail(v string) error {
	if len(v) == 0 {
		return ErrEmailUndefined
	}
	if !govalidator.IsEmail(v) {
		return ErrInvalidEmail
	}
	if len(v) > 128 || len(v) < 5 {
		return ErrInvalidEmail
	}

	return nil
}

func checkPassword(v string) error {
	if len(v) == 0 {
		return ErrPasswordUndefined
	}
	if len(v) > 128 {
		return ErrPasswordTooLong
	}
	if len(v) < 6 {
		return ErrPasswordTooShort
	}

	return nil
}

// USER HANDLERS
func (R *Route) getUserHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ok, ctx := R.NewContext(r).Authenticate()
		if !ok {
			ctx.PrintError(w, errors.ErrUnathorized)
			return
		}
		if ctx.Role != AdminRole {
			ctx.PrintError(w, errors.ErrUnathorized)
			return
		}

		id := r.FormValue("id")
		keyId, err := datastore.DecodeKey(id)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		// get user
		user := new(User)
		err = datastore.Get(ctx, keyId, user)
		if err != nil {
			if err == datastore.ErrNoSuchEntity {
				ctx.PrintError(w, errors.ErrUserDoesNotExist)
				return
			}
			ctx.PrintError(w, err)
			return
		}

		ctx.Print(w, user)
	}
}

func (R *Route) getUsersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ok, ctx := R.NewContext(r).Authenticate()
		if !ok {
			ctx.PrintError(w, errors.ErrUnathorized)
			return
		}
		if ctx.Role != AdminRole {
			ctx.PrintError(w, errors.ErrUnathorized)
			return
		}

		var hs []*User
		var err error

		q := datastore.NewQuery("_user")

		t := q.Run(ctx)
		for {
			var h = new(User)
			h.Id, err = t.Next(h)
			if err == datastore.Done {
				break
			}
			if err != nil {
				ctx.PrintError(w, err)
				return
			}
			hs = append(hs, h)
		}

		ctx.Print(w, hs)
	}
}

func (R *Route) loginHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := R.NewContext(r)

		email, password := r.FormValue("email"), r.FormValue("password")

		err := checkEmail(email)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}
		err = checkPassword(password)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		email = strings.ToLower(email)

		// get user
		userKey := datastore.NewKey(ctx, "_user", email, 0, nil)
		user := new(User)
		err = datastore.Get(ctx, userKey, user)
		if err != nil {
			if err == datastore.ErrNoSuchEntity {
				ctx.PrintError(w, errors.ErrUserDoesNotExist)
				return
			}
			ctx.PrintError(w, err)
			return
		}

		// decrypt hash
		err = decrypt(user.hash, []byte(password))
		if err != nil {
			ctx.PrintError(w, errors.ErrUserPasswordIncorrect)
			// todo: log and report
			return
		}

		// get profile
		//user.LoadProfile(ctx, a.options.UserProfileKind)

		// create a token
		token := NewToken(user)

		// sign the new token
		signedToken, err := R.a.SignToken(token)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.PrintAuth(w, signedToken, user)
	}
}

func (R *Route) registrationHandler(role Role) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := R.NewContext(r)

		email, password, meta := r.FormValue("email"), r.FormValue("password"), r.FormValue("meta")

		err := checkEmail(email)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}
		err = checkPassword(password)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		email = strings.ToLower(email)

		// create password hash
		hash, err := crypt([]byte(password))
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		// create User
		user := &User{
			Email: email,
			hash:  hash,
			Role:  string(role),
		}

		if len(meta) > 0 {
			json.Unmarshal([]byte(meta), &user.Meta)
		}

		if user.Meta == nil {
			user.Meta = map[string]interface{}{}
		}

		if _, ok := user.Meta["lang"]; !ok {
			user.Meta["lang"] = ctx.Language()
		}

		err = datastore.RunInTransaction(ctx, func(tc context.Context) error {
			userKey := datastore.NewKey(tc, "_user", user.Email, 0, nil)
			err := datastore.Get(tc, userKey, &datastore.PropertyList{})
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					// register
					_, err := datastore.Put(tc, userKey, user)
					return err
				}
				return err
			}
			return errors.ErrUserAlreadyExists
		}, nil)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		// create a token
		token := NewToken(user)

		// sign the new token
		signedToken, err := R.a.SignToken(token)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		if R.a.options.RequireEmailConfirmation && user.HasConfirmedEmail {
			ctx.PrintAuth(w, signedToken, user)
		} else {
			ctx.Print(w, "success")
		}

		if R.a.OnUserSignUp != nil {
			R.a.OnUserSignUp(ctx, *user, *signedToken)
		}
	}
}

func (R *Route) confirmEmailHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ok, ctx := R.NewContext(r).Authenticate()

		if !ok {
			ctx.PrintError(w, errors.ErrUnathorized)
			return
		}

		callback := r.FormValue("callback")
		if !govalidator.IsURL(callback) {
			ctx.PrintError(w, ErrInvalidCallback)
			return
		}

		user := new(User)
		// update User
		err := datastore.RunInTransaction(ctx, func(tc context.Context) error {

			err := datastore.Get(tc, ctx.UserKey, user)
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					return errors.ErrUserDoesNotExist
				}
				return err
			}

			user.HasConfirmedEmail = true

			_, err = datastore.Put(tc, ctx.UserKey, user)
			return err
		}, nil)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		// create a token
		token := NewToken(user)

		// sign the new token
		signedToken, err := R.a.SignToken(token)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		if R.a.OnUserVerified != nil {
			R.a.OnUserVerified(ctx, *user, *signedToken)
		}

		http.Redirect(w, r, callback, http.StatusTemporaryRedirect)
	}
}

func (R *Route) changePasswordHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ok, ctx := R.NewContext(r).Authenticate()

		if !ok {
			ctx.PrintError(w, errors.ErrUnathorized)
			return
		}

		password, newPassword := r.FormValue("password"), r.FormValue("newPassword")

		err := checkPassword(newPassword)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		user := new(User)
		// update User
		err = datastore.RunInTransaction(ctx, func(tc context.Context) error {

			err = datastore.Get(tc, ctx.UserKey, user)
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					return errors.ErrUserDoesNotExist
				}
				return err
			}

			// decrypt hash
			err = decrypt(user.hash, []byte(password))
			if err != nil {
				return errors.ErrUserPasswordIncorrect
			}

			// create new password hash
			user.hash, err = crypt([]byte(newPassword))
			if err != nil {
				return err
			}

			_, err := datastore.Put(tc, ctx.UserKey, user)
			return err
		}, nil)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.Print(w, "success")

	}
}

func (R *Route) updateMeta() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ok, ctx := R.NewContext(r).Authenticate()
		if !ok {
			ctx.PrintError(w, errors.ErrUnathorized)
			return
		}

		meta := ctx.Body()

		var m map[string]interface{}
		if len(meta) > 0 {
			json.Unmarshal(meta, &m)
		} else {
			ctx.PrintError(w, errors.New("body is empty"))
			return
		}

		// do everything in a transaction
		user := new(User)

		err := datastore.RunInTransaction(ctx, func(tc context.Context) error {
			// get user
			err := datastore.Get(ctx, ctx.UserKey, user)
			if err != nil {
				return err
			}

			for k, v := range m {
				user.SetMeta(k, v)
			}

			_, err = datastore.Put(ctx, ctx.UserKey, user)
			return err
		}, &datastore.TransactionOptions{XG: true})
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.Print(w, user)
	}
}
