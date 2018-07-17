package apis

import "github.com/asaskevich/govalidator"

func IsSlug(i interface{}, context interface{}) bool {
	switch v := i.(type) { // type switch on the struct field being validated
	case string:
		if len(v) > 0 && govalidator.Matches(v, `^[\w-]+$`) && govalidator.IsAlphanumeric(v[:1]) && govalidator.IsAlphanumeric(v[len(v)-1:]) {
			return true
		}
	}
	return false
}
