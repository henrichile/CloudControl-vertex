package container

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// StackDefinition describes a supported project template.
type StackDefinition struct {
	Name            string
	Description     string
	Services        []ServiceDef
	EnvVars         []EnvVar
	// MainService is the name of the HTTP-facing service that receives
	// Traefik labels when the project is created with a Domain.
	MainService string
	// MainServicePort is the internal container port Traefik proxies to.
	MainServicePort string
	// AdditionalFiles maps relative paths to file content templates.
	// These are written alongside docker-compose.yml when creating a project.
	AdditionalFiles map[string]string
}

type ServiceDef struct {
	Name        string
	Image       string
	Ports       []string
	Environment []string
	Volumes     []string
	DependsOn   []string
	Restart     string
	Command     string
	Labels      []string
	Networks    []string
}

type EnvVar struct {
	Key         string
	DefaultValue string
	Description string
}

// ProjectParams holds user-supplied values for template rendering.
type ProjectParams struct {
	ProjectName string
	Stack       string
	Domain      string
	DBName      string
	DBUser      string
	DBPassword  string
	AppPort     string
	ExtraEnv    map[string]string
}

var stacks = map[string]StackDefinition{
	"django-postgres": {
		Name:            "django-postgres",
		Description:     "Django 5 + PostgreSQL + Redis + Celery",
		MainService:     "web",
		MainServicePort: "8000",
		Services: []ServiceDef{
			{Name: "postgres", Image: "postgres:16-alpine", Environment: []string{"POSTGRES_DB={{.DBName}}", "POSTGRES_USER={{.DBUser}}", "POSTGRES_PASSWORD={{.DBPassword}}"}, Volumes: []string{"pg_data:/var/lib/postgresql/data"}, Restart: "unless-stopped"},
			{Name: "redis", Image: "redis:7-alpine", Restart: "unless-stopped"},
			{Name: "web", Image: "python:3.12-slim", Ports: []string{"{{.AppPort}}:8000"}, Environment: []string{"DATABASE_URL=postgresql://{{.DBUser}}:{{.DBPassword}}@postgres/{{.DBName}}", "REDIS_URL=redis://redis:6379/0", "DEBUG=1", "SECRET_KEY=dev-secret-change-in-production"}, DependsOn: []string{"postgres", "redis"}, Command: "bash -c 'pip install -r requirements.txt && python manage.py migrate && python manage.py runserver 0.0.0.0:8000'", Volumes: []string{"./:/app"}, Restart: "unless-stopped"},
			{Name: "celery", Image: "python:3.12-slim", Environment: []string{"DATABASE_URL=postgresql://{{.DBUser}}:{{.DBPassword}}@postgres/{{.DBName}}", "REDIS_URL=redis://redis:6379/0"}, DependsOn: []string{"redis", "postgres"}, Command: "celery -A core worker --loglevel=info", Volumes: []string{"./:/app"}, Restart: "unless-stopped"},
		},
		EnvVars: []EnvVar{
			{Key: "DB_NAME", DefaultValue: "django", Description: "Nombre de la base de datos"},
			{Key: "DB_USER", DefaultValue: "django", Description: "Usuario PostgreSQL"},
			{Key: "DB_PASSWORD", DefaultValue: "secret", Description: "Contraseña PostgreSQL"},
			{Key: "APP_PORT", DefaultValue: "8000", Description: "Puerto Django"},
		},
	},
	"spring-boot": {
		Name:            "spring-boot",
		Description:     "Spring Boot 3 + PostgreSQL + pgAdmin",
		MainService:     "app",
		MainServicePort: "8080",
		Services: []ServiceDef{
			{Name: "postgres", Image: "postgres:16-alpine", Environment: []string{"POSTGRES_DB={{.DBName}}", "POSTGRES_USER={{.DBUser}}", "POSTGRES_PASSWORD={{.DBPassword}}"}, Volumes: []string{"pg_data:/var/lib/postgresql/data"}, Restart: "unless-stopped"},
			{Name: "pgadmin", Image: "dpage/pgadmin4:8", Ports: []string{"5050:80"}, Environment: []string{"PGADMIN_DEFAULT_EMAIL=admin@admin.com", "PGADMIN_DEFAULT_PASSWORD={{.DBPassword}}"}, DependsOn: []string{"postgres"}, Restart: "unless-stopped"},
			{Name: "app", Image: "maven:3.9-eclipse-temurin-21", Ports: []string{"{{.AppPort}}:8080"}, Environment: []string{"SPRING_DATASOURCE_URL=jdbc:postgresql://postgres/{{.DBName}}", "SPRING_DATASOURCE_USERNAME={{.DBUser}}", "SPRING_DATASOURCE_PASSWORD={{.DBPassword}}", "SPRING_JPA_HIBERNATE_DDL_AUTO=update"}, DependsOn: []string{"postgres"}, Command: "mvn spring-boot:run", Volumes: []string{"./:/app", "maven_cache:/root/.m2"}, Restart: "unless-stopped"},
		},
		EnvVars: []EnvVar{
			{Key: "DB_NAME", DefaultValue: "springdb", Description: "Nombre de la base de datos"},
			{Key: "DB_USER", DefaultValue: "spring", Description: "Usuario PostgreSQL"},
			{Key: "DB_PASSWORD", DefaultValue: "secret", Description: "Contraseña PostgreSQL"},
			{Key: "APP_PORT", DefaultValue: "8080", Description: "Puerto Spring Boot"},
		},
	},
	"rails-postgres": {
		Name:            "rails-postgres",
		Description:     "Ruby on Rails 7 + PostgreSQL + Sidekiq + Redis",
		MainService:     "web",
		MainServicePort: "3000",
		Services: []ServiceDef{
			{Name: "postgres", Image: "postgres:16-alpine", Environment: []string{"POSTGRES_DB={{.DBName}}", "POSTGRES_USER={{.DBUser}}", "POSTGRES_PASSWORD={{.DBPassword}}"}, Volumes: []string{"pg_data:/var/lib/postgresql/data"}, Restart: "unless-stopped"},
			{Name: "redis", Image: "redis:7-alpine", Restart: "unless-stopped"},
			{Name: "web", Image: "ruby:3.3-slim", Ports: []string{"{{.AppPort}}:3000"}, Environment: []string{"DATABASE_URL=postgresql://{{.DBUser}}:{{.DBPassword}}@postgres/{{.DBName}}", "REDIS_URL=redis://redis:6379/0", "RAILS_ENV=development", "SECRET_KEY_BASE=dev-secret"}, DependsOn: []string{"postgres", "redis"}, Command: "bash -c 'bundle install && rails db:create db:migrate && rails server -b 0.0.0.0'", Volumes: []string{"./:/app", "bundle_cache:/usr/local/bundle"}, Restart: "unless-stopped"},
			{Name: "sidekiq", Image: "ruby:3.3-slim", Environment: []string{"DATABASE_URL=postgresql://{{.DBUser}}:{{.DBPassword}}@postgres/{{.DBName}}", "REDIS_URL=redis://redis:6379/0", "RAILS_ENV=development"}, DependsOn: []string{"web"}, Command: "bundle exec sidekiq", Volumes: []string{"./:/app", "bundle_cache:/usr/local/bundle"}, Restart: "unless-stopped"},
		},
		EnvVars: []EnvVar{
			{Key: "DB_NAME", DefaultValue: "rails_dev", Description: "Nombre de la base de datos"},
			{Key: "DB_USER", DefaultValue: "rails", Description: "Usuario PostgreSQL"},
			{Key: "DB_PASSWORD", DefaultValue: "secret", Description: "Contraseña PostgreSQL"},
			{Key: "APP_PORT", DefaultValue: "3000", Description: "Puerto Rails"},
		},
	},
	"wordpress": {
		Name:            "wordpress",
		Description:     "WordPress + FrankenPHP (worker mode) + MySQL 8 + Redis + phpMyAdmin",
		MainService:     "frankenphp",
		MainServicePort: "80",
		Services: []ServiceDef{
			{
				Name:  "mysql",
				Image: "mysql:8.3",
				Environment: []string{
					"MYSQL_ROOT_PASSWORD={{.DBPassword}}",
					"MYSQL_DATABASE={{.DBName}}",
					"MYSQL_USER={{.DBUser}}",
					"MYSQL_PASSWORD={{.DBPassword}}",
				},
				Command: "--default-authentication-plugin=caching_sha2_password --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci",
				Volumes: []string{
					"mysql_data:/var/lib/mysql",
					"./docker/mysql/performance.cnf:/etc/mysql/conf.d/performance.cnf:ro",
				},
				Restart: "unless-stopped",
			},
			{
				Name:    "redis",
				Image:   "redis:7-alpine",
				Command: "redis-server --maxmemory 256mb --maxmemory-policy allkeys-lru --save \"\" --appendonly no --tcp-backlog 511 --hz 20",
				Restart: "unless-stopped",
			},
			{
				Name:  "frankenphp",
				Image: "dunglas/frankenphp:php8.3-alpine",
				Ports: []string{"{{.AppPort}}:80", "443:443", "443:443/udp"},
				Environment: []string{
					"SERVER_NAME=:80",
					"WORDPRESS_DB_HOST=mysql",
					"WORDPRESS_DB_NAME={{.DBName}}",
					"WORDPRESS_DB_USER={{.DBUser}}",
					"WORDPRESS_DB_PASSWORD={{.DBPassword}}",
					"WORDPRESS_DB_CHARSET=utf8mb4",
					"WP_REDIS_HOST=redis",
					"WP_REDIS_PORT=6379",
					"WP_CACHE=true",
					"WP_DEBUG=false",
					"PHP_INI_SCAN_DIR=/usr/local/etc/php/conf.d",
				},
				DependsOn: []string{"mysql", "redis"},
				Volumes: []string{
					"wp_data:/app/public",
					"./docker/frankenphp/Caddyfile:/etc/caddy/Caddyfile:ro",
					"./docker/php/opcache.ini:/usr/local/etc/php/conf.d/opcache.ini:ro",
					"./docker/php/php.ini:/usr/local/etc/php/conf.d/custom.ini:ro",
				},
				Restart: "unless-stopped",
			},
			{
				Name:      "phpmyadmin",
				Image:     "phpmyadmin:5",
				Ports:     []string{"8081:80"},
				Environment: []string{
					"PMA_HOST=mysql",
					"PMA_USER={{.DBUser}}",
					"PMA_PASSWORD={{.DBPassword}}",
					"UPLOAD_LIMIT=128M",
				},
				DependsOn: []string{"mysql"},
				Restart:   "unless-stopped",
			},
		},
		EnvVars: []EnvVar{
			{Key: "DB_NAME", DefaultValue: "wordpress", Description: "Nombre de la base de datos MySQL"},
			{Key: "DB_USER", DefaultValue: "wordpress", Description: "Usuario MySQL"},
			{Key: "DB_PASSWORD", DefaultValue: "secret", Description: "Contraseña MySQL"},
			{Key: "APP_PORT", DefaultValue: "80", Description: "Puerto HTTP principal (FrankenPHP/Caddy)"},
		},
		AdditionalFiles: map[string]string{
			"docker/frankenphp/Caddyfile": `# FrankenPHP — Caddyfile optimizado para WordPress
# Generado por Cloud Control

{
	# Habilitar FrankenPHP
	frankenphp {
		# Worker mode: PHP se inicializa UNA sola vez y sirve N requests
		# sin overhead de bootstrap por petición (similar a Swoole/RoadRunner)
		worker {
			file    /app/public/index.php
			num     4           # ajustar según núcleos disponibles
			env     APP_ENV production
		}
	}
	order php_server before file_server

	# Desactivar telemetría de Caddy
	admin off
}

:80 {
	root * /app/public

	# Compresión en cascada: zstd → brotli → gzip
	encode {
		zstd
		br
		gzip 6
		minimum_length 1024
	}

	# Cabeceras de seguridad
	header {
		X-Content-Type-Options    "nosniff"
		X-Frame-Options           "SAMEORIGIN"
		X-XSS-Protection          "1; mode=block"
		Referrer-Policy           "strict-origin-when-cross-origin"
		Permissions-Policy        "geolocation=(), microphone=(), camera=()"
		-Server
	}

	# Caché larga para assets estáticos (WP añade ?ver= para cache busting)
	@staticAssets {
		path *.css *.js *.woff *.woff2 *.ttf *.eot *.otf
		path *.jpg *.jpeg *.png *.gif *.webp *.avif *.svg *.ico
		path *.mp4 *.webm *.pdf *.zip
	}
	header @staticAssets Cache-Control "public, max-age=31536000, immutable"

	# Uploads: no ejecutar PHP en /wp-content/uploads/
	@uploads {
		path /wp-content/uploads/*
	}
	route @uploads {
		file_server
	}

	# Bloquear acceso a archivos sensibles
	@sensitive {
		path /wp-config.php /wp-config-sample.php /.env /xmlrpc.php
		path /wp-includes/*.php /wp-admin/install.php
	}
	respond @sensitive 403

	# WordPress — FrankenPHP gestiona el rewrite automáticamente
	php_server {
		root /app/public
	}
}
`,
			"docker/mysql/performance.cnf": `# MySQL 8.3 — Configuración de alto rendimiento para WordPress
# Generado por Cloud Control
# Ajusta innodb_buffer_pool_size al 70-80% de tu RAM disponible

[mysqld]
# ── Juego de caracteres ──────────────────────────────────────────────────────
character-set-server  = utf8mb4
collation-server      = utf8mb4_unicode_ci
skip-character-set-client-handshake

# ── InnoDB — motor principal ─────────────────────────────────────────────────
innodb_buffer_pool_size        = 1G          # Caché principal de datos e índices
innodb_buffer_pool_instances   = 4           # 1 instancia por cada 1 GB
innodb_log_file_size           = 256M        # Logs de transacciones
innodb_log_buffer_size         = 64M
innodb_flush_log_at_trx_commit = 2          # 0=más rápido, 2=balance, 1=más seguro
innodb_flush_method            = O_DIRECT    # Evita doble caché con el SO
innodb_file_per_table          = 1
innodb_read_io_threads         = 8
innodb_write_io_threads        = 8
innodb_io_capacity             = 2000        # IOPS disponibles (SSD)
innodb_io_capacity_max         = 4000

# ── Conexiones ───────────────────────────────────────────────────────────────
max_connections        = 200
thread_cache_size      = 20
back_log               = 100

# ── Caché de consultas (desactivada en MySQL 8, usamos Redis) ────────────────
# query_cache_type = 0  -- ya es default en MySQL 8

# ── Buffers de consulta ──────────────────────────────────────────────────────
join_buffer_size       = 4M
sort_buffer_size       = 4M
read_buffer_size       = 2M
read_rnd_buffer_size   = 2M
tmp_table_size         = 128M
max_heap_table_size    = 128M

# ── Tablas ───────────────────────────────────────────────────────────────────
table_open_cache       = 4000
table_definition_cache = 2000
open_files_limit       = 65535

# ── Red ──────────────────────────────────────────────────────────────────────
max_allowed_packet     = 128M
net_read_timeout       = 30
net_write_timeout      = 60

# ── Slow query log ───────────────────────────────────────────────────────────
slow_query_log         = 1
long_query_time        = 1
log_queries_not_using_indexes = 0

[client]
default-character-set = utf8mb4

[mysql]
default-character-set = utf8mb4
`,
			"docker/php/opcache.ini": `; OPcache — Configuración de producción para FrankenPHP
; Generado por Cloud Control

[opcache]
opcache.enable                 = 1
opcache.enable_cli             = 0

; Memoria: mínimo 256M para WordPress con plugins
opcache.memory_consumption     = 256
opcache.interned_strings_buffer = 16
opcache.max_accelerated_files  = 20000
opcache.max_wasted_percentage  = 5

; En producción: no revalidar timestamps (recarga con kill -USR2 o reinicio del worker)
opcache.validate_timestamps    = 0
opcache.revalidate_freq        = 0

; JIT — habilitar en PHP 8+ (trashlines=1255 = todos los opcodes)
opcache.jit                    = 1255
opcache.jit_buffer_size        = 128M

; Optimizaciones adicionales
opcache.save_comments          = 1   ; necesario para doctrine/annotations
opcache.fast_shutdown          = 1
opcache.enable_file_override   = 0
opcache.huge_code_pages        = 1   ; usa huge pages del SO si están disponibles
opcache.preload_user           = www-data
`,
			"docker/php/php.ini": `; PHP — Configuración de producción para WordPress + FrankenPHP
; Generado por Cloud Control

; ── Memoria y límites ────────────────────────────────────────────────────────
memory_limit           = 512M
max_execution_time     = 60
max_input_time         = 60
max_input_vars         = 5000

; ── Uploads ──────────────────────────────────────────────────────────────────
upload_max_filesize    = 128M
post_max_size          = 128M
file_uploads           = On

; ── Seguridad ────────────────────────────────────────────────────────────────
expose_php             = Off
display_errors         = Off
log_errors             = On
error_log              = /var/log/php/error.log
error_reporting        = E_ALL & ~E_DEPRECATED & ~E_STRICT

; ── Sesiones ─────────────────────────────────────────────────────────────────
session.gc_maxlifetime = 1440
session.cookie_httponly = 1
session.cookie_samesite = Lax
session.use_strict_mode = 1

; ── Zona horaria ─────────────────────────────────────────────────────────────
date.timezone          = UTC

; ── Realpath cache (mejora resolución de includes) ───────────────────────────
realpath_cache_size    = 4096K
realpath_cache_ttl     = 600

; ── Misc ─────────────────────────────────────────────────────────────────────
default_charset        = UTF-8
short_open_tag         = Off
`,
		},
	},
	"rust-actix": {
		Name:            "rust-actix",
		Description:     "Rust Actix-Web 4 + PostgreSQL + Redis",
		MainService:     "api",
		MainServicePort: "8080",
		Services: []ServiceDef{
			{Name: "postgres", Image: "postgres:16-alpine", Ports: []string{"5432:5432"}, Environment: []string{"POSTGRES_DB={{.DBName}}", "POSTGRES_USER={{.DBUser}}", "POSTGRES_PASSWORD={{.DBPassword}}"}, Volumes: []string{"pg_data:/var/lib/postgresql/data"}, Restart: "unless-stopped"},
			{Name: "redis", Image: "redis:7-alpine", Ports: []string{"6379:6379"}, Restart: "unless-stopped"},
			{Name: "api", Image: "rust:1.79-slim", Ports: []string{"{{.AppPort}}:8080"}, Environment: []string{"DATABASE_URL=postgresql://{{.DBUser}}:{{.DBPassword}}@postgres/{{.DBName}}", "REDIS_URL=redis://redis:6379", "RUST_LOG=info"}, DependsOn: []string{"postgres", "redis"}, Command: "bash -c 'cargo build && cargo run'", Volumes: []string{"./:/app", "cargo_cache:/usr/local/cargo/registry", "target_cache:/app/target"}, Restart: "unless-stopped"},
		},
		EnvVars: []EnvVar{
			{Key: "DB_NAME", DefaultValue: "rustapp", Description: "Nombre de la base de datos"},
			{Key: "DB_USER", DefaultValue: "rustapp", Description: "Usuario PostgreSQL"},
			{Key: "DB_PASSWORD", DefaultValue: "secret", Description: "Contraseña PostgreSQL"},
			{Key: "APP_PORT", DefaultValue: "8080", Description: "Puerto Actix-Web"},
		},
	},
	"go-postgres": {
		Name:            "go-postgres",
		Description:     "Go (Gin/Chi) + PostgreSQL + Redis",
		MainService:     "api",
		MainServicePort: "8080",
		Services: []ServiceDef{
			{Name: "postgres", Image: "postgres:16-alpine", Ports: []string{"5432:5432"}, Environment: []string{"POSTGRES_DB={{.DBName}}", "POSTGRES_USER={{.DBUser}}", "POSTGRES_PASSWORD={{.DBPassword}}"}, Volumes: []string{"pg_data:/var/lib/postgresql/data"}, Restart: "unless-stopped"},
			{Name: "redis", Image: "redis:7-alpine", Ports: []string{"6379:6379"}, Restart: "unless-stopped"},
			{Name: "api", Image: "golang:1.22-alpine", Ports: []string{"{{.AppPort}}:8080"}, Environment: []string{"DATABASE_URL=postgresql://{{.DBUser}}:{{.DBPassword}}@postgres/{{.DBName}}?sslmode=disable", "REDIS_URL=redis://redis:6379", "PORT=8080"}, DependsOn: []string{"postgres", "redis"}, Command: "sh -c 'go mod download && go run ./cmd/server'", Volumes: []string{"./:/app", "go_cache:/go/pkg/mod"}, Restart: "unless-stopped"},
		},
		EnvVars: []EnvVar{
			{Key: "DB_NAME", DefaultValue: "goapp", Description: "Nombre de la base de datos"},
			{Key: "DB_USER", DefaultValue: "goapp", Description: "Usuario PostgreSQL"},
			{Key: "DB_PASSWORD", DefaultValue: "secret", Description: "Contraseña PostgreSQL"},
			{Key: "APP_PORT", DefaultValue: "8080", Description: "Puerto del servidor Go"},
		},
	},
	"elk": {
		Name:            "elk",
		Description:     "Elasticsearch + Logstash + Kibana (ELK Stack)",
		MainService:     "kibana",
		MainServicePort: "5601",
		Services: []ServiceDef{
			{Name: "elasticsearch", Image: "elasticsearch:8.14.3", Ports: []string{"9200:9200"}, Environment: []string{"discovery.type=single-node", "ES_JAVA_OPTS=-Xms512m -Xmx512m", "xpack.security.enabled=false"}, Volumes: []string{"es_data:/usr/share/elasticsearch/data"}, Restart: "unless-stopped"},
			{Name: "logstash", Image: "logstash:8.14.3", Ports: []string{"5044:5044", "5000:5000/tcp", "5000:5000/udp", "9600:9600"}, Environment: []string{"LS_JAVA_OPTS=-Xms256m -Xmx256m"}, DependsOn: []string{"elasticsearch"}, Volumes: []string{"./logstash/config:/usr/share/logstash/config", "./logstash/pipeline:/usr/share/logstash/pipeline"}, Restart: "unless-stopped"},
			{Name: "kibana", Image: "kibana:8.14.3", Ports: []string{"{{.AppPort}}:5601"}, Environment: []string{"ELASTICSEARCH_HOSTS=http://elasticsearch:9200"}, DependsOn: []string{"elasticsearch"}, Restart: "unless-stopped"},
		},
		EnvVars: []EnvVar{
			{Key: "APP_PORT", DefaultValue: "5601", Description: "Puerto Kibana"},
		},
	},
	"MERN": {
		Name:            "MERN",
		Description:     "MongoDB + Express + React + Node.js",
		MainService:     "frontend",
		MainServicePort: "5173",
		Services: []ServiceDef{
			{Name: "mongo", Image: "mongo:7", Ports: []string{"27017:27017"}, Volumes: []string{"mongo_data:/data/db"}, Restart: "unless-stopped"},
			{Name: "backend", Image: "node:20-alpine", Ports: []string{"{{.AppPort}}:3000"}, Environment: []string{"MONGO_URI=mongodb://mongo:27017/{{.DBName}}", "PORT=3000"}, DependsOn: []string{"mongo"}, Restart: "unless-stopped", Command: "sh -c 'npm install && npm run dev'", Volumes: []string{"./backend:/app", "/app/node_modules"}},
			{Name: "frontend", Image: "node:20-alpine", Ports: []string{"5173:5173"}, DependsOn: []string{"backend"}, Restart: "unless-stopped", Command: "sh -c 'npm install && npm run dev -- --host'", Volumes: []string{"./frontend:/app", "/app/node_modules"}},
		},
		EnvVars: []EnvVar{
			{Key: "DB_NAME", DefaultValue: "appdb", Description: "MongoDB database name"},
			{Key: "APP_PORT", DefaultValue: "3000", Description: "Backend API port"},
		},
	},
	"LAMP": {
		Name:            "LAMP",
		Description:     "Apache + MySQL + PHP 8",
		MainService:     "php",
		MainServicePort: "80",
		Services: []ServiceDef{
			{Name: "mysql", Image: "mysql:8.3", Ports: []string{"3306:3306"}, Environment: []string{"MYSQL_ROOT_PASSWORD={{.DBPassword}}", "MYSQL_DATABASE={{.DBName}}", "MYSQL_USER={{.DBUser}}", "MYSQL_PASSWORD={{.DBPassword}}"}, Volumes: []string{"mysql_data:/var/lib/mysql"}, Restart: "unless-stopped"},
			{Name: "php", Image: "php:8.3-apache", Ports: []string{"{{.AppPort}}:80"}, Environment: []string{"DB_HOST=mysql", "DB_NAME={{.DBName}}", "DB_USER={{.DBUser}}", "DB_PASS={{.DBPassword}}"}, DependsOn: []string{"mysql"}, Volumes: []string{"./src:/var/www/html"}, Restart: "unless-stopped"},
			{Name: "phpmyadmin", Image: "phpmyadmin:5", Ports: []string{"8081:80"}, Environment: []string{"PMA_HOST=mysql"}, DependsOn: []string{"mysql"}, Restart: "unless-stopped"},
		},
		EnvVars: []EnvVar{
			{Key: "DB_NAME", DefaultValue: "app", Description: "MySQL database name"},
			{Key: "DB_USER", DefaultValue: "appuser", Description: "MySQL user"},
			{Key: "DB_PASSWORD", DefaultValue: "secret", Description: "MySQL password"},
			{Key: "APP_PORT", DefaultValue: "80", Description: "Web server port"},
		},
	},
	"laravel-redis": {
		Name:            "laravel-redis",
		Description:     "Laravel + MySQL + Redis + Queue Worker",
		MainService:     "nginx",
		MainServicePort: "80",
		Services: []ServiceDef{
			{Name: "mysql", Image: "mysql:8.3", Environment: []string{"MYSQL_ROOT_PASSWORD={{.DBPassword}}", "MYSQL_DATABASE={{.DBName}}", "MYSQL_USER={{.DBUser}}", "MYSQL_PASSWORD={{.DBPassword}}"}, Volumes: []string{"mysql_data:/var/lib/mysql"}, Restart: "unless-stopped"},
			{Name: "redis", Image: "redis:7-alpine", Ports: []string{"6379:6379"}, Restart: "unless-stopped"},
			{Name: "app", Image: "php:8.3-fpm-alpine", Ports: []string{"{{.AppPort}}:9000"}, Environment: []string{"DB_CONNECTION=mysql", "DB_HOST=mysql", "DB_DATABASE={{.DBName}}", "DB_USERNAME={{.DBUser}}", "DB_PASSWORD={{.DBPassword}}", "REDIS_HOST=redis", "QUEUE_CONNECTION=redis"}, DependsOn: []string{"mysql", "redis"}, Volumes: []string{"./:/var/www/html"}, Restart: "unless-stopped"},
			{Name: "worker", Image: "php:8.3-fpm-alpine", Command: "php artisan queue:work --sleep=3 --tries=3", DependsOn: []string{"app"}, Volumes: []string{"./:/var/www/html"}, Restart: "unless-stopped"},
			{Name: "nginx", Image: "nginx:alpine", Ports: []string{"80:80"}, DependsOn: []string{"app"}, Volumes: []string{"./:/var/www/html", "./docker/nginx/default.conf:/etc/nginx/conf.d/default.conf"}, Restart: "unless-stopped"},
		},
		EnvVars: []EnvVar{
			{Key: "DB_NAME", DefaultValue: "laravel", Description: "Database name"},
			{Key: "DB_USER", DefaultValue: "laravel", Description: "Database user"},
			{Key: "DB_PASSWORD", DefaultValue: "secret", Description: "Database password"},
			{Key: "APP_PORT", DefaultValue: "9000", Description: "PHP-FPM port"},
		},
	},
	"fastapi-postgres": {
		Name:            "fastapi-postgres",
		Description:     "FastAPI + PostgreSQL + Alembic migrations",
		MainService:     "api",
		MainServicePort: "8000",
		Services: []ServiceDef{
			{Name: "postgres", Image: "postgres:16-alpine", Ports: []string{"5432:5432"}, Environment: []string{"POSTGRES_DB={{.DBName}}", "POSTGRES_USER={{.DBUser}}", "POSTGRES_PASSWORD={{.DBPassword}}"}, Volumes: []string{"pg_data:/var/lib/postgresql/data"}, Restart: "unless-stopped"},
			{Name: "api", Image: "python:3.12-slim", Ports: []string{"{{.AppPort}}:8000"}, Environment: []string{"DATABASE_URL=postgresql://{{.DBUser}}:{{.DBPassword}}@postgres/{{.DBName}}", "DEBUG=1"}, DependsOn: []string{"postgres"}, Command: "bash -c 'pip install -r requirements.txt && uvicorn main:app --host 0.0.0.0 --port 8000 --reload'", Volumes: []string{"./:/app"}, Restart: "unless-stopped"},
		},
		EnvVars: []EnvVar{
			{Key: "DB_NAME", DefaultValue: "fastapi", Description: "PostgreSQL database name"},
			{Key: "DB_USER", DefaultValue: "fastapi", Description: "PostgreSQL user"},
			{Key: "DB_PASSWORD", DefaultValue: "secret", Description: "PostgreSQL password"},
			{Key: "APP_PORT", DefaultValue: "8000", Description: "FastAPI port"},
		},
	},
	"data-science": {
		Name:            "data-science",
		Description:     "Jupyter Lab + Pandas + PostgreSQL + PgAdmin",
		MainService:     "jupyter",
		MainServicePort: "8888",
		Services: []ServiceDef{
			{Name: "jupyter", Image: "jupyter/datascience-notebook:latest", Ports: []string{"8888:8888"}, Environment: []string{"JUPYTER_ENABLE_LAB=yes", "GRANT_SUDO=yes"}, Volumes: []string{"./notebooks:/home/jovyan/work"}, Restart: "unless-stopped"},
			{Name: "postgres", Image: "postgres:16-alpine", Ports: []string{"5432:5432"}, Environment: []string{"POSTGRES_DB={{.DBName}}", "POSTGRES_USER={{.DBUser}}", "POSTGRES_PASSWORD={{.DBPassword}}"}, Volumes: []string{"pg_data:/var/lib/postgresql/data"}, Restart: "unless-stopped"},
			{Name: "pgadmin", Image: "dpage/pgadmin4:8", Ports: []string{"5050:80"}, Environment: []string{"PGADMIN_DEFAULT_EMAIL=admin@admin.com", "PGADMIN_DEFAULT_PASSWORD={{.DBPassword}}"}, DependsOn: []string{"postgres"}, Restart: "unless-stopped"},
		},
		EnvVars: []EnvVar{
			{Key: "DB_NAME", DefaultValue: "datasci", Description: "PostgreSQL database name"},
			{Key: "DB_USER", DefaultValue: "datasci", Description: "PostgreSQL user"},
			{Key: "DB_PASSWORD", DefaultValue: "secret", Description: "PostgreSQL password"},
		},
	},
	"nextjs-prisma": {
		Name:            "nextjs-prisma",
		Description:     "Next.js 14 + Prisma ORM + PostgreSQL",
		MainService:     "app",
		MainServicePort: "3000",
		Services: []ServiceDef{
			{Name: "postgres", Image: "postgres:16-alpine", Environment: []string{"POSTGRES_DB={{.DBName}}", "POSTGRES_USER={{.DBUser}}", "POSTGRES_PASSWORD={{.DBPassword}}"}, Volumes: []string{"pg_data:/var/lib/postgresql/data"}, Restart: "unless-stopped"},
			{Name: "app", Image: "node:20-alpine", Ports: []string{"{{.AppPort}}:3000"}, Environment: []string{"DATABASE_URL=postgresql://{{.DBUser}}:{{.DBPassword}}@postgres/{{.DBName}}", "NODE_ENV=development"}, DependsOn: []string{"postgres"}, Command: "sh -c 'npm install && npx prisma migrate deploy && npm run dev'", Volumes: []string{"./:/app", "/app/node_modules", "/app/.next"}, Restart: "unless-stopped"},
		},
		EnvVars: []EnvVar{
			{Key: "DB_NAME", DefaultValue: "nextapp", Description: "PostgreSQL database name"},
			{Key: "DB_USER", DefaultValue: "nextapp", Description: "PostgreSQL user"},
			{Key: "DB_PASSWORD", DefaultValue: "secret", Description: "PostgreSQL password"},
			{Key: "APP_PORT", DefaultValue: "3000", Description: "Next.js port"},
		},
	},
}

const composeTemplate = `# Auto-generated by Cloud Control — {{.ProjectName}}
# Stack: {{.Stack}}
# Generated at: {{ now }}

services:
{{- range .Services}}
  {{.Name}}:
    image: {{.Image}}
{{- if .Command}}
    command: {{.Command}}
{{- end}}
{{- if .Ports}}
    ports:
{{- range .Ports}}
      - "{{.}}"
{{- end}}
{{- end}}
{{- if .Environment}}
    environment:
{{- range .Environment}}
      - {{.}}
{{- end}}
{{- end}}
{{- if .Labels}}
    labels:
{{- range .Labels}}
      - "{{.}}"
{{- end}}
{{- end}}
{{- if .DependsOn}}
    depends_on:
{{- range .DependsOn}}
      - {{.}}
{{- end}}
{{- end}}
{{- if .Volumes}}
    volumes:
{{- range .Volumes}}
      - {{.}}
{{- end}}
{{- end}}
{{- if .Networks}}
    networks:
{{- range .Networks}}
      - {{.}}
{{- end}}
{{- end}}
{{- if .Restart}}
    restart: {{.Restart}}
{{- end}}
{{end}}
{{- if .NamedVolumes}}
volumes:
{{- range .NamedVolumes}}
  {{.}}:
{{- end}}
{{- end}}
{{- if .UseTraefik}}

networks:
  default:
  traefik-public:
    external: true
{{- end}}
`

// TemplateEngine generates Docker Compose files from stack templates.
type TemplateEngine struct{}

func NewTemplateEngine() *TemplateEngine {
	return &TemplateEngine{}
}

// ListStacks returns all available stack names and descriptions.
func (e *TemplateEngine) ListStacks() []map[string]string {
	result := make([]map[string]string, 0, len(stacks))
	for k, v := range stacks {
		result = append(result, map[string]string{
			"name":        k,
			"description": v.Description,
		})
	}
	return result
}

// GetStackEnvVars returns the required env vars for a stack.
func (e *TemplateEngine) GetStackEnvVars(stackName string) ([]EnvVar, error) {
	def, ok := stacks[stackName]
	if !ok {
		return nil, fmt.Errorf("unknown stack: %s", stackName)
	}
	return def.EnvVars, nil
}

// Generate renders a docker-compose.yml from a stack template and user params.
func (e *TemplateEngine) Generate(params ProjectParams) (string, error) {
	def, ok := stacks[params.Stack]
	if !ok {
		return "", fmt.Errorf("unknown stack: %s (available: %s)", params.Stack, strings.Join(availableStackNames(), ", "))
	}

	useTraefik := params.Domain != "" && def.MainService != ""

	// Inject Traefik labels into the main HTTP service when a domain is set.
	if useTraefik {
		def = injectTraefikLabels(def, params)
	}

	// Render each service field through params substitution.
	rendered, err := renderStack(def, params)
	if err != nil {
		return "", err
	}

	namedVolumes := collectNamedVolumes(rendered.Services)

	type templateData struct {
		ProjectParams
		Services     []ServiceDef
		NamedVolumes []string
		UseTraefik   bool
	}

	funcMap := template.FuncMap{
		"now":       func() string { return "{{ now }}" },
		"hasPrefix": strings.HasPrefix,
	}

	tmpl, err := template.New("compose").Funcs(funcMap).Parse(composeTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData{
		ProjectParams: params,
		Services:      rendered.Services,
		NamedVolumes:  namedVolumes,
		UseTraefik:    useTraefik,
	}); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// injectTraefikLabels adds Traefik routing labels to def.MainService and
// connects it to the traefik-public network.
func injectTraefikLabels(def StackDefinition, p ProjectParams) StackDefinition {
	routerName := sanitizeRouterName(p.ProjectName)
	port := def.MainServicePort
	if port == "" {
		port = "80"
	}

	labels := []string{
		"traefik.enable=true",
		// HTTPS router
		fmt.Sprintf("traefik.http.routers.%s.rule=Host(`%s`)", routerName, p.Domain),
		fmt.Sprintf("traefik.http.routers.%s.entrypoints=websecure", routerName),
		fmt.Sprintf("traefik.http.routers.%s.tls.certresolver=letsencrypt", routerName),
		fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port=%s", routerName, port),
		// HTTP → HTTPS redirect router
		fmt.Sprintf("traefik.http.routers.%s-http.rule=Host(`%s`)", routerName, p.Domain),
		fmt.Sprintf("traefik.http.routers.%s-http.entrypoints=web", routerName),
		fmt.Sprintf("traefik.http.routers.%s-http.middlewares=%s-https", routerName, routerName),
		fmt.Sprintf("traefik.http.middlewares.%s-https.redirectscheme.scheme=https", routerName),
		fmt.Sprintf("traefik.http.middlewares.%s-https.redirectscheme.permanent=true", routerName),
	}

	networks := []string{"default", "traefik-public"}

	result := def
	result.Services = make([]ServiceDef, len(def.Services))
	copy(result.Services, def.Services)

	for i, svc := range result.Services {
		if svc.Name == def.MainService {
			result.Services[i].Labels = append(svc.Labels, labels...)
			result.Services[i].Networks = append(svc.Networks, networks...)
		}
	}
	return result
}

// sanitizeRouterName converts a project name into a valid Traefik router name
// (alphanumeric and hyphens only).
func sanitizeRouterName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	return b.String()
}

// renderStack substitutes params into all service string fields.
func renderStack(def StackDefinition, params ProjectParams) (StackDefinition, error) {
	rendered := def
	rendered.Services = make([]ServiceDef, len(def.Services))

	for i, svc := range def.Services {
		rs, err := renderServiceDef(svc, params)
		if err != nil {
			return rendered, err
		}
		rendered.Services[i] = rs
	}
	return rendered, nil
}

func renderServiceDef(svc ServiceDef, params ProjectParams) (ServiceDef, error) {
	render := func(s string) (string, error) {
		t, err := template.New("").Parse(s)
		if err != nil {
			return "", err
		}
		var buf bytes.Buffer
		if err := t.Execute(&buf, params); err != nil {
			return "", err
		}
		return buf.String(), nil
	}

	renderSlice := func(ss []string) ([]string, error) {
		out := make([]string, len(ss))
		for i, s := range ss {
			v, err := render(s)
			if err != nil {
				return nil, err
			}
			out[i] = v
		}
		return out, nil
	}

	rs := svc
	var err error
	if rs.Environment, err = renderSlice(svc.Environment); err != nil {
		return rs, err
	}
	if rs.Command, err = render(svc.Command); err != nil {
		return rs, err
	}
	if rs.Ports, err = renderSlice(svc.Ports); err != nil {
		return rs, err
	}
	if rs.Labels, err = renderSlice(svc.Labels); err != nil {
		return rs, err
	}
	return rs, nil
}

func collectNamedVolumes(services []ServiceDef) []string {
	seen := map[string]bool{}
	var result []string
	for _, svc := range services {
		for _, v := range svc.Volumes {
			parts := strings.SplitN(v, ":", 2)
			name := parts[0]
			if !strings.HasPrefix(name, ".") && !strings.HasPrefix(name, "/") {
				if !seen[name] {
					seen[name] = true
					result = append(result, name)
				}
			}
		}
	}
	return result
}

// GenerateFiles returns the docker-compose.yml plus any additional config files
// defined by the stack (e.g. Caddyfile, my.cnf, opcache.ini).
// The returned map keys are relative paths; values are rendered file contents.
func (e *TemplateEngine) GenerateFiles(params ProjectParams) (map[string]string, error) {
	compose, err := e.Generate(params)
	if err != nil {
		return nil, err
	}

	files := map[string]string{
		"docker-compose.yml": compose,
	}

	def := stacks[params.Stack]
	for relPath, contentTmpl := range def.AdditionalFiles {
		t, err := template.New("").Parse(contentTmpl)
		if err != nil {
			return nil, fmt.Errorf("parsing additional file template %q: %w", relPath, err)
		}
		var buf bytes.Buffer
		if err := t.Execute(&buf, params); err != nil {
			return nil, fmt.Errorf("rendering additional file %q: %w", relPath, err)
		}
		files[relPath] = buf.String()
	}

	return files, nil
}

func availableStackNames() []string {
	names := make([]string, 0, len(stacks))
	for k := range stacks {
		names = append(names, k)
	}
	return names
}
