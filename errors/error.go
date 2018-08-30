package errors

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

func New(msg string) *Error {
	return &Error{Message: msg}
}

var (
	ErrPasswordLength                = New("password must be between 6 and 128 characters long")
	ErrDecodingKey                   = New("error decoding key")
	ErrOrderUnavailableWithIdParam   = New("order unavailable with param id")
	ErrLimitUnavailableWithIdParam   = New("limit unavailable with param id")
	ErrOffsetUnavailableWithIdParam  = New("offset unavailable with param id")
	ErrFiltersUnavailableWithIdParam = New("filters unavailable with param id")
	ErrEntityExists                  = New("entity already exists")
	ErrEntityNameTooShort            = New("entity name must be at least 3 characters long")
	ErrEntrySlugDouble               = New("entry with the same slug already exists")
	ErrUserDoesNotExist              = New("user with that email does not exist")
	ErrUserPasswordIncorrect         = New("email or password is not correct")
	ErrPhotoInvalidFormat            = New("photo not a valid url")
	ErrUserAlreadyExists             = New("user with that email already exists")
	ErrInvalidFormInput              = New("invalid form input")
	ErrProjectAlreadyExists          = New("project already exists")
	ErrFieldRequired                 = New("field required")
	ErrFieldValueTypeNotValid        = New("field value type not valid")
	ErrUnathorized                   = New("unathorized")
	ErrCallbackUndefined             = New("callback undefined")
	ErrUserProfileDoesNotExist       = New("user profile does not exist")
	ErrForbidden                     = New("action forbidden")
	ErrPageNotFound                  = New("404 page not found")
)
