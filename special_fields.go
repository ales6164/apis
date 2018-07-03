package apis

import (
	"time"
	"google.golang.org/appengine/datastore"
)

type File struct {
	CreatedBy   *datastore.Key
	CreatedAt   time.Time
	URL         string
	Filename    string
	ContentType string
	BytesLength int `datastore:"-"`
}