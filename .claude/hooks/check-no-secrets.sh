#!/bin/bash
# Hook: verhindert das Stagen von Dateien, die wahrscheinlich Secrets enthalten.
# Laeuft als PreToolUse auf dem Bash-Tool, wenn `git add` erkannt wird.
# Exit 2 = blockieren, Exit 0 = erlauben.
#
# KANONISCH (Source of Truth: claude-command-center). Robuste Variante:
# iteriert ueber die echten Pfad-Argumente + Suffix-Allowlist (statt blindem
# Substring-Match ueber das ganze Kommando).

INPUT=$(cat)
# Capture endet an der Grenze des `git add`-Subkommandos: Shell-Trenner (&& ; |) und das
# Ende des JSON-Strings (" bzw. \) beenden den Match. Ohne diese Grenze zieht `.*` bei
# `git add X && git commit -m "... .env.production ..."` die komplette Commit-Message mit
# hinein, die unten wortweise als Pfad geprueft wird — und der Commit wird faelschlich
# blockiert, obwohl gar keine .env-Datei gestaget ist.
COMMAND=$(echo "$INPUT" | grep -oP 'git add\b[^&;|"\\]*' || true)

if [ -z "$COMMAND" ]; then
  exit 0
fi

BLOCKED_PATTERNS=(".env" "credentials.json" ".pem" ".key" "secrets.json" "service-account.json")
ALLOWED_SUFFIXES=(".example" ".template" ".sample" ".dist")

# "git add" abstreifen und ueber die echten Pfad-Argumente iterieren.
PATHS=$(echo "$COMMAND" | sed -E 's/^git add //')

for PATH_ARG in $PATHS; do
  # Flags und Leeres ueberspringen.
  case "$PATH_ARG" in
    -*|"") continue ;;
  esac

  # Beispiel-/Template-/Sample-/Dist-Dateien sind erlaubt.
  IS_ALLOWED=false
  for SUFFIX in "${ALLOWED_SUFFIXES[@]}"; do
    case "$PATH_ARG" in
      *"$SUFFIX") IS_ALLOWED=true; break ;;
    esac
  done
  [ "$IS_ALLOWED" = true ] && continue

  for PATTERN in "${BLOCKED_PATTERNS[@]}"; do
    case "$PATH_ARG" in
      *"$PATTERN"*)
        echo "BLOCKED: Versuch, eine Datei mit sensiblem Muster zu stagen: $PATTERN" >&2
        echo "Pfad: $PATH_ARG" >&2
        echo "Secrets/Credentials gehoeren nie ins Repo — nutze Umgebungsvariablen (siehe CLAUDE.md)." >&2
        echo "Hinweis: Dateien auf .example, .template, .sample, .dist sind ausgenommen." >&2
        exit 2
        ;;
    esac
  done
done

exit 0
