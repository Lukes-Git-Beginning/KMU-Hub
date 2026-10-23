package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kmuhub/kmuhub/internal/biz/invoice"
	"github.com/kmuhub/kmuhub/internal/biz/quote"
)

type recordingInvoiceSetter struct{ got invoice.EventEmitter }

func (r *recordingInvoiceSetter) SetEventEmitter(e invoice.EventEmitter) { r.got = e }

type recordingQuoteSetter struct{ got quote.EventEmitter }

func (r *recordingQuoteSetter) SetEventEmitter(e quote.EventEmitter) { r.got = e }

func TestWireFinanceEventEmitters_SetsBothEmitters(t *testing.T) {
	inv := &recordingInvoiceSetter{}
	q := &recordingQuoteSetter{}

	wireFinanceEventEmitters(nil, inv, q)

	assert.IsType(t, &invoice.PGEventEmitter{}, inv.got, "invoice must emit biz.invoice.sent")
	assert.IsType(t, &quote.PGEventEmitter{}, q.got, "quote must emit biz.quote.created")
}
