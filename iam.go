package apis

// parent is entry key, id is user key
type IAM struct {
	Scopes []string
}

var IAMKind = NewKind(&KindOptions{
	Path: "iam",
	Type: IAM{},
})

type Role string

