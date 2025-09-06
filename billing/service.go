package billing

import (
	"encore.app/billing/service"
	"encore.app/billing/store"
	"encore.dev/rlog"
	"encore.dev/storage/sqldb"
)

var paveBillDB = sqldb.NewDatabase("pave_bill", sqldb.DatabaseConfig{
	Migrations: "./db/migrations",
})

//encore:service
type Billing struct {
	services service.Services
}

func initService() (*Billing, error) {
	pgxdb := sqldb.Driver(paveBillDB)

	rlog.Info("Initializing Store", "pgxdb", pgxdb)
	repo := store.NewStore(pgxdb)

	rlog.Info("Initializing Store", "repo", repo)
	services := service.NewServices(repo)

	return &Billing{
		services: services,
	}, nil
}
