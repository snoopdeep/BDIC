module bdic/backend

// 1.26 rather than 1.26.3: a patch version here makes any machine on an older
// 1.26.x try to download that exact toolchain, which fails without network.
go 1.26.0

// Dependencies are resolved by `go mod tidy`, which `make deps` runs. There are
// only two direct ones:
//
//   github.com/jackc/pgx/v5   PostgreSQL driver and connection pool
//   golang.org/x/crypto       bcrypt, for password hashing
//
// Everything else the school management system does is standard library.

require (
	github.com/jackc/pgx/v5 v5.11.0
	golang.org/x/crypto v0.57.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/text v0.42.0 // indirect
)
