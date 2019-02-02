package apis

import (
	"errors"
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/mail"
	"time"
)

const EmailConfirmationKind = "_emailConfirmation"

type EmailWaitingConfirmation struct {
	ValidUntil time.Time      `json:"validUntil"`
	Confirmed  bool           `json:"confirmed"`
	Provider   string         `json:"provider"`
	Identity   *datastore.Key `json:"identity"`
	Email      string         `json:"email"`
}

func sendEmailConfirmation(ctx context.Context, provider Provider, identityKey *datastore.Key, email string) error {
	if appengine.IsDevAppServer() {
		return errors.New("sending confirmation emails is prohibited from local servers")
	}

	// create email confirmation db entry
	var emailWaitingConfirmation = &EmailWaitingConfirmation{
		ValidUntil: time.Now().Add(time.Hour * 24),
		Provider:   provider.Name(),
		Identity:   identityKey,
		Email:      email,
	}
	var key = datastore.NewIncompleteKey(ctx, EmailConfirmationKind, identityKey)
	key, err := datastore.Put(ctx, key, emailWaitingConfirmation)
	if err != nil {
		return ErrSendingConfirmationEmail
	}

	url, err := createConfirmationURL(ctx, key)
	msg := &mail.Message{
		Sender:  "No-Reply <no-reply@" + appengine.ModuleName(ctx) + ".appspotmail.com>",
		To:      []string{email},
		Subject: "Confirm your email",
		Body:    fmt.Sprintf(confirmMessage, url),
	}
	err = mail.Send(ctx, msg)
	if err != nil {
		return ErrSendingConfirmationEmail
	}
	return nil
}

const confirmMessage = `
Thank you for creating an account!
Please confirm your email address by clicking on the link below:

%s
`