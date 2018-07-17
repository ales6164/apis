package apis

import (
	"google.golang.org/appengine/datastore"
	"time"
)

type File struct {
	CreatedBy   *datastore.Key
	CreatedAt   time.Time
	URL         string
	Filename    string
	ContentType string
	BytesLength int `datastore:"-"`
}
