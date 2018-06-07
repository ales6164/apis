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
	"google.golang.org/appengine/search"
	"io/ioutil"
	"cloud.google.com/go/storage"
	"github.com/ales6164/apis/kind"
	"reflect"
)

type StoredFile struct {
	CreatedBy   *datastore.Key `json:"createdBy,omitempty"`
	CreatedAt   time.Time      `json:"createdAt,omitempty"`
	BlobKey     string         `datastore:",noindex" json:"blobKey,omitempty"`
	URL         string         `datastore:",noindex" json:"url,omitempty"`
	Image       Image          `datastore:",noindex" json:"image,omitempty"`
	Filename    string         `datastore:",noindex" json:"filename,omitempty"`
	Title       string         `datastore:",noindex" json:"title,omitempty"`
	Description string         `datastore:",noindex" json:"description"`
	ContentType string         `json:"contentType,omitempty"`
	Serving     bool           `json:"serving,omitempty"`
	Size        int64          `json:"size,omitempty"`
}

type StoredFileDoc struct {
	CreatedBy   search.Atom
	CreatedAt   time.Time
	Filename    search.Atom
	ContentType search.Atom
	Title       string
	Description string
	Size        float64
}

var MediaKind = kind.New(reflect.TypeOf(StoredFile{}), &kind.Options{
	SearchType:           reflect.TypeOf(StoredFileDoc{}),
	EnableSearch:         true,
	Name:                 "_file",
	RetrieveByIDOnSearch: true,
})

//
type Image struct {
	Orig string `json:"orig,omitempty"`
}

func initMedia(a *Apis, r *mux.Router) {
	// MEDIA
	if len(a.options.StorageBucket) > 0 {
		mediaRoute := &Route{
			kind:    MediaKind,
			a:       a,
			path:    "/apis/media",
			methods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		}
		// GET MEDIA
		r.Handle(mediaRoute.path, a.middleware.Handler(mediaRoute.getHandler())).Methods(http.MethodGet)
		// UPLOAD
		r.Handle(mediaRoute.path, a.middleware.Handler(uploadHandler(mediaRoute))).Methods(http.MethodPost)
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

		// read file
		fileMultipart, fileHeader, err := ctx.r.FormFile("file")
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
		fileName := RandStringBytesMaskImprSrc(32)

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
		var storedFile = StoredFile{
			CreatedBy:   ctx.UserKey(),
			CreatedAt:   time.Now(),
			Filename:    fileHeader.Filename,
			ContentType: http.DetectContentType(bytes),
			Size:        int64(len(bytes)),
			URL:         "https://storage.googleapis.com/" + path.Join(bucketName, storageFilePath),
			BlobKey:     string(blobKey),
		}

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

		key := datastore.NewIncompleteKey(ctx, "_file", nil)
		if key, err = datastore.Put(ctx, key, &storedFile); err != nil {
			ctx.PrintError(w, err)
			return
		}

		index, err := search.Open("_file")
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		doc := &StoredFileDoc{
			CreatedAt:   storedFile.CreatedAt,
			ContentType: search.Atom(storedFile.ContentType),
			Filename:    search.Atom(storedFile.Filename),
			Title:       storedFile.Title,
			Description: storedFile.Description,
			Size:        float64(storedFile.Size),
		}

		if storedFile.CreatedBy != nil {
			doc.CreatedBy = search.Atom(storedFile.CreatedBy.Encode())
		}

		_, err = index.Put(ctx, key.Encode(), doc)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		ctx.Print(w, storedFile)
	}
}
