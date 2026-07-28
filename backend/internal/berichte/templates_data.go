package berichte

// demoTemplates mirrors the frontend's former DEMO_TEMPLATES mock
// (desktop/src/renderer/src/mocks/handlers/berichte.ts) block-for-block, so
// "Neuer Bericht aus Vorlage" produces the same starter documents that used
// to come from MSW. Rows/settings follow the same opaque-JSONB contract as
// Document (see normalizeDocumentRows): the block tree is frontend-owned.
//
// One deviation from the frontend source: tpl-projekt's module is "cross"
// here, not "work" — the backend module enum (isValidModule, matching the
// report_documents CHECK constraint) only allows
// finanzen|crm|helpdesk|inventar|produktion|cross. "work" is a frontend-only
// ReportModule value with no backend equivalent; creating a document from
// this template must pass the module check, so it is remapped at the
// template level. Block-internal fields (e.g. kpi.source: "work") are opaque
// JSONB and untouched.
var demoTemplates = []Template{
	{
		ID:          "tpl-monat",
		Title:       "Monats-/Managementbericht",
		Description: "Deckblatt, Zusammenfassung, Kennzahlen, Finanzen, Empfehlung.",
		Module:      "cross",
		Icon:        templateIcon("CalendarRange"),
		Settings:    []byte(templateDefaultSettings),
		Rows: []byte(`[
			{"id":"row-1","columns":[{"id":"col-1","blocks":[
				{"id":"cover-1","type":"cover","title":"Monatsbericht","subtitle":"Monat Jahr","showDate":true}
			]}]},
			{"id":"row-2","columns":[{"id":"col-2","blocks":[
				{"id":"h-1","type":"heading","level":1,"text":"Zusammenfassung"}
			]}]},
			{"id":"row-3","columns":[{"id":"col-3","blocks":[
				{"id":"t-1","type":"text","html":"<p>Kernaussage des Monats in zwei bis drei Sätzen.</p>"}
			]}]},
			{"id":"row-4","columns":[
				{"id":"col-4","blocks":[{"id":"kpi-1","type":"kpi","label":"Umsatz","value":"—","unit":"€","changePercent":null,"source":"finanzen"}]},
				{"id":"col-5","blocks":[{"id":"kpi-2","type":"kpi","label":"Kosten","value":"—","unit":"€","changePercent":null,"source":"finanzen"}]},
				{"id":"col-6","blocks":[{"id":"kpi-3","type":"kpi","label":"Ergebnis","value":"—","unit":"€","changePercent":null,"source":"finanzen"}]}
			]},
			{"id":"row-5","columns":[{"id":"col-7","blocks":[
				{"id":"h-2","type":"heading","level":2,"text":"Finanzen"}
			]}]},
			{"id":"row-6","columns":[{"id":"col-8","blocks":[
				{"id":"c-1","type":"chart","caption":"Umsatz nach Monat"}
			]}]},
			{"id":"row-7","columns":[{"id":"col-9","blocks":[
				{"id":"cl-1","type":"callout","variant":"recommendation","title":"Empfehlung","html":"<p>Maßnahmen für den nächsten Monat.</p>"}
			]}]}
		]`),
	},
	{
		ID:          "tpl-sales",
		Title:       "Vertriebsbericht",
		Description: "Kennzahlen, Pipeline-Trichter, Abschlüsse, nächste Schritte.",
		Module:      "crm",
		Icon:        templateIcon("TrendingUp"),
		Settings:    []byte(templateDefaultSettings),
		Rows: []byte(`[
			{"id":"row-1","columns":[{"id":"col-1","blocks":[
				{"id":"cover-1","type":"cover","title":"Vertriebsbericht","subtitle":"Periode","showDate":true}
			]}]},
			{"id":"row-2","columns":[
				{"id":"col-2","blocks":[{"id":"kpi-1","type":"kpi","label":"Umsatz","value":"—","unit":"€","changePercent":null,"source":"crm"}]},
				{"id":"col-3","blocks":[{"id":"kpi-2","type":"kpi","label":"Gewinnrate","value":"—","unit":"%","changePercent":null,"source":"crm"}]},
				{"id":"col-4","blocks":[{"id":"kpi-3","type":"kpi","label":"Pipeline","value":"—","unit":"€","changePercent":null,"source":"crm"}]}
			]},
			{"id":"row-3","columns":[{"id":"col-5","blocks":[
				{"id":"h-1","type":"heading","level":2,"text":"Pipeline"}
			]}]},
			{"id":"row-4","columns":[{"id":"col-6","blocks":[
				{"id":"c-1","type":"chart","caption":"Pipeline nach Phase"}
			]}]},
			{"id":"row-5","columns":[{"id":"col-7","blocks":[
				{"id":"tb-1","type":"table","caption":"Abschlüsse der Periode"}
			]}]},
			{"id":"row-6","columns":[{"id":"col-8","blocks":[
				{"id":"cl-1","type":"callout","variant":"recommendation","title":"Nächste Schritte","html":"<p>Top-3-Maßnahmen.</p>"}
			]}]}
		]`),
	},
	{
		ID:          "tpl-bi",
		Title:       "Analyse-Bericht",
		Description: "Kennzahlen-Übersicht, Zeitreihe, Vergleiche, Detaildaten.",
		Module:      "cross",
		Icon:        templateIcon("BarChart3"),
		Settings:    []byte(templateDefaultSettings),
		Rows: []byte(`[
			{"id":"row-1","columns":[{"id":"col-1","blocks":[
				{"id":"cover-1","type":"cover","title":"Analyse-Bericht","subtitle":"Zeitraum","showDate":true}
			]}]},
			{"id":"row-2","columns":[
				{"id":"col-2","blocks":[{"id":"kpi-1","type":"kpi","label":"Kennzahl A","value":"—","unit":"","changePercent":null,"source":"cross"}]},
				{"id":"col-3","blocks":[{"id":"kpi-2","type":"kpi","label":"Kennzahl B","value":"—","unit":"","changePercent":null,"source":"cross"}]},
				{"id":"col-4","blocks":[{"id":"kpi-3","type":"kpi","label":"Kennzahl C","value":"—","unit":"","changePercent":null,"source":"cross"}]}
			]},
			{"id":"row-3","columns":[{"id":"col-5","blocks":[
				{"id":"c-1","type":"chart","caption":"Entwicklung über Zeit"}
			]}]},
			{"id":"row-4","columns":[
				{"id":"col-6","blocks":[{"id":"c-2","type":"chart","caption":"Verteilung A"}]},
				{"id":"col-7","blocks":[{"id":"c-3","type":"chart","caption":"Verteilung B"}]}
			]},
			{"id":"row-5","columns":[{"id":"col-8","blocks":[
				{"id":"tb-1","type":"table","caption":"Detaildaten"}
			]}]}
		]`),
	},
	{
		ID:          "tpl-projekt",
		Title:       "Projektbericht",
		Description: "Status, Fortschritt, Budget, Meilensteine, Risiken.",
		Module:      "cross", // frontend template uses "work" — see file doc comment
		Icon:        templateIcon("ClipboardList"),
		Settings:    []byte(templateDefaultSettings),
		Rows: []byte(`[
			{"id":"row-1","columns":[{"id":"col-1","blocks":[
				{"id":"cover-1","type":"cover","title":"Projektbericht","subtitle":"Projektname","showDate":true}
			]}]},
			{"id":"row-2","columns":[{"id":"col-2","blocks":[
				{"id":"h-1","type":"heading","level":1,"text":"Status & Fortschritt"}
			]}]},
			{"id":"row-3","columns":[
				{"id":"col-3","blocks":[{"id":"kpi-1","type":"kpi","label":"Fortschritt","value":"—","unit":"%","changePercent":null,"source":"work"}]},
				{"id":"col-4","blocks":[{"id":"kpi-2","type":"kpi","label":"Budget","value":"—","unit":"€","changePercent":null,"source":"work"}]}
			]},
			{"id":"row-4","columns":[{"id":"col-5","blocks":[
				{"id":"t-1","type":"text","html":"<p>Wichtigste Meilensteine und ihr Stand.</p>"}
			]}]},
			{"id":"row-5","columns":[{"id":"col-6","blocks":[
				{"id":"bl-1","type":"bullet","items":["Risiko 1","Risiko 2"]}
			]}]},
			{"id":"row-6","columns":[{"id":"col-7","blocks":[
				{"id":"cl-1","type":"callout","variant":"recommendation","title":"Maßnahmen","html":"<p>Empfohlene nächste Schritte.</p>"}
			]}]}
		]`),
	},
	{
		ID:          "tpl-exec",
		Title:       "Executive-Kurzbericht",
		Description: "Kernaussage, drei Kennzahlen, Top-Findings, Empfehlung.",
		Module:      "cross",
		Icon:        templateIcon("Zap"),
		Settings:    []byte(templateDefaultSettings),
		Rows: []byte(`[
			{"id":"row-1","columns":[{"id":"col-1","blocks":[
				{"id":"cover-1","type":"cover","title":"Executive Update","subtitle":"Datum","showDate":true}
			]}]},
			{"id":"row-2","columns":[{"id":"col-2","blocks":[
				{"id":"t-1","type":"text","html":"<p>Kernaussage zuerst: das Wichtigste in zwei Sätzen.</p>"}
			]}]},
			{"id":"row-3","columns":[
				{"id":"col-3","blocks":[{"id":"kpi-1","type":"kpi","label":"Kennzahl 1","value":"—","unit":"","changePercent":null,"source":"cross"}]},
				{"id":"col-4","blocks":[{"id":"kpi-2","type":"kpi","label":"Kennzahl 2","value":"—","unit":"","changePercent":null,"source":"cross"}]},
				{"id":"col-5","blocks":[{"id":"kpi-3","type":"kpi","label":"Kennzahl 3","value":"—","unit":"","changePercent":null,"source":"cross"}]}
			]},
			{"id":"row-4","columns":[{"id":"col-6","blocks":[
				{"id":"bl-1","type":"bullet","items":["Finding 1","Finding 2","Finding 3"]}
			]}]},
			{"id":"row-5","columns":[{"id":"col-7","blocks":[
				{"id":"cl-1","type":"callout","variant":"recommendation","title":"Empfehlung","html":"<p>Was als Nächstes zu tun ist.</p>"}
			]}]}
		]`),
	},
}

const templateDefaultSettings = `{"showHeader":true,"showFooter":true,"showPageNumbers":true}`

func templateIcon(name string) *string { return &name }
