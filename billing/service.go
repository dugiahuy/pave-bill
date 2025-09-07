package billing

import (
	"encore.dev/storage/sqldb"
	"github.com/go-playground/validator/v10"

	"encore.app/billing/repository"
	"encore.app/billing/service"
)

var (
	paveBillDB = sqldb.NewDatabase("pave_bill", sqldb.DatabaseConfig{
		Migrations: "./db/migrations",
	})

	validate = validator.New()
)

//encore:service
type Service struct {
	services service.Services
}

func initService() (*Service, error) {
	pgxdb := sqldb.Driver(paveBillDB)
	repo := repository.NewRepository(pgxdb)
	services := service.NewServices(repo)

	return &Service{
		services: services,
	}, nil
}
