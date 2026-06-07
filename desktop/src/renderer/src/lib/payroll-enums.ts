/**
 * Payroll master-data enumerations (DATEV-Lohn / SV-Anmeldung).
 *
 * Shared value lists for the "Lohn-Stammdaten" section on the employee profile.
 * Each option carries an i18n key (label rendered via t()), never a hard string.
 * Mirrors the DATEV Personalstammdaten topic areas — see team-lohn-stammdaten-spec.md.
 */

export type Gender = 'm' | 'w' | 'd'
export type MaritalStatus = 'single' | 'married' | 'divorced' | 'widowed'
export type TaxClass = 'I' | 'II' | 'III' | 'IV' | 'V' | 'VI'
export type Confession = 'rk' | 'ev' | 'none'
export type SvStatus = 'compulsory' | 'voluntary' | 'private' | 'minijob_flat'
export type EmploymentType =
  | 'fulltime'
  | 'parttime'
  | 'minijob'
  | 'midijob'
  | 'werkstudent'
  | 'azubi'
export type PayType = 'fixed' | 'hourly'

export interface EnumOption<T extends string> {
  value: T
  labelKey: string
}

export const GENDER_OPTIONS: EnumOption<Gender>[] = [
  { value: 'm', labelKey: 'team.payroll.masterData.enums.gender.m' },
  { value: 'w', labelKey: 'team.payroll.masterData.enums.gender.w' },
  { value: 'd', labelKey: 'team.payroll.masterData.enums.gender.d' },
]

export const MARITAL_STATUS_OPTIONS: EnumOption<MaritalStatus>[] = [
  { value: 'single', labelKey: 'team.payroll.masterData.enums.marital.single' },
  { value: 'married', labelKey: 'team.payroll.masterData.enums.marital.married' },
  { value: 'divorced', labelKey: 'team.payroll.masterData.enums.marital.divorced' },
  { value: 'widowed', labelKey: 'team.payroll.masterData.enums.marital.widowed' },
]

// Steuerklasse — Roman numerals are the legal labels, no translation needed.
export const TAX_CLASS_OPTIONS: EnumOption<TaxClass>[] = [
  { value: 'I', labelKey: 'team.payroll.masterData.enums.taxClass.I' },
  { value: 'II', labelKey: 'team.payroll.masterData.enums.taxClass.II' },
  { value: 'III', labelKey: 'team.payroll.masterData.enums.taxClass.III' },
  { value: 'IV', labelKey: 'team.payroll.masterData.enums.taxClass.IV' },
  { value: 'V', labelKey: 'team.payroll.masterData.enums.taxClass.V' },
  { value: 'VI', labelKey: 'team.payroll.masterData.enums.taxClass.VI' },
]

export const CONFESSION_OPTIONS: EnumOption<Confession>[] = [
  { value: 'rk', labelKey: 'team.payroll.masterData.enums.confession.rk' },
  { value: 'ev', labelKey: 'team.payroll.masterData.enums.confession.ev' },
  { value: 'none', labelKey: 'team.payroll.masterData.enums.confession.none' },
]

export const SV_STATUS_OPTIONS: EnumOption<SvStatus>[] = [
  { value: 'compulsory', labelKey: 'team.payroll.masterData.enums.svStatus.compulsory' },
  { value: 'voluntary', labelKey: 'team.payroll.masterData.enums.svStatus.voluntary' },
  { value: 'private', labelKey: 'team.payroll.masterData.enums.svStatus.private' },
  { value: 'minijob_flat', labelKey: 'team.payroll.masterData.enums.svStatus.minijobFlat' },
]

export const EMPLOYMENT_TYPE_OPTIONS: EnumOption<EmploymentType>[] = [
  { value: 'fulltime', labelKey: 'team.payroll.masterData.enums.employment.fulltime' },
  { value: 'parttime', labelKey: 'team.payroll.masterData.enums.employment.parttime' },
  { value: 'minijob', labelKey: 'team.payroll.masterData.enums.employment.minijob' },
  { value: 'midijob', labelKey: 'team.payroll.masterData.enums.employment.midijob' },
  { value: 'werkstudent', labelKey: 'team.payroll.masterData.enums.employment.werkstudent' },
  { value: 'azubi', labelKey: 'team.payroll.masterData.enums.employment.azubi' },
]

export const PAY_TYPE_OPTIONS: EnumOption<PayType>[] = [
  { value: 'fixed', labelKey: 'team.payroll.masterData.enums.payType.fixed' },
  { value: 'hourly', labelKey: 'team.payroll.masterData.enums.payType.hourly' },
]
