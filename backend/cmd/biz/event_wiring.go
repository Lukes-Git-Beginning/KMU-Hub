package main

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/biz/invoice"
	"github.com/kmuhub/kmuhub/internal/biz/quote"
)

type invoiceEventSetter interface {
	SetEventEmitter(invoice.EventEmitter)
}

type quoteEventSetter interface {
	SetEventEmitter(quote.EventEmitter)
}

// wireFinanceEventEmitters connects invoice and quote to the event bus. Without
// it emitEvent is a no-op and the automation triggers biz.invoice.sent and
// biz.quote.created (see automation/trigger/registry.go) never fire.
// lexware keeps its noopEmitter on purpose: no trigger listens on lexware.*.
func wireFinanceEventEmitters(pool *pgxpool.Pool, invoiceSvc invoiceEventSetter, quoteSvc quoteEventSetter) {
	invoiceSvc.SetEventEmitter(invoice.NewPGEventEmitter(pool))
	quoteSvc.SetEventEmitter(quote.NewPGEventEmitter(pool))
}
