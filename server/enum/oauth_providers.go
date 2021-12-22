package enum

type OAuthProvider int

const (
	GoogleProvider OAuthProvider = iota
	GithubProvider
)

func (d OAuthProvider) String() string {
	return [...]string{
		"google_provider",
		"github_provider",
	}[d]
}
