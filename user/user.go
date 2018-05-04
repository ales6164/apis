package user

import (
	"google.golang.org/appengine/datastore"
	"golang.org/x/net/context"
	"time"
)

type User struct {
	Id                *datastore.Key `datastore:"-"`
	Email             string         `json:"-"`
	Role              string         `json:"-"`
	Language          string         `json:"-"`
	IsAnonymous       bool           `json:"-"`
	HasConfirmedEmail bool           `json:"-"`
	Profile           Profile        `json:"-"`
	Hash              []byte         `json:"-"`
}

type SocialProfile struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Profile struct {
	Id             *datastore.Key  `datastore:"-" json:"id"`
	Photo          string          `json:"photo"`
	Email          string          `json:"email"`
	Title          string          `json:"title"`
	FirstName      string          `json:"firstName"`
	SecondName     string          `json:"secondName"`
	LastName       string          `json:"lastName"`
	Sex            string          `json:"sex"`
	DateOfBirth    time.Time       `json:"dateOfBirth"`
	CityOfBirth    Place           `json:"cityOfBirth"`
	Address        Place           `json:"address"`
	Address2       Place           `json:"address2"`
	Company        Company         `json:"company"`
	Contact        Contact         `json:"contact"`
	SocialProfiles []SocialProfile `json:"socialProfiles"`
}

type Place struct {
	Address string  `json:"address"`
	City    string  `json:"city"`
	State   string  `json:"state"`
	Country string  `json:"country"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
}

type Company struct {
	Name           string          `json:"name"`
	VatNumber      string          `json:"vatNumber"`
	Address        Place           `json:"address"`
	Contact        Contact         `json:"contact"`
	SocialProfiles []SocialProfile `json:"socialProfiles"`
}

type Contact struct {
	Email        string `json:"email"`
	Email2       string `json:"email2"`
	PhoneNumber  string `json:"phoneNumber"`
	PhoneNumber2 string `json:"phoneNumber2"`
}

func GetProfile(ctx context.Context, userKey *datastore.Key) (Profile, error) {
	usr := new(User)
	err := datastore.Get(ctx, userKey, usr)
	return usr.Profile, err
}
