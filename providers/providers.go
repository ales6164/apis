package providers

func WithEmailPasswordProvider(cost int, signingKey []byte) *EmailPasswordProvider {
	return &EmailPasswordProvider{
		Cost:       cost,
		SigningKey: signingKey,
	}
}
