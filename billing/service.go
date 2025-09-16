package billing

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"encore.dev/storage/sqldb"
	"github.com/go-playground/validator/v10"

	"encore.app/billing/business/bill"
	"encore.app/billing/business/currency"
	domain "encore.app/billing/domain/bill_state_machine"
	"encore.app/billing/repository"
	"encore.app/billing/workflow"
)

var (
	paveBillDB = sqldb.NewDatabase("pave_bill", sqldb.DatabaseConfig{
		Migrations: "./db/migrations",
	})
	validate = validator.New()

	taskQueue = "billing-queue"
)

//encore:service
type Service struct {
	business bill.Business
	temporal client.Client
	worker   worker.Worker
}

func initService() (*Service, error) {
	pgxdb := sqldb.Driver[*pgxpool.Pool](paveBillDB)
	repo := repository.NewRepository(pgxdb)

	temporal, worker, err := initTemporal()
	if err != nil {
		return nil, err
	}

	currencyBusiness := currency.NewCurrencyBusiness(repo.Currencies)
	billStateMachine := domain.NewBillStateMachine(pgxdb, repo.Bills, repo.LineItems)
	billService := bill.NewBillBusiness(repo.Bills, repo.LineItems, billStateMachine, currencyBusiness)

	// Set activity dependencies for Temporal workflows
	workflow.SetActivityDependencies(billService)

	return &Service{
		business: billService,
		temporal: temporal,
		worker:   worker,
	}, nil
}

func initTemporal() (client.Client, worker.Worker, error) {
	c, err := client.Dial(client.Options{
		HostPort:  "localhost:7233",
		Namespace: "default",
	})
	if err != nil {
		return nil, nil, err
	}

	w := worker.New(c, taskQueue, worker.Options{})

	w.RegisterWorkflow(workflow.BillingPeriod)

	w.RegisterActivity(workflow.CloseBillActivity)
	w.RegisterActivity(workflow.ActivateBillActivity)
	w.RegisterActivity(workflow.UpdateBillTotalActivity)

	if err = w.Start(); err != nil {
		c.Close()
		return nil, nil, fmt.Errorf("start temporal worker: %v", err)
	}

	return c, w, nil
}

func (s *Service) Shutdown(force context.Context) {
	s.temporal.Close()
	s.worker.Stop()
}
