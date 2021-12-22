package enum

type TokenType int

const (
	RefreshToken TokenType = iota
	AccessToken
)

func (d TokenType) String() string {
	return [...]string{
		"refresh_token",
		"access_token",
	}[d]
}
