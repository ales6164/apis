package module

import (
	"github.com/gorilla/mux"
)

type Module interface {
	Init() error
	Name() string
	Router(p string) *mux.Router
}
