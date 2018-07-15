package apis

import (
	"net/http"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"time"
	"google.golang.org/appengine/blobstore"
	"google.golang.org/appengine/image"
	"strings"
	"github.com/gorilla/mux"
	"path"
	"io/ioutil"
	"cloud.google.com/go/storage"
	"github.com/ales6164/apis/kind"
	"reflect"
	"github.com/ales6164/apis/errors"
)

type StoredFile struct {
	Id          *datastore.Key `search:"-" datastore:"-" apis:"id" json:"id"`
	CreatedBy   *datastore.Key `search:"-" apis:"createdBy" json:"createdBy,omitempty"`
	CreatedAt   time.Time      `apis:"createdAt" json:"createdAt,omitempty"`
	BlobKey     string         `search:"-" datastore:",noindex" json:"blobKey,omitempty"`
	URL         string         `search:"-" datastore:",noindex" json:"url,omitempty"`
	Image       Image          `search:"-" datastore:",noindex" json:"image,omitempty"`
	Filename    string         `search:",,search.Atom" datastore:",noindex" json:"filename,omitempty"`
	Title       string         `datastore:",noindex" json:"title,omitempty"`
	Description string         `datastore:",noindex" json:"description"`
	Dir         string         `json:"-"`
	ContentType string         `search:",,search.Atom" json:"contentType,omitempty"`
	Serving     bool           `json:"serving,omitempty"`
	Size        int64          `search:",,float64" json:"size,omitempty"`
}

var MediaKind = kind.New(reflect.TypeOf(StoredFile{}), &kind.Options{
	EnableSearch:         true,
	Name:                 "_file",
	RetrieveByIDOnSearch: true,
})

//
type Image struct {
	Orig string `json:"orig,omitempty"`
}

// todo: handle /media/some-dir/some-other-dir as a route to a media library folder
// todo: each route/file can have a different set of rules of who can access it
// this could also be true for any other db entries ???!!
func initMedia(a *Apis, r *mux.Router) {
	// MEDIA
	if len(a.options.StorageBucket) > 0 {
		mediaRoute := &Route{
			kind:    MediaKind,
			a:       a,
			path:    "/media",
			methods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		}
		// GET MEDIA
		r.Handle("/media", a.middleware.Handler(mediaRoute.queryHandler())).Methods(http.MethodGet)
		r.Handle("/media/{id}", a.middleware.Handler(mediaRoute.getHandler())).Methods(http.MethodGet)
		// UPLOAD
		r.Handle("/media/{dir}", a.middleware.Handler(uploadHandler(mediaRoute))).Methods(http.MethodPost)
		r.Handle("/media", a.middleware.Handler(uploadHandler(mediaRoute))).Methods(http.MethodPost)
		/*r.Handle("/media/{blobKey}", a.middleware.Handler(serveHandler(mediaRoute))).Methods(http.MethodGet)*/
	}
}

func serveHandler(R *Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		blobstore.Send(w, appengine.BlobKey(mux.Vars(r)["blobKey"]))
	}
}

func uploadHandler(R *Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if appengine.IsDevAppServer() {
			http.Error(w, "This works only on production server", http.StatusNotImplemented)
		}

		ctx := R.NewContext(r)
		if ok, _ := ctx.HasPermission(R.kind, CREATE); !ok {
			ctx.PrintError(w, errors.ErrForbidden)
			return
		}

		dir := mux.Vars(r)["dir"]

		// read file
		fileMultipart, fileHeader, err := r.FormFile("file")
		if err != nil {
			ctx.PrintError(w, err)
			return
		}
		defer fileMultipart.Close()

		// read file
		bytes, err := ioutil.ReadAll(fileMultipart)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		// generate file name
		fileName := RandStringBytesMaskImprSrc(LetterBytes, 32)

		// save file to storage bucket
		client, err := storage.NewClient(ctx)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}
		defer client.Close()

		pathSplit := strings.Split(R.a.options.StorageBucket, "/")

		bucketName := pathSplit[0]
		bucket := client.Bucket(bucketName)

		filePath := strings.Join(pathSplit[1:], "/")

		// storage path
		storageFilePath := path.Join(filePath, fileName)

		// storage object
		obj := bucket.Object(storageFilePath)

		// write
		wc := obj.NewWriter(ctx)
		_, err = wc.Write(bytes)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}
		err = wc.Close()
		if err != nil {
			ctx.PrintError(w, err)
			return
		}
		acl := obj.ACL()
		if err := acl.Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
			ctx.PrintError(w, err)
			return
		}

		// get blob key
		gsPath := path.Join("/gs/", bucketName, storageFilePath)
		blobKey, err := blobstore.BlobKeyForFile(ctx.Context, gsPath)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		// create stored file object
		var storedFile = &StoredFile{
			CreatedBy:   ctx.UserKey(),
			CreatedAt:   time.Now(),
			Dir:         dir,
			Filename:    fileHeader.Filename,
			ContentType: http.DetectContentType(bytes),
			Size:        int64(len(bytes)),
			URL:         "https://storage.googleapis.com/" + path.Join(bucketName, storageFilePath),
			BlobKey:     string(blobKey),
		}

		mediaHolder := MediaKind.NewHolder(ctx.UserKey())
		mediaHolder.SetValue(storedFile)

		// create fast delivery image url
		if strings.Split(storedFile.ContentType, "/")[0] == "image" {
			if imgUrl, err := image.ServingURL(ctx, blobKey, &image.ServingURLOptions{
				Secure: true,
			}); err == nil {
				storedFile.Image = Image{
					Orig: imgUrl.String(),
				}
			}
		}

		if err := mediaHolder.Add(ctx); err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.Print(w, storedFile)
	}
}
