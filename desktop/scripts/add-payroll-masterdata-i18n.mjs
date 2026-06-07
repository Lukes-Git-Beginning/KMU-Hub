// Adds payroll master-data (Lohn-Stammdaten) i18n keys to all 4 locales.
// Flat dotted keys; appends only missing keys to preserve existing order.
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const dir = join(dirname(fileURLToPath(import.meta.url)), '..', 'src', 'renderer', 'src', 'i18n', 'messages')

const T = {
  'team.payroll.masterData.title': { de: 'Lohn-Stammdaten', en: 'Payroll master data', fr: 'Données de paie', it: 'Dati anagrafici busta paga' },
  'team.payroll.masterData.complete': { de: 'Vollständig', en: 'Complete', fr: 'Complet', it: 'Completo' },
  'team.payroll.masterData.incomplete': { de: 'Unvollständig', en: 'Incomplete', fr: 'Incomplet', it: 'Incompleto' },
  'team.payroll.masterData.dsgvoHint': {
    de: 'Besondere Personaldaten — nur für HR/Lohn sichtbar. Aufbewahrung 6 Jahre nach Austritt (§41 EStG).',
    en: 'Special personnel data — visible to HR/payroll only. Retained 6 years after departure (§41 EStG).',
    fr: 'Données personnelles sensibles — visibles uniquement par les RH/la paie. Conservation 6 ans après le départ (§41 EStG).',
    it: 'Dati personali particolari — visibili solo a HR/buste paga. Conservazione 6 anni dopo la cessazione (§41 EStG).',
  },
  'team.payroll.masterData.groups.personal': { de: 'Persönlich', en: 'Personal', fr: 'Personnel', it: 'Personale' },
  'team.payroll.masterData.groups.tax': { de: 'Steuer', en: 'Tax', fr: 'Impôts', it: 'Imposte' },
  'team.payroll.masterData.groups.socialSecurity': { de: 'Sozialversicherung', en: 'Social security', fr: 'Sécurité sociale', it: 'Previdenza sociale' },
  'team.payroll.masterData.groups.employment': { de: 'Beschäftigung', en: 'Employment', fr: 'Emploi', it: 'Rapporto di lavoro' },
  'team.payroll.masterData.groups.compensation': { de: 'Bezüge & Bank', en: 'Compensation & bank', fr: 'Rémunération et banque', it: 'Retribuzione e banca' },

  'team.payroll.masterData.fields.birthDate': { de: 'Geburtsdatum', en: 'Date of birth', fr: 'Date de naissance', it: 'Data di nascita' },
  'team.payroll.masterData.fields.birthPlace': { de: 'Geburtsort', en: 'Place of birth', fr: 'Lieu de naissance', it: 'Luogo di nascita' },
  'team.payroll.masterData.fields.gender': { de: 'Geschlecht', en: 'Gender', fr: 'Sexe', it: 'Sesso' },
  'team.payroll.masterData.fields.nationality': { de: 'Staatsangehörigkeit', en: 'Nationality', fr: 'Nationalité', it: 'Cittadinanza' },
  'team.payroll.masterData.fields.maritalStatus': { de: 'Familienstand', en: 'Marital status', fr: 'État civil', it: 'Stato civile' },
  'team.payroll.masterData.fields.taxId': { de: 'Steuer-ID (IdNr)', en: 'Tax ID', fr: 'Numéro fiscal', it: 'Codice fiscale' },
  'team.payroll.masterData.fields.taxClass': { de: 'Steuerklasse', en: 'Tax class', fr: 'Classe d’imposition', it: 'Classe fiscale' },
  'team.payroll.masterData.fields.childAllowances': { de: 'Kinderfreibeträge', en: 'Child allowances', fr: 'Parts pour enfants', it: 'Detrazioni per figli' },
  'team.payroll.masterData.fields.confession': { de: 'Konfession', en: 'Religion', fr: 'Confession', it: 'Confessione' },
  'team.payroll.masterData.fields.svNumber': { de: 'SV-Nummer', en: 'Social security no.', fr: 'N° de sécurité sociale', it: 'Numero previdenziale' },
  'team.payroll.masterData.fields.healthInsurance': { de: 'Krankenkasse', en: 'Health insurer', fr: 'Caisse maladie', it: 'Cassa malattia' },
  'team.payroll.masterData.fields.healthInsuranceNr': { de: 'Betriebsnummer KK', en: 'Insurer company no.', fr: 'N° d’établissement caisse', it: 'Numero azienda cassa' },
  'team.payroll.masterData.fields.svStatus': { de: 'SV-Status', en: 'SS status', fr: 'Statut sécurité sociale', it: 'Stato previdenziale' },
  'team.payroll.masterData.fields.parentProperty': { de: 'Elterneigenschaft', en: 'Parental status', fr: 'Statut parental', it: 'Stato genitoriale' },
  'team.payroll.masterData.fields.weeklyHours': { de: 'Wochenarbeitszeit (Std)', en: 'Weekly hours', fr: 'Heures hebdomadaires', it: 'Ore settimanali' },
  'team.payroll.masterData.fields.employmentType': { de: 'Beschäftigungsart', en: 'Employment type', fr: 'Type d’emploi', it: 'Tipo di impiego' },
  'team.payroll.masterData.fields.jobKey': { de: 'Tätigkeitsschlüssel', en: 'Occupation code', fr: 'Code activité', it: 'Codice attività' },
  'team.payroll.masterData.fields.endDate': { de: 'Austrittsdatum', en: 'End date', fr: 'Date de fin', it: 'Data di fine' },
  'team.payroll.masterData.fields.payType': { de: 'Entlohnungsart', en: 'Pay type', fr: 'Mode de rémunération', it: 'Tipo di retribuzione' },
  'team.payroll.masterData.fields.hourlyWage': { de: 'Stundenlohn', en: 'Hourly wage', fr: 'Salaire horaire', it: 'Paga oraria' },
  'team.payroll.masterData.fields.monthlySalary': { de: 'Bruttogehalt / Monat', en: 'Gross salary / month', fr: 'Salaire brut / mois', it: 'Stipendio lordo / mese' },
  'team.payroll.masterData.fields.specialPayments': { de: 'Sonderzahlungen', en: 'Special payments', fr: 'Paiements spéciaux', it: 'Pagamenti speciali' },
  'team.payroll.masterData.fields.payrollGroup': { de: 'Abrechnungsgruppe', en: 'Payroll group', fr: 'Groupe de paie', it: 'Gruppo di paga' },
  'team.payroll.masterData.fields.iban': { de: 'IBAN', en: 'IBAN', fr: 'IBAN', it: 'IBAN' },
  'team.payroll.masterData.fields.bic': { de: 'BIC', en: 'BIC', fr: 'BIC', it: 'BIC' },
  'team.payroll.masterData.fields.accountHolder': { de: 'Kontoinhaber', en: 'Account holder', fr: 'Titulaire du compte', it: 'Intestatario conto' },

  'team.payroll.masterData.enums.gender.m': { de: 'Männlich', en: 'Male', fr: 'Masculin', it: 'Maschile' },
  'team.payroll.masterData.enums.gender.w': { de: 'Weiblich', en: 'Female', fr: 'Féminin', it: 'Femminile' },
  'team.payroll.masterData.enums.gender.d': { de: 'Divers', en: 'Diverse', fr: 'Divers', it: 'Diverso' },
  'team.payroll.masterData.enums.marital.single': { de: 'Ledig', en: 'Single', fr: 'Célibataire', it: 'Celibe/Nubile' },
  'team.payroll.masterData.enums.marital.married': { de: 'Verheiratet', en: 'Married', fr: 'Marié(e)', it: 'Coniugato/a' },
  'team.payroll.masterData.enums.marital.divorced': { de: 'Geschieden', en: 'Divorced', fr: 'Divorcé(e)', it: 'Divorziato/a' },
  'team.payroll.masterData.enums.marital.widowed': { de: 'Verwitwet', en: 'Widowed', fr: 'Veuf/Veuve', it: 'Vedovo/a' },
  'team.payroll.masterData.enums.taxClass.I': { de: 'I', en: 'I', fr: 'I', it: 'I' },
  'team.payroll.masterData.enums.taxClass.II': { de: 'II', en: 'II', fr: 'II', it: 'II' },
  'team.payroll.masterData.enums.taxClass.III': { de: 'III', en: 'III', fr: 'III', it: 'III' },
  'team.payroll.masterData.enums.taxClass.IV': { de: 'IV', en: 'IV', fr: 'IV', it: 'IV' },
  'team.payroll.masterData.enums.taxClass.V': { de: 'V', en: 'V', fr: 'V', it: 'V' },
  'team.payroll.masterData.enums.taxClass.VI': { de: 'VI', en: 'VI', fr: 'VI', it: 'VI' },
  'team.payroll.masterData.enums.confession.rk': { de: 'Römisch-katholisch', en: 'Roman Catholic', fr: 'Catholique romain', it: 'Cattolico romano' },
  'team.payroll.masterData.enums.confession.ev': { de: 'Evangelisch', en: 'Protestant', fr: 'Protestant', it: 'Protestante' },
  'team.payroll.masterData.enums.confession.none': { de: 'Keine', en: 'None', fr: 'Aucune', it: 'Nessuna' },
  'team.payroll.masterData.enums.svStatus.compulsory': { de: 'Pflichtversichert', en: 'Compulsory', fr: 'Obligatoire', it: 'Obbligatorio' },
  'team.payroll.masterData.enums.svStatus.voluntary': { de: 'Freiwillig versichert', en: 'Voluntary', fr: 'Volontaire', it: 'Volontario' },
  'team.payroll.masterData.enums.svStatus.private': { de: 'Privat versichert', en: 'Private', fr: 'Privé', it: 'Privato' },
  'team.payroll.masterData.enums.svStatus.minijobFlat': { de: 'Minijob (pauschal)', en: 'Minijob (flat-rate)', fr: 'Minijob (forfait)', it: 'Minijob (forfettario)' },
  'team.payroll.masterData.enums.employment.fulltime': { de: 'Vollzeit', en: 'Full-time', fr: 'Temps plein', it: 'Tempo pieno' },
  'team.payroll.masterData.enums.employment.parttime': { de: 'Teilzeit', en: 'Part-time', fr: 'Temps partiel', it: 'Tempo parziale' },
  'team.payroll.masterData.enums.employment.minijob': { de: 'Minijob', en: 'Minijob', fr: 'Minijob', it: 'Minijob' },
  'team.payroll.masterData.enums.employment.midijob': { de: 'Midijob', en: 'Midijob', fr: 'Midijob', it: 'Midijob' },
  'team.payroll.masterData.enums.employment.werkstudent': { de: 'Werkstudent', en: 'Working student', fr: 'Étudiant salarié', it: 'Studente lavoratore' },
  'team.payroll.masterData.enums.employment.azubi': { de: 'Auszubildende:r', en: 'Apprentice', fr: 'Apprenti(e)', it: 'Apprendista' },
  'team.payroll.masterData.enums.payType.fixed': { de: 'Festgehalt', en: 'Fixed salary', fr: 'Salaire fixe', it: 'Stipendio fisso' },
  'team.payroll.masterData.enums.payType.hourly': { de: 'Stundenlohn', en: 'Hourly wage', fr: 'Salaire horaire', it: 'Paga oraria' },

  'team.payroll.run.incompleteWarning': {
    de: '{count} unvollständig',
    en: '{count} incomplete',
    fr: '{count} incomplet(s)',
    it: '{count} incompleti',
  },
  'team.payroll.run.incompleteRow': {
    de: 'Lohn-Stammdaten unvollständig',
    en: 'Payroll master data incomplete',
    fr: 'Données de paie incomplètes',
    it: 'Dati busta paga incompleti',
  },
}

for (const lang of ['de', 'en', 'fr', 'it']) {
  const file = join(dir, `${lang}.json`)
  const json = JSON.parse(readFileSync(file, 'utf8'))
  let added = 0
  for (const [key, vals] of Object.entries(T)) {
    if (!(key in json)) {
      json[key] = vals[lang]
      added++
    }
  }
  writeFileSync(file, JSON.stringify(json, null, 2) + '\n', 'utf8')
  console.log(`${lang}: +${added} keys`)
}
