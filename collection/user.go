package collection

type User struct {
	Name  string   `json:"name"`
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}

type PublicUser struct {
	Name  string   `json:"name"`
	Email string   `json:"-"`
	Roles []string `json:"-"`
}

var UserCollection = New("users", User{})
var PublicUserCollection = New("users", PublicUser{})
