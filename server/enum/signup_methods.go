package enum

type SignupMethod int

const (
	BasicAuth SignupMethod = iota
	MagicLinkLogin
	Google
	Github
	Facebook
)

func (d SignupMethod) String() string {
	return [...]string{
		"basic_auth",
		"magic_link_login",
		"google",
		"github",
		"facebook",
	}[d]
}
