package collection

// parent is entry key, id is user key
type GroupRelationship struct {
	Roles []string // fullControl, ...
}

type Role string

const (
	AllUsers              = "allUsers"              // given to all requests
	AllAuthenticatedUsers = "allAuthenticatedUsers" // giver to all authenticated requests

	FullControl = "fullcontrol"
	ReadOnly    = "readonly"
	ReadWrite   = "readwrite"
	Delete      = "delete"
)


func ContainsScope(arr []string, scopes ...string) bool {
	for _, scp := range scopes {
		for _, r := range arr {
			if r == scp {
				return true
			}
		}
	}
	return false
}