package enum

type DbType int

const (
	Postgres DbType = iota
	Sqlite
	Mysql
)

func (d DbType) String() string {
	return [...]string{
		"postgres",
		"sqlit",
		"mysql",
	}[d]
}
