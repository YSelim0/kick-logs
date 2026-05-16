package ports

type PasswordHasher interface {
	Hash(plainPassword string) (string, error)
	Verify(plainPassword string, passwordHash string) bool
}

type TokenService interface {
	CreateAccessToken(userID int64) (string, error)
	GetUserID(token string) (int64, bool)
}
