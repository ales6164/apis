package apis

import (
	"github.com/ales6164/apis/kind"
)

type Role string

const (
	PublicRole Role = "public"
	AdminRole  Role = "admin"
)

// userGroup: kind: scope
type Permissions map[Role]map[*kind.Kind][]Scope

func (p Permissions) parse() (permissions, error) {
	var perms = permissions{}
	for userGroupName, entityScopeMap := range p {
		if _, ok := perms[userGroupName]; !ok {
			perms[userGroupName] = map[*kind.Kind]map[Scope]bool{}
		}

		for theKind, entityScope := range entityScopeMap {

			if _, ok := perms[userGroupName][theKind]; !ok {
				perms[userGroupName][theKind] = map[Scope]bool{}
			}

			for _, scope := range entityScope {
				perms[userGroupName][theKind][Scope(scope)] = true
			}
		}
	}

	return perms, nil
}

type permissions map[Role]map[*kind.Kind]map[Scope]bool

type Scope string

const (
	// work under default user group
	Read   Scope = "read"
	Create Scope = "create"
	Update Scope = "update"
	Delete Scope = "delete"
)
