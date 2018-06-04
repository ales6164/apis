package apis

import (
	"github.com/ales6164/apis/kind"
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
type Permissions map[Role]map[Scope][]*kind.Kind

func (p Permissions) parse() (permissions, error) {
	var perms = permissions{}
	for userGroupName, entityScopeMap := range p {
		if _, ok := perms[string(userGroupName)]; !ok {
			perms[string(userGroupName)] = map[Scope]map[*kind.Kind]bool{}
		}
		for theScope, theKinds := range entityScopeMap {
			var isPrivate bool
			switch theScope {
			case P_QUERY:
				theScope = QUERY
				isPrivate = true
			case P_READ:
				theScope = READ
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

			if _, ok := perms[string(userGroupName)][theScope]; !ok {
				perms[string(userGroupName)][theScope] = map[*kind.Kind]bool{}
			}
			for _, theKind := range theKinds {
				perms[string(userGroupName)][theScope][theKind] = isPrivate
			}
		}
	}
	return perms, nil
}

type permissions map[string]map[Scope]map[*kind.Kind]bool // last bool indicates if permission is private or not

type Scope string

const (
	// work under default user group
	QUERY  Scope = "q"
	READ   Scope = "r"
	CREATE Scope = "c"
	UPDATE Scope = "u"
	DELETE Scope = "d"

	P_QUERY  Scope = "pq"
	P_READ   Scope = "pr"
	P_CREATE Scope = "pc"
	P_UPDATE Scope = "pu"
	P_DELETE Scope = "pd"
)
