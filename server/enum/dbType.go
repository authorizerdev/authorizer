package enum

type DbType int

const (
	Postgres DbType = iota
	Sqlite
	Mysql
	SQLServer
	Arangodb
)

func (d DbType) String() string {
	return [...]string{
		"postgres",
		"sqlite",
		"mysql",
		"sqlserver",
		"arangodb",
	}[d]
}
