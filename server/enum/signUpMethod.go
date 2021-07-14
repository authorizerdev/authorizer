package enum

type SignupMethod int

const (
	BasicAuth SignupMethod = iota
	MagicLink
	Google
	Github
	Facebook
)

func (d SignupMethod) String() string {
	return [...]string{
		"basic_auth",
		"magic_link",
		"google",
		"github",
		"facebook",
	}[d]
}
