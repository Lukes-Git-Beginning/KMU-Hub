import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'

const dir = resolve('src/renderer/src/i18n/messages')
const anchor = 'kontakte.detail.tags'

// All advisory (Beratungsprotokoll) keys + the new contact-detail tab label.
// ICU single-brace interpolation: {count}, {index}, {sri}.
const K = {
  'kontakte.detail.tabOverview': { de: 'Übersicht', en: 'Overview', fr: 'Aperçu', it: 'Panoramica' },

  // Tab / history
  'advisory.tab.title': { de: 'Beratungsprotokolle', en: 'Advisory protocols', fr: 'Comptes rendus de conseil', it: 'Verbali di consulenza' },
  'advisory.tab.subtitle': { de: 'Dokumentierte Beratungsgespräche', en: 'Documented advisory sessions', fr: 'Entretiens de conseil documentés', it: 'Colloqui di consulenza documentati' },
  'advisory.tab.new': { de: 'Neues Protokoll', en: 'New protocol', fr: 'Nouveau compte rendu', it: 'Nuovo verbale' },
  'advisory.tab.emptyTitle': { de: 'Noch keine Protokolle', en: 'No protocols yet', fr: 'Aucun compte rendu', it: 'Nessun verbale' },
  'advisory.tab.emptyBody': { de: 'Halte das erste Beratungsgespräch rechtssicher fest.', en: 'Capture the first advisory session in a compliant record.', fr: 'Consignez le premier entretien de manière conforme.', it: 'Registra il primo colloquio in modo conforme.' },
  'advisory.tab.undated': { de: 'Ohne Datum', en: 'Undated', fr: 'Sans date', it: 'Senza data' },
  'advisory.tab.noAdvisor': { de: 'Kein Berater erfasst', en: 'No advisor recorded', fr: 'Aucun conseiller renseigné', it: 'Nessun consulente indicato' },

  // Editor chrome
  'advisory.title': { de: 'Beratungsprotokoll', en: 'Advisory protocol', fr: 'Compte rendu de conseil', it: 'Verbale di consulenza' },
  'advisory.notFound': { de: 'Protokoll nicht gefunden.', en: 'Protocol not found.', fr: 'Compte rendu introuvable.', it: 'Verbale non trovato.' },
  'advisory.saved': { de: 'Protokoll gespeichert', en: 'Protocol saved', fr: 'Compte rendu enregistré', it: 'Verbale salvato' },
  'advisory.finalize': { de: 'Abschließen', en: 'Finalize', fr: 'Finaliser', it: 'Finalizza' },
  'advisory.finalized': { de: 'Protokoll abgeschlossen', en: 'Protocol finalized', fr: 'Compte rendu finalisé', it: 'Verbale finalizzato' },
  'advisory.status.draft': { de: 'Entwurf', en: 'Draft', fr: 'Brouillon', it: 'Bozza' },
  'advisory.status.finalized': { de: 'Abgeschlossen', en: 'Finalized', fr: 'Finalisé', it: 'Finalizzato' },
  'advisory.readOnlyHint': { de: 'Abgeschlossene Protokolle sind unveränderbar (gesetzliche Dokumentationspflicht).', en: 'Finalized protocols are immutable (statutory documentation duty).', fr: 'Les comptes rendus finalisés sont immuables (obligation légale de documentation).', it: 'I verbali finalizzati sono immutabili (obbligo legale di documentazione).' },
  'advisory.minutes': { de: '{count} Min.', en: '{count} min', fr: '{count} min', it: '{count} min' },
  'advisory.retentionNotice': { de: 'Dieses Protokoll unterliegt einer Aufbewahrungspflicht von 10 Jahren und wird dem Kunden ausgehändigt (DSGVO Art. 6 Abs. 1 lit. c).', en: 'This protocol is subject to a 10-year retention duty and is handed to the customer (GDPR Art. 6(1)(c)).', fr: 'Ce compte rendu est soumis à une conservation de 10 ans et remis au client (RGPD art. 6, §1, c).', it: 'Questo verbale è soggetto a conservazione decennale e consegnato al cliente (GDPR art. 6, par. 1, lett. c).' },
  'advisory.confirmFinalize.title': { de: 'Protokoll abschließen?', en: 'Finalize protocol?', fr: 'Finaliser le compte rendu ?', it: 'Finalizzare il verbale?' },
  'advisory.confirmFinalize.body': { de: 'Nach dem Abschließen ist das Protokoll unveränderbar und gilt als ausgehändigt. Dieser Schritt kann nicht rückgängig gemacht werden.', en: 'Once finalized the protocol becomes immutable and counts as handed over. This step cannot be undone.', fr: 'Une fois finalisé, le compte rendu devient immuable et est considéré comme remis. Cette action est irréversible.', it: 'Una volta finalizzato, il verbale diventa immutabile e si considera consegnato. Questa azione è irreversibile.' },

  // Section titles
  'advisory.section.head': { de: 'Gesprächskopf', en: 'Session header', fr: 'En-tête d’entretien', it: 'Intestazione colloquio' },
  'advisory.section.customer': { de: 'Kunde & Profil', en: 'Customer & profile', fr: 'Client et profil', it: 'Cliente e profilo' },
  'advisory.section.knowledge': { de: 'Kenntnisse & Erfahrungen', en: 'Knowledge & experience', fr: 'Connaissances et expérience', it: 'Conoscenze ed esperienza' },
  'advisory.section.financial': { de: 'Finanzielle Situation', en: 'Financial situation', fr: 'Situation financière', it: 'Situazione finanziaria' },
  'advisory.section.goals': { de: 'Anlageziele & Risikoprofil', en: 'Goals & risk profile', fr: 'Objectifs et profil de risque', it: 'Obiettivi e profilo di rischio' },
  'advisory.section.products': { de: 'Besprochene Produkte', en: 'Discussed products', fr: 'Produits abordés', it: 'Prodotti trattati' },
  'advisory.section.recommendation': { de: 'Empfehlung & Geeignetheit', en: 'Recommendation & suitability', fr: 'Recommandation et adéquation', it: 'Raccomandazione e adeguatezza' },
  'advisory.section.compliance': { de: 'Abschluss & Compliance', en: 'Closing & compliance', fr: 'Clôture et conformité', it: 'Chiusura e conformità' },

  // Section 1
  'advisory.field.date': { de: 'Beratungsdatum', en: 'Advisory date', fr: 'Date du conseil', it: 'Data della consulenza' },
  'advisory.field.timeFrom': { de: 'Uhrzeit von', en: 'Time from', fr: 'Heure de début', it: 'Ora inizio' },
  'advisory.field.timeTo': { de: 'Uhrzeit bis', en: 'Time to', fr: 'Heure de fin', it: 'Ora fine' },
  'advisory.field.duration': { de: 'Dauer', en: 'Duration', fr: 'Durée', it: 'Durata' },
  'advisory.field.durationHint': { de: 'Wird automatisch berechnet', en: 'Calculated automatically', fr: 'Calculée automatiquement', it: 'Calcolata automaticamente' },
  'advisory.field.location': { de: 'Ort', en: 'Location', fr: 'Lieu', it: 'Luogo' },
  'advisory.field.advisor': { de: 'Berater', en: 'Advisor', fr: 'Conseiller', it: 'Consulente' },
  'advisory.field.occasion': { de: 'Anlass', en: 'Occasion', fr: 'Motif', it: 'Occasione' },
  'advisory.field.occasionNote': { de: 'Anlass (Freitext)', en: 'Occasion (free text)', fr: 'Motif (texte libre)', it: 'Occasione (testo libero)' },
  'advisory.field.customerCategory': { de: 'Kundenkategorie', en: 'Customer category', fr: 'Catégorie de client', it: 'Categoria cliente' },

  // Section 2
  'advisory.field.birthDate': { de: 'Geburtsdatum', en: 'Date of birth', fr: 'Date de naissance', it: 'Data di nascita' },
  'advisory.field.maritalStatus': { de: 'Familienstand', en: 'Marital status', fr: 'État civil', it: 'Stato civile' },
  'advisory.field.taxStatus': { de: 'Steuerstatus', en: 'Tax status', fr: 'Statut fiscal', it: 'Stato fiscale' },

  // Section 3
  'advisory.field.knownAssetClasses': { de: 'Bekannte Anlagearten', en: 'Known asset classes', fr: 'Classes d’actifs connues', it: 'Classi di attività note' },
  'advisory.field.pastTransactions': { de: 'Zurückliegende Transaktionen', en: 'Past transactions', fr: 'Transactions passées', it: 'Transazioni passate' },
  'advisory.field.pastTransactionsHint': { de: 'Art, Häufigkeit, Zeitraum', en: 'Type, frequency, period', fr: 'Type, fréquence, période', it: 'Tipo, frequenza, periodo' },
  'advisory.field.financialEducation': { de: 'Finanzrelevante Bildung', en: 'Finance-relevant education', fr: 'Formation en finance', it: 'Formazione finanziaria' },
  'advisory.field.professionalExperience': { de: 'Berufserfahrung', en: 'Professional experience', fr: 'Expérience professionnelle', it: 'Esperienza professionale' },
  'advisory.field.selfAssessment': { de: 'Selbsteinschätzung', en: 'Self-assessment', fr: 'Auto-évaluation', it: 'Autovalutazione' },
  'advisory.field.selfAssessmentHint': { de: 'Skala 1–5', en: 'Scale 1–5', fr: 'Échelle 1–5', it: 'Scala 1–5' },

  // Section 4
  'advisory.field.monthlyNetIncome': { de: 'Nettoeinkommen / Monat (€)', en: 'Net income / month (€)', fr: 'Revenu net / mois (€)', it: 'Reddito netto / mese (€)' },
  'advisory.field.recurringLiabilities': { de: 'Regelm. Verbindlichkeiten (€)', en: 'Recurring liabilities (€)', fr: 'Engagements récurrents (€)', it: 'Passività ricorrenti (€)' },
  'advisory.field.liquidAssets': { de: 'Liquide Mittel (€)', en: 'Liquid assets (€)', fr: 'Liquidités (€)', it: 'Liquidità (€)' },
  'advisory.field.currentInvestments': { de: 'Kapitalanlagen aktuell (€)', en: 'Current investments (€)', fr: 'Placements actuels (€)', it: 'Investimenti attuali (€)' },
  'advisory.field.realEstate': { de: 'Immobilien', en: 'Real estate', fr: 'Biens immobiliers', it: 'Immobili' },
  'advisory.field.existingInsurance': { de: 'Bestehende Versicherungen / Vorsorge', en: 'Existing insurance / provision', fr: 'Assurances / prévoyance existantes', it: 'Assicurazioni / previdenza esistenti' },
  'advisory.field.maxLossAbs': { de: 'Max. Verlusttragfähigkeit (€)', en: 'Max. loss capacity (€)', fr: 'Capacité de perte max. (€)', it: 'Capacità di perdita max. (€)' },
  'advisory.field.maxLossPct': { de: 'Max. Verlusttragfähigkeit (%)', en: 'Max. loss capacity (%)', fr: 'Capacité de perte max. (%)', it: 'Capacità di perdita max. (%)' },

  // Section 5
  'advisory.field.investmentPurpose': { de: 'Anlagezweck', en: 'Investment purpose', fr: 'Objectif de placement', it: 'Scopo dell’investimento' },
  'advisory.field.horizon': { de: 'Anlagehorizont', en: 'Investment horizon', fr: 'Horizon de placement', it: 'Orizzonte d’investimento' },
  'advisory.field.horizonPlaceholder': { de: 'Horizont wählen', en: 'Select horizon', fr: 'Choisir l’horizon', it: 'Seleziona orizzonte' },
  'advisory.field.riskClass': { de: 'Risikoklasse (SRI)', en: 'Risk class (SRI)', fr: 'Classe de risque (SRI)', it: 'Classe di rischio (SRI)' },
  'advisory.field.riskTolerance': { de: 'Risikobereitschaft', en: 'Risk tolerance', fr: 'Tolérance au risque', it: 'Propensione al rischio' },
  'advisory.field.riskCapacity': { de: 'Risikotragfähigkeit (objektiv)', en: 'Risk capacity (objective)', fr: 'Capacité de risque (objective)', it: 'Capacità di rischio (oggettiva)' },
  'advisory.field.oneTimeAmount': { de: 'Anlagebetrag einmalig (€)', en: 'One-time amount (€)', fr: 'Montant unique (€)', it: 'Importo una tantum (€)' },
  'advisory.field.monthlySavings': { de: 'Sparrate / Monat (€)', en: 'Savings rate / month (€)', fr: 'Épargne / mois (€)', it: 'Risparmio / mese (€)' },
  'advisory.field.esgPreference': { de: 'Nachhaltigkeitspräferenz (ESG)', en: 'Sustainability preference (ESG)', fr: 'Préférence durabilité (ESG)', it: 'Preferenza sostenibilità (ESG)' },
  'advisory.field.esgHint': { de: 'Seit 08/2022 abfragepflichtig (EU-DelegVO 2021/1253).', en: 'Mandatory to ask since 08/2022 (EU Deleg. Reg. 2021/1253).', fr: 'Obligatoire depuis 08/2022 (règl. délég. UE 2021/1253).', it: 'Obbligatoria da 08/2022 (reg. deleg. UE 2021/1253).' },
  'advisory.field.esgDetailsPlaceholder': { de: 'SFDR / Taxonomie / PAI', en: 'SFDR / taxonomy / PAI', fr: 'SFDR / taxonomie / PAI', it: 'SFDR / tassonomia / PAI' },

  // Section 7
  'advisory.field.recommendationSummary': { de: 'Empfehlungszusammenfassung', en: 'Recommendation summary', fr: 'Résumé de la recommandation', it: 'Sintesi della raccomandazione' },
  'advisory.field.suitabilityReasoning': { de: 'Begründung der Geeignetheit', en: 'Suitability reasoning', fr: 'Justification de l’adéquation', it: 'Motivazione dell’adeguatezza' },
  'advisory.field.suitabilityHint': { de: 'Warum passt die Empfehlung zum Kundenprofil?', en: 'Why does the recommendation fit the customer profile?', fr: 'En quoi la recommandation correspond-elle au profil ?', it: 'Perché la raccomandazione è adatta al profilo?' },
  'advisory.field.goalReference': { de: 'Bezug zu Kundenzielen', en: 'Reference to customer goals', fr: 'Lien avec les objectifs du client', it: 'Riferimento agli obiettivi del cliente' },
  'advisory.field.riskClassReference': { de: 'Bezug zur Risikoklasse', en: 'Reference to risk class', fr: 'Lien avec la classe de risque', it: 'Riferimento alla classe di rischio' },
  'advisory.field.riskClassRefHint': { de: 'Automatisch aus Abschnitt 5', en: 'Auto from section 5', fr: 'Auto depuis la section 5', it: 'Auto dalla sezione 5' },
  'advisory.field.riskClassRefValue': { de: 'Risikoklasse SRI {sri}', en: 'Risk class SRI {sri}', fr: 'Classe de risque SRI {sri}', it: 'Classe di rischio SRI {sri}' },
  'advisory.field.alternatives': { de: 'Geprüfte Alternativen', en: 'Alternatives considered', fr: 'Alternatives étudiées', it: 'Alternative valutate' },
  'advisory.field.notRecommended': { de: 'Nicht empfohlen + Grund', en: 'Not recommended + reason', fr: 'Non recommandé + motif', it: 'Non raccomandato + motivo' },

  // Section 8
  'advisory.field.mainConcerns': { de: 'Wesentliche Anliegen + Gewichtung', en: 'Main concerns + weighting', fr: 'Préoccupations principales + pondération', it: 'Esigenze principali + ponderazione' },
  'advisory.field.mainConcernsHint': { de: 'Was war dem Kunden besonders wichtig?', en: 'What mattered most to the customer?', fr: 'Qu’est-ce qui importait le plus au client ?', it: 'Cosa contava di più per il cliente?' },
  'advisory.field.warningsGiven': { de: 'Erteilte Warnhinweise', en: 'Warnings given', fr: 'Avertissements donnés', it: 'Avvertenze fornite' },
  'advisory.field.deliveryForm': { de: 'Aushändigungsform', en: 'Delivery form', fr: 'Forme de remise', it: 'Forma di consegna' },
  'advisory.field.documentDeliveredDate': { de: 'Aushändigungsdatum', en: 'Delivery date', fr: 'Date de remise', it: 'Data di consegna' },
  'advisory.field.advisorSignature': { de: 'Unterschrift Berater', en: 'Advisor signature', fr: 'Signature du conseiller', it: 'Firma del consulente' },
  'advisory.field.signatureHint': { de: 'Name in Druckbuchstaben', en: 'Name in block letters', fr: 'Nom en majuscules', it: 'Nome in stampatello' },
  'advisory.field.followupDate': { de: 'Folgeberatung am', en: 'Follow-up on', fr: 'Conseil de suivi le', it: 'Consulenza successiva il' },
  'advisory.field.documentDelivered': { de: 'Kunde hat das Dokument erhalten', en: 'Customer received the document', fr: 'Le client a reçu le document', it: 'Il cliente ha ricevuto il documento' },
  'advisory.field.customerConfirmation': { de: 'Bestätigung durch Kunde liegt vor', en: 'Customer confirmation on file', fr: 'Confirmation du client au dossier', it: 'Conferma del cliente agli atti' },
  'advisory.field.documentWaiver': { de: 'Dokumentenverzicht (separate Erklärung)', en: 'Document waiver (separate declaration)', fr: 'Renonciation au document (déclaration séparée)', it: 'Rinuncia al documento (dichiarazione separata)' },
  'advisory.field.internalNotes': { de: 'Interne Notizen', en: 'Internal notes', fr: 'Notes internes', it: 'Note interne' },
  'advisory.field.internalNotesHint': { de: 'Nicht Teil der Kundenkopie', en: 'Not part of the customer copy', fr: 'Ne figure pas sur la copie client', it: 'Non incluso nella copia cliente' },

  // Products
  'advisory.products.empty': { de: 'Noch keine Produkte erfasst.', en: 'No products recorded yet.', fr: 'Aucun produit enregistré.', it: 'Nessun prodotto registrato.' },
  'advisory.products.add': { de: 'Produkt hinzufügen', en: 'Add product', fr: 'Ajouter un produit', it: 'Aggiungi prodotto' },
  'advisory.products.item': { de: 'Produkt {index}', en: 'Product {index}', fr: 'Produit {index}', it: 'Prodotto {index}' },
  'advisory.products.name': { de: 'Produktname', en: 'Product name', fr: 'Nom du produit', it: 'Nome prodotto' },
  'advisory.products.isin': { de: 'ISIN / WKN', en: 'ISIN / WKN', fr: 'ISIN / WKN', it: 'ISIN / WKN' },
  'advisory.products.category': { de: 'Kategorie', en: 'Category', fr: 'Catégorie', it: 'Categoria' },
  'advisory.products.opportunities': { de: 'Chancen', en: 'Opportunities', fr: 'Opportunités', it: 'Opportunità' },
  'advisory.products.risks': { de: 'Risiken', en: 'Risks', fr: 'Risques', it: 'Rischi' },
  'advisory.products.costsOneTime': { de: 'Kosten einmalig (€)', en: 'One-time costs (€)', fr: 'Coûts uniques (€)', it: 'Costi una tantum (€)' },
  'advisory.products.costsRunning': { de: 'Kosten laufend p.a. (€)', en: 'Running costs p.a. (€)', fr: 'Coûts annuels (€)', it: 'Costi annui (€)' },
  'advisory.products.recommended': { de: 'Empfohlen', en: 'Recommended', fr: 'Recommandé', it: 'Raccomandato' },

  // SRI classes
  'advisory.sri.1': { de: 'konservativ', en: 'conservative', fr: 'conservateur', it: 'conservativo' },
  'advisory.sri.2': { de: 'konservativ', en: 'conservative', fr: 'conservateur', it: 'conservativo' },
  'advisory.sri.3': { de: 'ausgewogen', en: 'balanced', fr: 'équilibré', it: 'bilanciato' },
  'advisory.sri.4': { de: 'ausgewogen', en: 'balanced', fr: 'équilibré', it: 'bilanciato' },
  'advisory.sri.5': { de: 'wachstumsorientiert', en: 'growth-oriented', fr: 'axé croissance', it: 'orientato alla crescita' },
  'advisory.sri.6': { de: 'dynamisch', en: 'dynamic', fr: 'dynamique', it: 'dinamico' },
  'advisory.sri.7': { de: 'spekulativ', en: 'speculative', fr: 'spéculatif', it: 'speculativo' },

  // Locations
  'advisory.location.office': { de: 'Büro', en: 'Office', fr: 'Bureau', it: 'Ufficio' },
  'advisory.location.phone': { de: 'Telefon', en: 'Phone', fr: 'Téléphone', it: 'Telefono' },
  'advisory.location.video': { de: 'Video', en: 'Video', fr: 'Vidéo', it: 'Video' },
  'advisory.location.onsite': { de: 'Beim Kunden', en: 'On site', fr: 'Chez le client', it: 'Presso il cliente' },

  // Occasions
  'advisory.occasion.initial': { de: 'Erstberatung', en: 'Initial advice', fr: 'Premier conseil', it: 'Prima consulenza' },
  'advisory.occasion.followup': { de: 'Folgeberatung', en: 'Follow-up advice', fr: 'Conseil de suivi', it: 'Consulenza successiva' },
  'advisory.occasion.event': { de: 'Anlassberatung', en: 'Event-driven advice', fr: 'Conseil ponctuel', it: 'Consulenza occasionale' },

  // Customer category
  'advisory.customerCategory.private': { de: 'Privatkunde', en: 'Retail client', fr: 'Client particulier', it: 'Cliente al dettaglio' },
  'advisory.customerCategory.professional': { de: 'Professioneller Kunde', en: 'Professional client', fr: 'Client professionnel', it: 'Cliente professionale' },

  // Asset classes
  'advisory.assetClass.stocks': { de: 'Aktien', en: 'Stocks', fr: 'Actions', it: 'Azioni' },
  'advisory.assetClass.funds': { de: 'Fonds', en: 'Funds', fr: 'Fonds', it: 'Fondi' },
  'advisory.assetClass.etf': { de: 'ETF', en: 'ETFs', fr: 'ETF', it: 'ETF' },
  'advisory.assetClass.bonds': { de: 'Anleihen', en: 'Bonds', fr: 'Obligations', it: 'Obbligazioni' },
  'advisory.assetClass.derivatives': { de: 'Derivate', en: 'Derivatives', fr: 'Dérivés', it: 'Derivati' },
  'advisory.assetClass.realestate': { de: 'Immobilien', en: 'Real estate', fr: 'Immobilier', it: 'Immobiliare' },
  'advisory.assetClass.crypto': { de: 'Krypto', en: 'Crypto', fr: 'Crypto', it: 'Cripto' },

  // Investment purposes
  'advisory.purpose.retirement': { de: 'Altersvorsorge', en: 'Retirement', fr: 'Retraite', it: 'Pensione' },
  'advisory.purpose.liquidity': { de: 'Liquidität', en: 'Liquidity', fr: 'Liquidité', it: 'Liquidità' },
  'advisory.purpose.growth': { de: 'Wachstum', en: 'Growth', fr: 'Croissance', it: 'Crescita' },
  'advisory.purpose.saving': { de: 'Sparen', en: 'Saving', fr: 'Épargne', it: 'Risparmio' },
  'advisory.purpose.speculation': { de: 'Spekulation', en: 'Speculation', fr: 'Spéculation', it: 'Speculazione' },

  // Horizons
  'advisory.horizon.lt1': { de: 'unter 1 Jahr', en: 'under 1 year', fr: 'moins d’1 an', it: 'meno di 1 anno' },
  'advisory.horizon.1to3': { de: '1–3 Jahre', en: '1–3 years', fr: '1–3 ans', it: '1–3 anni' },
  'advisory.horizon.3to5': { de: '3–5 Jahre', en: '3–5 years', fr: '3–5 ans', it: '3–5 anni' },
  'advisory.horizon.5to10': { de: '5–10 Jahre', en: '5–10 years', fr: '5–10 ans', it: '5–10 anni' },
  'advisory.horizon.gt10': { de: 'über 10 Jahre', en: 'over 10 years', fr: 'plus de 10 ans', it: 'oltre 10 anni' },

  // Delivery forms
  'advisory.deliveryForm.paper': { de: 'Papier', en: 'Paper', fr: 'Papier', it: 'Cartaceo' },
  'advisory.deliveryForm.email': { de: 'E-Mail', en: 'E-mail', fr: 'E-mail', it: 'E-mail' },
  'advisory.deliveryForm.portal': { de: 'Kundenportal', en: 'Customer portal', fr: 'Portail client', it: 'Portale cliente' },

  // Warnings checklist
  'advisory.warning.risk': { de: 'Risikohinweise zu den Produkten', en: 'Product risk disclosures', fr: 'Avertissements sur les risques produits', it: 'Avvertenze sui rischi dei prodotti' },
  'advisory.warning.costs': { de: 'Kosten- und Gebührentransparenz (ex-ante)', en: 'Cost & fee transparency (ex-ante)', fr: 'Transparence des coûts (ex-ante)', it: 'Trasparenza dei costi (ex-ante)' },
  'advisory.warning.conflicts': { de: 'Hinweis auf Interessenkonflikte', en: 'Conflict-of-interest notice', fr: 'Avis sur les conflits d’intérêts', it: 'Avviso sui conflitti di interesse' },
  'advisory.warning.pastPerformance': { de: 'Keine Garantie aus Wertentwicklung der Vergangenheit', en: 'No guarantee from past performance', fr: 'Pas de garantie liée aux performances passées', it: 'Nessuna garanzia da performance passate' },
  'advisory.warning.liquidity': { de: 'Hinweis auf Liquiditäts-/Verfügbarkeitsrisiken', en: 'Liquidity / availability risk notice', fr: 'Avis sur les risques de liquidité', it: 'Avviso sui rischi di liquidità' },
}

const report = {}
for (const loc of ['de', 'en', 'fr', 'it']) {
  const file = resolve(dir, `${loc}.json`)
  const obj = JSON.parse(readFileSync(file, 'utf8'))
  let lines = readFileSync(file, 'utf8').split('\n')
  const nk = Object.keys(K).filter((k) => !(k in obj)).sort()
  if (!nk.length) { report[loc] = 0; continue }
  const block = nk.map((k) => `  ${JSON.stringify(k)}: ${JSON.stringify(K[k][loc])},`)
  const idx = lines.findIndex((l) => l.trimStart().startsWith(`"${anchor}":`))
  if (idx === -1) throw new Error(`anchor ${anchor} missing in ${loc}`)
  lines = [...lines.slice(0, idx + 1), ...block, ...lines.slice(idx + 1)]
  const out = lines.join('\n')
  JSON.parse(out) // validate
  writeFileSync(file, out, 'utf8')
  report[loc] = block.length
}
console.log(JSON.stringify(report))
