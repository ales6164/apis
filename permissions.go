package apis

type Role string

const (
	PublicRole Role = "public"
	AdminRole  Role = "admin"
)

// userGroup: kind: scope
type Permissions map[Role]map[Scope][]*Kind

func (p Permissions) parse() (permissions, error) {
	var perms = permissions{}
	for userGroupName, entityScopeMap := range p {
		if _, ok := perms[userGroupName]; !ok {
			perms[userGroupName] = map[Scope]map[*Kind]bool{}
		}

		for theScope, theKinds := range entityScopeMap {

			if _, ok := perms[userGroupName][theScope]; !ok {
				perms[userGroupName][theScope] = map[*Kind]bool{}
			}

			for _, theKind := range theKinds {
				perms[userGroupName][theScope][theKind] = true
			}
		}
	}

	return perms, nil
}

type permissions map[Role]map[Scope]map[*Kind]bool

type Scope string

const (
	// work under default user group
	READ   Scope = "r"
	CREATE Scope = "c"
	UPDATE Scope = "u"
	DELETE Scope = "d"
)
