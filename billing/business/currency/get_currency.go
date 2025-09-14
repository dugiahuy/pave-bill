package currency

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"encore.dev/beta/errs"

	"encore.app/billing/model"
)

func (b *business) GetCurrency(ctx context.Context, code string) (*model.CurrencyInfo, error) {
	dbCurrency, err := b.currencyRepo.GetCurrency(ctx, pgtype.Text{String: code, Valid: true})
	if err != nil {
		return nil, &errs.Error{Code: errs.NotFound, Message: "currency not supported"}
	}

	rate, err := dbCurrency.Rate.Float64Value()
	if err != nil && !dbCurrency.Rate.Valid {
		return nil, &errs.Error{Code: errs.Internal, Message: "invalid currency rate"}
	}

	currency := &model.CurrencyInfo{
		ID:      dbCurrency.ID,
		Code:    dbCurrency.Code.String,
		Rate:    rate.Float64,
		Enabled: dbCurrency.Enabled,
	}

	if dbCurrency.Symbol.Valid {
		currency.Symbol = &dbCurrency.Symbol.String
	}

	return currency, nil
}
