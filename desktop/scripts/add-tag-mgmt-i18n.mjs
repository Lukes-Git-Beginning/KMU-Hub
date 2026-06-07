// One-off: add tag-management i18n keys (ICU single-brace syntax).
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'
const __dirname = dirname(fileURLToPath(import.meta.url))
const MSG = join(__dirname, '..', 'src', 'renderer', 'src', 'i18n', 'messages')
const add = {
  de: {
    'crm.settings.tags.title': 'Tag-Verwaltung',
    'crm.settings.tags.desc': 'Tags für Kontakte zentral anlegen, umbenennen und löschen.',
    'crm.settings.tags.count': '{count} Tags',
    'crm.settings.tags.empty': 'Noch keine Tags angelegt.',
    'crm.settings.tags.newTag': 'Neuer Tag',
    'crm.settings.tags.namePlaceholder': 'Tag-Name',
    'crm.settings.tags.addButton': 'Tag hinzufügen',
    'crm.settings.tags.added': 'Tag „{name}" hinzugefügt',
    'crm.settings.tags.deleted': 'Tag „{name}" gelöscht',
    'crm.settings.tags.deleteTitle': 'Tag löschen?',
    'crm.settings.tags.deleteDescription': 'Soll der Tag „{name}" wirklich gelöscht werden? Er wird von allen Kontakten entfernt.',
  },
  en: {
    'crm.settings.tags.title': 'Tag management',
    'crm.settings.tags.desc': 'Centrally create, rename and delete contact tags.',
    'crm.settings.tags.count': '{count} tags',
    'crm.settings.tags.empty': 'No tags yet.',
    'crm.settings.tags.newTag': 'New tag',
    'crm.settings.tags.namePlaceholder': 'Tag name',
    'crm.settings.tags.addButton': 'Add tag',
    'crm.settings.tags.added': 'Tag "{name}" added',
    'crm.settings.tags.deleted': 'Tag "{name}" deleted',
    'crm.settings.tags.deleteTitle': 'Delete tag?',
    'crm.settings.tags.deleteDescription': 'Really delete the tag "{name}"? It will be removed from all contacts.',
  },
  fr: {
    'crm.settings.tags.title': 'Gestion des tags',
    'crm.settings.tags.desc': 'Créer, renommer et supprimer les tags de contact de manière centralisée.',
    'crm.settings.tags.count': '{count} tags',
    'crm.settings.tags.empty': 'Aucun tag pour le moment.',
    'crm.settings.tags.newTag': 'Nouveau tag',
    'crm.settings.tags.namePlaceholder': 'Nom du tag',
    'crm.settings.tags.addButton': 'Ajouter un tag',
    'crm.settings.tags.added': 'Tag « {name} » ajouté',
    'crm.settings.tags.deleted': 'Tag « {name} » supprimé',
    'crm.settings.tags.deleteTitle': 'Supprimer le tag ?',
    'crm.settings.tags.deleteDescription': 'Supprimer vraiment le tag « {name} » ? Il sera retiré de tous les contacts.',
  },
  it: {
    'crm.settings.tags.title': 'Gestione tag',
    'crm.settings.tags.desc': 'Crea, rinomina ed elimina i tag dei contatti in modo centralizzato.',
    'crm.settings.tags.count': '{count} tag',
    'crm.settings.tags.empty': 'Ancora nessun tag.',
    'crm.settings.tags.newTag': 'Nuovo tag',
    'crm.settings.tags.namePlaceholder': 'Nome del tag',
    'crm.settings.tags.addButton': 'Aggiungi tag',
    'crm.settings.tags.added': 'Tag «{name}» aggiunto',
    'crm.settings.tags.deleted': 'Tag «{name}» eliminato',
    'crm.settings.tags.deleteTitle': 'Eliminare il tag?',
    'crm.settings.tags.deleteDescription': 'Eliminare davvero il tag «{name}»? Verrà rimosso da tutti i contatti.',
  },
}
for (const [loc, keys] of Object.entries(add)) {
  const f = join(MSG, `${loc}.json`)
  const data = JSON.parse(readFileSync(f, 'utf8'))
  let n = 0
  for (const [k, v] of Object.entries(keys)) { if (!(k in data)) { data[k] = v; n++ } }
  writeFileSync(f, JSON.stringify(data, null, 2) + '\n', 'utf8')
  console.log(`${loc}: +${n}`)
}
