package admin

import (
	"fmt"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/user"
	"html/template"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

const (
	ClientKind = "_client"
)

type Client struct {
	AppID        string `json:"app_id"`
	ClientID     int64  `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

const templ = `
{{define "index"}}
    <!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="UTF-8">
        <title>Admin</title>
        <link rel="stylesheet" href="https://stackpath.bootstrapcdn.com/bootstrap/4.2.1/css/bootstrap.min.css"
              integrity="sha384-GJzZqFGwb1QTTN6wy59ffF1BuGJpLSa9DkKMp0DgiMDm4iYMj70gZWKYbI706tWS"
              crossorigin="anonymous">
    </head>
    <body>
    <div class="container">
        <div class="row justify-content-left mt-5">

            <h2>Admin</h2>

        </div>
        <div class="row justify-content-center mt-5">

            <div class="table-responsive">
                <table class="table table-bordered">
                    <thead class="thead-light">
                    <tr>
                        <th scope="col">App ID</th>
                        <th scope="col">Client ID</th>
                        <th scope="col">Client Secret</th>
                        <th scope="col"></th>
                    </tr>
                    </thead>
                    <tbody>
                    {{range .Items}}
                        <tr>
                            <td>{{.AppID}}</td>
                            <td>{{.ClientID}}</td>
                            <td>{{.ClientSecret}}</td>
                            <td>
                                <button type="button" class="btn btn-danger js-delete" aria-label="Close" data-client="{{.ClientID}}"><span
                                            aria-hidden="true">&times;</span> Delete
                                </button>
                            </td>
                        </tr>
                    {{end}}
                    </tbody>
                </table>
            </div>

        </div>
        <div class="row justify-content-left mt-2">
            <button type="button" class="btn btn-primary" data-toggle="modal" data-target="#addEntryModal">Add Client Access</button>
        </div>
    </div>

    <!-- Modal -->
    <div class="modal fade" id="addEntryModal" tabindex="-1" role="dialog" aria-labelledby="addEntryModal"
         aria-hidden="true">
        <div class="modal-dialog modal-dialog-centered" role="document">
            <form class="modal-content" method="post" action="/_service/add">
                <div class="modal-header">
                    <h5 class="modal-title" id="exampleModalCenterTitle">Add Client Access</h5>
                    <button type="button" class="close" data-dismiss="modal" aria-label="Close">
                        <span aria-hidden="true">&times;</span>
                    </button>
                </div>
                <div class="modal-body">
                    <div class="form-group">
                        <label for="app_id">App ID</label>
<div class="input-group mb-3">
                        <input type="text" class="form-control" name="app_id" id="app_id">
<div class="input-group-append">
    <span class="input-group-text" id="basic-addon2">.appspot.com</span>
  </div>
</div>
                    </div>
                </div>
                <div class="modal-footer">
                    <button type="button" class="btn btn-secondary" data-dismiss="modal">Close</button>
                    <button type="submit" class="btn btn-primary">Add</button>
                </div>
            </form>
        </div>
    </div>

    </body>
    </html>
    <script src="https://ajax.googleapis.com/ajax/libs/jquery/3.3.1/jquery.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/popper.js/1.14.6/umd/popper.min.js"
            integrity="sha384-wHAiFfRlMFy6i5SRaxvfOCifBUQy1xHdJ/yoi7FRNXMRBu5WHdZYu1hA6ZOblgut"
            crossorigin="anonymous"></script>
    <script src="https://stackpath.bootstrapcdn.com/bootstrap/4.2.1/js/bootstrap.min.js"
            integrity="sha384-B0UglyR+jN6CkvvICOB2joaf5I4l3gm9GU6Hc1og6Ls7i6U/mkkaduKaBhlAXv9k"
            crossorigin="anonymous"></script>
    <script>
        $('.js-delete').on('click', function () {
             $.ajax({
                 url: '/_service/' + $(this).data('client'),
                 type: 'DELETE',
                 success: function(result) {
                     location.reload();
                 }
             });
         });
    </script>
{{end}}`

func Serve(r *mux.Router) {
	t, _ := template.New("").Parse(templ)

	var renderAdmin = func(w http.ResponseWriter, r *http.Request) {
		ctx := appengine.NewContext(r)
		var sas []Client
		_, _ = datastore.NewQuery(ClientKind).GetAll(ctx, &sas)

		_ = t.ExecuteTemplate(w, "index", map[string]interface{}{
			"Items": sas,
		})
	}

	r.HandleFunc("/_service", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-type", "text/html; charset=utf-8")
		ctx := appengine.NewContext(r)
		u := user.Current(ctx)
		if u == nil || !u.Admin {
			url, _ := user.LoginURL(ctx, "/")
			_, _ = fmt.Fprintf(w, `<a href="%s">Sign in as admin</a>`, url)
			return
		}
		/*url, _ := user.LogoutURL(ctx, "/_service")
		_, _ = fmt.Fprintf(w, `Welcome, %s! (<a href="%s">sign out</a>)`, u, url)
		*/

		time.Sleep(1 * time.Second)

		renderAdmin(w, r)
	}).Methods(http.MethodGet)

	// Add service account
	r.HandleFunc("/_service/add", func(w http.ResponseWriter, r *http.Request) {
		ctx := appengine.NewContext(r)
		u := user.Current(ctx)
		if u == nil || !u.Admin {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		appId := r.FormValue("app_id")
		if len(appId) == 0 {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		err := datastore.RunInTransaction(ctx, func(ctx context.Context) error {
			key := datastore.NewIncompleteKey(ctx, ClientKind, nil)
			sa := &Client{AppID: appId, ClientSecret: RandStringBytesMaskImprSrc(LetterNumberBytes, 32)}
			key, err := datastore.Put(ctx, key, sa)
			if err != nil {
				return err
			}
			sa.ClientID = key.IntID()
			key, err = datastore.Put(ctx, key, sa)
			return err
		}, nil)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		time.Sleep(1 * time.Second)

		r.Method = http.MethodGet
		http.Redirect(w, r, "/_service", http.StatusSeeOther)
	}).Methods(http.MethodPost)

	// Remove
	r.HandleFunc("/_service/{client}", func(w http.ResponseWriter, r *http.Request) {
		ctx := appengine.NewContext(r)
		u := user.Current(ctx)
		if u == nil || !u.Admin {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		clientId := mux.Vars(r)["client"]
		if len(clientId) == 0 {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		clientIdNum, err := strconv.Atoi(clientId)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		key := datastore.NewKey(ctx, ClientKind, "", int64(clientIdNum), nil)
		err = datastore.Delete(ctx, key)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(http.StatusText(http.StatusOK)))
	}).Methods(http.MethodDelete)
}

const LetterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const LetterNumberBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const NumberBytes = "0123456789"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

func RandStringBytesMaskImprSrc(letterBytes string, n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
