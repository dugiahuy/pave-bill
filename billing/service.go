package billing

import (
	"encore.app/billing/service"
	"encore.app/billing/store"
	"encore.dev/storage/sqldb"
)

var paveBillDB = sqldb.NewDatabase("pave_bill", sqldb.DatabaseConfig{
	Migrations: "./db/migrations",
})

//encore:service
type Service struct {
	services service.Services
}

func initService() (*Service, error) {
	pgxdb := sqldb.Driver(paveBillDB)
	repo := store.NewStore(pgxdb)
	services := service.NewServices(repo)

	return &Service{
		services: services,
	}, nil
}
