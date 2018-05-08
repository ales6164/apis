package file

import (
	"net/http"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"time"
	"github.com/ales6164/apis"
	"path"
	"errors"
	"cloud.google.com/go/storage"
	"io/ioutil"
)

type StoredFile struct {
	CreatedBy   *datastore.Key
	CreatedAt   time.Time
	URL         string
	RelativeURL string
	FileName    string
	Type        string
	BytesLength int `datastore:"-"`
}

func FileHandler(R *apis.Route, bucketName string, basePath string, fileTypes ...string) http.HandlerFunc {
	var supportedFileTypes = map[string]string{}
	for _, t := range fileTypes {
		supportedFileTypes[t] = t
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if appengine.IsDevAppServer() {
			http.Error(w, "This works only on production server", http.StatusNotImplemented)
		}

		ctx := R.NewContext(r)

		fileMultipart, fileHeader, err := r.FormFile("file")
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		fileBytes, err := ioutil.ReadAll(fileMultipart)
		fileMultipart.Close()
		if err != nil {
			ctx.PrintError(w, err)
			return
		}

		fileName := path.Base(fileHeader.Filename)
		fileType := path.Ext(fileName)

		if len(fileName) == 0 {
			ctx.PrintError(w, errors.New("file name must not be empty"))
			return
		}

		if _, ok := supportedFileTypes[fileType]; !ok {
			ctx.PrintError(w, errors.New("File type "+fileType+" not supported"))
			return
		}

		// pre-setup bucket
		client, err := storage.NewClient(ctx)
		if err != nil {
			ctx.PrintError(w, err)
			return
		}
		defer client.Close()

		bucket := client.Bucket(bucketName)

		uniqKey := apis.RandStringBytesMaskImprSrc(16)
		relativeURL := path.Join(basePath, uniqKey, fileName)
		var storedFile = StoredFile{
			CreatedAt:   time.Now(),
			FileName:    fileName,
			Type:        fileType,
			CreatedBy:   ctx.UserKey(),
			URL:         "https://storage.googleapis.com/" + bucketName + "/" + relativeURL,
			RelativeURL: relativeURL,
		}

		key := datastore.NewKey(ctx, "_file", uniqKey, 0, nil)
		if _, err := datastore.Put(ctx, key, &storedFile); err != nil {
			ctx.PrintError(w, err)
			return
		}

		// upload to bucket
		var numOfBytes int
		writer := bucket.Object(relativeURL).NewWriter(ctx)
		writer.Metadata = map[string]string{
			"x-goog-acl": "public-read",
		}
		if numOfBytes, err = writer.Write(fileBytes); err != nil {
			ctx.PrintError(w, err)
			return
		}
		if err := writer.Close(); err != nil {
			ctx.PrintError(w, err)
			return
		}

		storedFile.BytesLength = numOfBytes

		ctx.Print(w, storedFile)
	}
}
