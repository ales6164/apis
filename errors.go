package apis

import "errors"

/*
Form errors
 */
type Error struct {
	Message string `json:"error"`
	Code    int    `json:"code"`
}

func (e *Error) Error() string {
	return e.Message
}

func NewError(msg string, code int) *Error {
	return &Error{msg, code}
}

var (
	ErrPasswordLength          = NewError("password must be between 6 and 128 characters long", 101)
	ErrEntityNameTooShort      = NewError("entity name must be at least 3 characters long", 102)
	ErrEntrySlugDouble         = NewError("entry with the same slug already exists", 103)
	ErrUserDoesNotExist        = NewError("user with that email does not exist", 104)
	ErrUserPasswordIncorrect   = NewError("email or password is not correct", 105)
	ErrPhotoInvalidFormat      = NewError("photo not a valid url", 106)
	ErrUserAlreadyExists       = NewError("user with that email already exists", 107)
	ErrInvalidFormInput        = NewError("invalid form input", 108)
	ErrProjectAlreadyExists    = NewError("project already exists", 109)
	ErrUnathorized             = errors.New("unathorized")
	ErrCallbackUndefined       = errors.New("callback undefined")
	ErrUserProfileDoesNotExist = NewError("user profile does not exist", 110)
	ErrForbidden               = errors.New("action forbidden")
	ErrPageNotFound            = errors.New("404 page not found")
)
