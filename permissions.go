package apis

import (
	"github.com/ales6164/apis-v1/kind"
)

type Role string

const (
	PublicRole Role = "public"
	AdminRole  Role = "admin"
)

var (
	PublicRoles = []string{string(PublicRole)}
)

// userGroup: kind: scope
type Permissions map[Role]map[*kind.Kind][]Scope

func (p Permissions) parse() (permissions, error) {
	var perms = permissions{}
	for userRole, entityKindMap := range p {
		if _, ok := perms[string(userRole)]; !ok {
			perms[string(userRole)] = map[*kind.Kind]map[Scope]bool{}
		}
		for theKind, theScopes := range entityKindMap {
			if _, ok := perms[string(userRole)][theKind]; !ok {
				perms[string(userRole)][theKind] = map[Scope]bool{}
			}
			for _, theScope := range theScopes {
				var isPrivate bool
				switch theScope {
				case P_QUERY:
					theScope = QUERY
					isPrivate = true
				case P_GET:
					theScope = GET
					isPrivate = true
				case P_CREATE:
					theScope = CREATE
					isPrivate = true
				case P_UPDATE:
					theScope = UPDATE
					isPrivate = true
				case P_DELETE:
					theScope = DELETE
					isPrivate = true
				}
				perms[string(userRole)][theKind][theScope] = isPrivate
			}
		}
	}
	return perms, nil
}

type permissions map[string]map[*kind.Kind]map[Scope]bool // last bool indicates if permission is private or not

type Scope string

const (
	// work under default user group
	QUERY  Scope = "q"
	GET    Scope = "g"
	CREATE Scope = "c"
	UPDATE Scope = "u"
	DELETE Scope = "d"

	P_QUERY  Scope = "pq"
	P_GET    Scope = "pr"
	P_CREATE Scope = "pc"
	P_UPDATE Scope = "pu"
	P_DELETE Scope = "pd"
)
