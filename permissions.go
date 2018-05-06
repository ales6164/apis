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
			if _, ok := perms[string(userGroupName)][theScope]; !ok {
				perms[string(userGroupName)][theScope] = map[*kind.Kind]bool{}
			}
			for _, theKind := range theKinds {
				perms[string(userGroupName)][theScope][theKind] = true
			}
		}
	}
	return perms, nil
}

type permissions map[string]map[Scope]map[*kind.Kind]bool

type Scope string

const (
	// work under default user group
	READ   Scope = "r"
	CREATE Scope = "c"
	UPDATE Scope = "u"
	DELETE Scope = "d"
)
