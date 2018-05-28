package apis

import (
	"net/http"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"time"
	"errors"
	"google.golang.org/appengine/blobstore"
	"google.golang.org/appengine/image"
	"strings"
	"github.com/gorilla/mux"
	"path"
	"google.golang.org/appengine/search"
)

type StoredFile struct {
	CreatedBy   *datastore.Key `json:"createdBy,omitempty"`
	CreatedAt   time.Time      `json:"createdAt,omitempty"`
	BlobKey     string         `datastore:",noindex" json:"blobKey,omitempty"`
	URL         string         `datastore:",noindex" json:"url,omitempty"`
	Image       *Image         `datastore:",noindex" json:"image,omitempty"`
	Filename    string         `datastore:",noindex" json:"filename,omitempty"`
	Title       string         `datastore:",noindex" json:"title,omitempty"`
	Description string         `datastore:",noindex" json:"description"`
	ContentType string         `json:"contentType,omitempty"`
	Size        int64          `json:"size,omitempty"`
}

type StoredFileDoc struct {
	CreatedBy   search.Atom
	CreatedAt   time.Time
	Filename    search.Atom
	ContentType search.Atom
	Title       string
	Description string
	Size        int64
}

//
type Image struct {
	Orig string `json:"orig,omitempty"`
}

func getMediaHandler(R *Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func serveHandler(R *Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		blobstore.Send(w, appengine.BlobKey(mux.Vars(r)["blobKey"]))
	}
}

func uploadHandler(R *Route, pathPrefix string) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		/*if appengine.IsDevAppServer() {
			http.Error(w, "This works only on production server", http.StatusNotImplemented)
		}*/

		ctx := R.NewContext(r)

		blobs, _, err := blobstore.ParseUpload(r)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		file := blobs["file"]
		if len(file) == 0 {
			ctx.PrintError(w, errors.New("no file uploaded"))
			return
		}

		endURL := path.Join(r.URL.Host, pathPrefix, "file", string(file[0].BlobKey))

		var storedFile = StoredFile{
			CreatedBy:   ctx.UserKey(),
			CreatedAt:   file[0].CreationTime,
			Filename:    file[0].Filename,
			ContentType: file[0].ContentType,
			Size:        file[0].Size,
			URL:         r.URL.Scheme + "://" + endURL,
			BlobKey:     string(file[0].BlobKey),
		}

		if strings.Split(file[0].ContentType, "/")[0] == "image" {
			if imgUrl, err := image.ServingURL(ctx, file[0].BlobKey, nil); err == nil {
				storedFile.Image = &Image{
					Orig: imgUrl.String(),
				}
			}
		}

		key := datastore.NewKey(ctx, "_file", storedFile.BlobKey, 0, nil)
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
			Size:        storedFile.Size,
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
