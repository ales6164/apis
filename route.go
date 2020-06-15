package apis

import (
	"cloud.google.com/go/datastore"
	"encoding/json"
	"github.com/ales6164/apis/errors"
	"github.com/ales6164/apis/kind"
	"github.com/asaskevich/govalidator"
	"google.golang.org/api/iterator"
	"net/http"
	"strconv"
	"strings"
)

type Route struct {
	a    *Apis
	kind *kind.Kind
	path string
	ProjectID string

	listeners      map[string]Listener
	searchListener func(ctx Context, query string) ([]interface{}, error)
	roles          map[Role]bool

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

		if len(q) > 0 && R.searchListener != nil {
			results, err := R.searchListener(ctx, q)
			if err != nil {
				ctx.PrintError(w, err)
				return
			}
			ctx.PrintResult(w, map[string]interface{}{
				"count":   len(results),
				"results": results,
			})
			return
		} else if len(id) > 0 {
			// ordinary get
			key, err := datastore.DecodeKey(id)
			if err != nil {
				ctx.PrintError(w, err)
				return
			}
			h := R.kind.NewHolder(R.ProjectID, ctx.UserKey)
			err = h.Get(ctx, key)
			if err != nil {
				ctx.PrintError(w, err)
				return
			}
			output, _ := ExpandMeta(ctx, h.Output())
			ctx.PrintResult(w, output)
			return
		} else if len(name) > 0 {
			// ordinary get
			var parent *datastore.Key
			if ancestor != "false" {
				parent = ctx.UserKey
			}

			key := R.kind.NewKey(ctx, name, parent)
			h := R.kind.NewHolder(R.ProjectID, ctx.UserKey)
			err := h.Get(ctx, key)
			if err != nil {
				ctx.PrintError(w, err)
				return
			}
			output, _ := ExpandMeta(ctx, h.Output())
			ctx.PrintResult(w, output)
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

			var out []map[string]interface{}
			for _, h := range hs {
				if err := R.trigger(AfterRead, ctx, h); err != nil {
					ctx.PrintError(w, err)
					return
				}
				dt, _ := ExpandMeta(ctx, h.Output())
				out = append(out, dt)
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

		h := R.kind.NewHolder(R.ProjectID, ctx.UserKey)
		err := h.ParseInput(ctx.Body())
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		if err := R.trigger(BeforeCreate, ctx, h); err != nil {
			ctx.PrintError(w, err)
			return
		}

		if err := h.Prepare(); err != nil {
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

		output, _ := ExpandMeta(ctx, h.Output())
		ctx.PrintResult(w, output)
	}
}

func (R *Route) putHandler() http.HandlerFunc {
	if R.put != nil {
		return R.put
	}
	if R.kind == nil {
		return func(w http.ResponseWriter, r *http.Request) {}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		_, ctx := R.NewContext(r).Authenticate()

		if ok := ctx.HasPermission(R.kind, UPDATE); !ok {
			ctx.PrintError(w, errors.ErrForbidden)
			return
		}

		h := R.kind.NewHolder(R.ProjectID, ctx.UserKey)
		err := h.ParseInput(ctx.Body())
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		var id string
		inId := h.ParsedInput["id"]
		if inId == nil {
			ctx.PrintError(w, errors.New("id not defined"))
			return
		}
		var ok bool
		if id, ok = inId.(string); !ok {
			ctx.PrintError(w, errors.New("id must be of type string"))
			return
		}

		key, err := datastore.DecodeKey(id)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		if err := R.trigger(BeforeUpdate, ctx, h); err != nil {
			ctx.PrintError(w, err)
			return
		}

		if err := h.Prepare(); err != nil {
			ctx.PrintError(w, err)
			return
		}

		err = h.Update(ctx, key)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		if err := R.trigger(AfterUpdate, ctx, h); err != nil {
			ctx.PrintError(w, err)
			return
		}

		output, _ := ExpandMeta(ctx, h.Output())
		ctx.PrintResult(w, output)
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

		c, err := datastore.NewClient(ctx, R.ProjectID)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		// get user
		user := new(User)
		err = c.Get(ctx, keyId, user)
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

		c, err := datastore.NewClient(ctx, R.ProjectID)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		var hs []*User

		q := datastore.NewQuery("_user")

		t := c.Run(ctx, q)
		for {
			var h = new(User)
			h.Id, err = t.Next(h)
			if err == iterator.Done {
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

		c, err := datastore.NewClient(ctx, R.ProjectID)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		email = strings.ToLower(email)

		// get user
		userKey := datastore.NameKey("_user", email, nil)
		user := new(User)
		err = c.Get(ctx, userKey, user)
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

		c, err := datastore.NewClient(ctx, R.ProjectID)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		_, err = c.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
			userKey := datastore.NameKey("_user", user.Email, nil)
			err := tx.Get(userKey, &datastore.PropertyList{})
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					// register
					_, err := tx.Put(userKey, user)
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

		c, err := datastore.NewClient(ctx, R.ProjectID)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		user := new(User)
		// update User
		_, err = c.RunInTransaction(ctx, func(tx *datastore.Transaction) error {

			err := tx.Get(ctx.UserKey, user)
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					return errors.ErrUserDoesNotExist
				}
				return err
			}

			user.HasConfirmedEmail = true

			_, err = tx.Put(ctx.UserKey, user)
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

		c, err := datastore.NewClient(ctx, R.ProjectID)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		user := new(User)
		// update User
		_, err = c.RunInTransaction(ctx, func(tx *datastore.Transaction) error {

			err = tx.Get(ctx.UserKey, user)
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

			_, err := tx.Put(ctx.UserKey, user)
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

		c, err := datastore.NewClient(ctx, R.ProjectID)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		// do everything in a transaction
		user := new(User)

		_, err = c.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
			// get user
			err := tx.Get(ctx.UserKey, user)
			if err != nil {
				return err
			}

			for k, v := range m {
				user.SetMeta(k, v)
			}

			_, err = tx.Put(ctx.UserKey, user)
			return err
		})
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.Print(w, user)
	}
}
