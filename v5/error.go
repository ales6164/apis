package apis

/*
Form errors
*/
type Error struct {
	Message string `json:"error"`
	error
}

func (e *Error) With(msg string) *Error {
	e.Message = msg
	return e
}

func (e Error) Error() string {
	return e.Message
}

func NewError(msg string) *Error {
	return &Error{Message: msg}
}

var (
	ErrPasswordLength          = NewError("password must be between 6 and 128 characters long")
	ErrEntityExists            = NewError("entity already exists")
	ErrEntityNameTooShort      = NewError("entity name must be at least 3 characters long")
	ErrEntrySlugDouble         = NewError("entry with the same slug already exists")
	ErrUserDoesNotExist        = NewError("user with that email does not exist")
	ErrUserPasswordIncorrect   = NewError("email or password is not correct")
	ErrPhotoInvalidFormat      = NewError("photo not a valid url")
	ErrUserAlreadyExists       = NewError("user with that email already exists")
	ErrInvalidFormInput        = NewError("invalid form input")
	ErrProjectAlreadyExists    = NewError("project already exists")
	ErrFieldRequired           = NewError("field required")
	ErrFieldValueTypeNotValid  = NewError("field value type not valid")
	ErrUnathorized             = NewError("unathorized")
	ErrCallbackUndefined       = NewError("callback undefined")
	ErrUserProfileDoesNotExist = NewError("user profile does not exist")
	ErrForbidden               = NewError("action forbidden")
	ErrPageNotFound            = NewError("404 page not found")
)
