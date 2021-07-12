package enum

type SignupMethod int

const (
	Basic SignupMethod = iota
	MagicLink
	Google
	Github
	Facebook
)

func (d SignupMethod) String() string {
	return [...]string{
		"basic",
		"magiclink",
		"google",
		"github",
		"facebook",
	}[d]
}
