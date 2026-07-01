# ERD — модель данных

ER-диаграмма панели. Источник истины по схеме — миграции в
`backend/internal/migrations`. Диаграмма отражает `SPEC.md §4`.

```mermaid
erDiagram
    admins ||--o{ audit_logs : "performs"
    servers ||--o{ inbounds : "hosts"
    clients ||--o{ client_inbounds : "granted"
    inbounds ||--o{ client_inbounds : "grants"

    admins {
        int id PK
        string username
        string password_hash
        string totp_secret_enc
        bool totp_enabled
        datetime created_at
        datetime updated_at
    }
    servers {
        int id PK
        string name
        string host
        int ssh_port
        string ssh_user
        string ssh_auth_method
        string ssh_secret_enc
        string ssh_passphrase_enc
        string xray_config_path
        string xray_service_name
        string ip
        string geo_country
        string geo_city
        string geo_asn
        string status
        datetime last_check_at
        datetime last_sync_at
        string last_sync_error
        datetime created_at
        datetime updated_at
    }
    inbounds {
        int id PK
        int server_id FK
        string tag
        string protocol
        string listen
        int port
        json settings_json
        json stream_settings_json
        json sniffing_json
        string remark
        bool enabled
        datetime created_at
        datetime updated_at
    }
    clients {
        int id PK
        string name
        string uuid
        string password
        string subscription_token
        bool enabled
        string remark
        datetime created_at
        datetime updated_at
    }
    client_inbounds {
        int id PK
        int client_id FK
        int inbound_id FK
        bool enabled
        datetime created_at
    }
    audit_logs {
        int id PK
        int admin_id FK
        string action
        string target_type
        int target_id
        json detail_json
        string ip
        string user_agent
        datetime created_at
    }
    settings {
        string key PK
        string value
    }
```

## Инварианты

- `client_inbounds` уникален по `(client_id, inbound_id)`.
- `inbounds.protocol` — строка из реестра протоколов; протокол-специфичные
  параметры в `*_json`, ALTER TABLE при добавлении протокола не требуется.
- `servers.ssh_secret_enc`, `servers.ssh_passphrase_enc`,
  `admins.totp_secret_enc` — AES-256-GCM (base64), не хранятся в открытом виде.
- `admins.password_hash` — bcrypt (cost 12).
- `clients.uuid` покрывает UUID-протоколы (VLESS/VMess), `clients.password` —
  парольные (Trojan/Shadowsocks). Оба генерируются при создании клиента.
