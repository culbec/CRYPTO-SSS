#!/bin/sh

set -eu

cd /app

CONFIG_DEFAULT="configs/config.json"
CONFIG_LOCAL="configs/config.local.json"

if [ ! -f "$CONFIG_LOCAL" ]; then
    if [ -f "$CONFIG_DEFAULT" ]; then
        cp "$CONFIG_DEFAULT" "$CONFIG_LOCAL"
    else
        echo '{}' > "$CONFIG_LOCAL"
    fi
fi

update_json_string() {
    key="$1"
    val="$2"
    esc="$(printf '%s' "$val" | sed -e 's/[\\\\/&|]/\\\\&/g')"
    sed -i -E "s|\\\"${key}\\\"[[:space:]]*:[[:space:]]*\\\"[^\\\"]*\\\"|\\\"${key}\\\": \\\"${esc}\\\"|g" "$CONFIG_LOCAL"
}

# Ensure server binds to all interfaces inside Docker
update_json_string "server_host" "${SERVER_HOST:-0.0.0.0}"
if [ -n "${SERVER_PORT:-}" ]; then update_json_string "server_port" "$SERVER_PORT"; fi
if [ -n "${JWT_SECRET_KEY:-}" ]; then update_json_string "jwt_secret_key" "$JWT_SECRET_KEY"; fi
if [ -n "${DB_URI:-}" ]; then update_json_string "db_uri" "$DB_URI"; fi
if [ -n "${DB_NAME:-}" ]; then update_json_string "db_name" "$DB_NAME"; fi

exec /app/backend