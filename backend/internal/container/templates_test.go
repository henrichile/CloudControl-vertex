package container_test

import (
	"strings"
	"testing"

	"github.com/etasoft/cloudcontrol/internal/container"
)

func TestTemplateEngine_ListStacks(t *testing.T) {
	engine := container.NewTemplateEngine()
	stacks := engine.ListStacks()

	if len(stacks) == 0 {
		t.Fatal("ListStacks() returned empty list")
	}

	expectedStacks := []string{"MERN", "LAMP", "laravel-redis", "fastapi-postgres", "data-science", "nextjs-prisma", "django-postgres", "spring-boot", "rails-postgres", "wordpress", "rust-actix", "go-postgres", "elk"}
	stackNames := make(map[string]bool)
	for _, s := range stacks {
		stackNames[s["name"]] = true
	}
	for _, expected := range expectedStacks {
		if !stackNames[expected] {
			t.Errorf("expected stack %q not found in ListStacks()", expected)
		}
	}
}

func TestTemplateEngine_Generate_MERN(t *testing.T) {
	engine := container.NewTemplateEngine()
	params := container.ProjectParams{
		ProjectName: "mi-app",
		Stack:       "MERN",
		DBName:      "testdb",
		AppPort:     "4000",
	}

	result, err := engine.Generate(params)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	checks := []string{
		"mongo:7",
		"node:20-alpine",
		"4000:3000",
		"mi-app",
		"MERN",
		"mongo_data",
		"services:",
		"volumes:",
	}
	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("Generate() output missing %q\nGot:\n%s", check, result)
		}
	}
}

func TestTemplateEngine_Generate_AllStacks(t *testing.T) {
	engine := container.NewTemplateEngine()
	stacks := engine.ListStacks()

	for _, stack := range stacks {
		name := stack["name"]
		t.Run(name, func(t *testing.T) {
			params := container.ProjectParams{
				ProjectName: "test-project",
				Stack:       name,
				DBName:      "testdb",
				DBUser:      "testuser",
				DBPassword:  "testpass",
				AppPort:     "8080",
			}
			result, err := engine.Generate(params)
			if err != nil {
				t.Fatalf("Generate(%q) error: %v", name, err)
			}
			if !strings.Contains(result, "services:") {
				t.Errorf("Generate(%q) missing 'services:' section", name)
			}
			if len(result) < 100 {
				t.Errorf("Generate(%q) output too short: %d chars", name, len(result))
			}
		})
	}
}

func TestTemplateEngine_Generate_UnknownStack(t *testing.T) {
	engine := container.NewTemplateEngine()
	_, err := engine.Generate(container.ProjectParams{
		ProjectName: "test",
		Stack:       "stack-que-no-existe",
	})
	if err == nil {
		t.Fatal("Generate() with unknown stack should return error")
	}
	if !strings.Contains(err.Error(), "unknown stack") {
		t.Errorf("expected 'unknown stack' in error, got: %v", err)
	}
}

func TestTemplateEngine_Generate_ParamSubstitution(t *testing.T) {
	engine := container.NewTemplateEngine()
	params := container.ProjectParams{
		ProjectName: "prueba",
		Stack:       "fastapi-postgres",
		DBName:      "mibasededatos",
		DBUser:      "miusuario",
		DBPassword:  "mipassword",
		AppPort:     "9999",
	}

	result, err := engine.Generate(params)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	for _, token := range []string{"{{.DBName}}", "{{.DBUser}}", "{{.DBPassword}}", "{{.AppPort}}"} {
		if strings.Contains(result, token) {
			t.Errorf("unresolved template token %q in output", token)
		}
	}

	if !strings.Contains(result, "mibasededatos") {
		t.Error("DBName substitution failed")
	}
	if !strings.Contains(result, "9999:8000") {
		t.Error("AppPort substitution failed")
	}
}

func TestTemplateEngine_GetStackEnvVars(t *testing.T) {
	engine := container.NewTemplateEngine()

	envVars, err := engine.GetStackEnvVars("MERN")
	if err != nil {
		t.Fatalf("GetStackEnvVars() error: %v", err)
	}
	if len(envVars) == 0 {
		t.Error("GetStackEnvVars() returned empty list for MERN")
	}

	_, err = engine.GetStackEnvVars("no-existe")
	if err == nil {
		t.Error("GetStackEnvVars() should error for unknown stack")
	}
}

func TestTemplateEngine_WordPress_FrankenPHP(t *testing.T) {
	engine := container.NewTemplateEngine()
	params := container.ProjectParams{
		ProjectName: "mi-blog",
		Stack:       "wordpress",
		DBName:      "wpdb",
		DBUser:      "wpuser",
		DBPassword:  "wpsecret",
		AppPort:     "8080",
	}

	result, err := engine.Generate(params)
	if err != nil {
		t.Fatalf("Generate(wordpress) error: %v", err)
	}

	checks := []string{
		"dunglas/frankenphp",
		"redis:7-alpine",
		"mysql:8.3",
		"8080:80",
		"443:443",
		"443:443/udp",
		"WP_CACHE=true",
		"WP_REDIS_HOST=redis",
		"allkeys-lru",
		"performance.cnf",
		"Caddyfile",
		"opcache.ini",
	}
	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("wordpress compose missing %q", check)
		}
	}

	// No debe aparecer nginx ni php-fpm standalone
	forbidden := []string{"nginx:alpine", "php-fpm", "wordpress:6-php8.3-fpm"}
	for _, f := range forbidden {
		if strings.Contains(result, f) {
			t.Errorf("wordpress compose should NOT contain %q", f)
		}
	}
}

func TestTemplateEngine_GenerateFiles_WordPress(t *testing.T) {
	engine := container.NewTemplateEngine()
	params := container.ProjectParams{
		ProjectName: "mi-blog",
		Stack:       "wordpress",
		DBName:      "wpdb",
		DBUser:      "wpuser",
		DBPassword:  "wpsecret",
		AppPort:     "80",
	}

	files, err := engine.GenerateFiles(params)
	if err != nil {
		t.Fatalf("GenerateFiles(wordpress) error: %v", err)
	}

	// Debe incluir docker-compose.yml más los archivos adicionales
	expectedFiles := []string{
		"docker-compose.yml",
		"docker/frankenphp/Caddyfile",
		"docker/mysql/performance.cnf",
		"docker/php/opcache.ini",
		"docker/php/php.ini",
	}
	for _, f := range expectedFiles {
		if _, ok := files[f]; !ok {
			t.Errorf("GenerateFiles() missing file %q", f)
		}
	}

	// Caddyfile debe tener configuración de worker mode
	caddyfile := files["docker/frankenphp/Caddyfile"]
	caddyChecks := []string{"frankenphp", "worker", "encode", "zstd", "br", "gzip", "php_server", "X-Content-Type-Options"}
	for _, c := range caddyChecks {
		if !strings.Contains(caddyfile, c) {
			t.Errorf("Caddyfile missing %q", c)
		}
	}

	// opcache.ini debe tener JIT habilitado
	opcache := files["docker/php/opcache.ini"]
	if !strings.Contains(opcache, "opcache.jit") {
		t.Error("opcache.ini missing JIT configuration")
	}
	if !strings.Contains(opcache, "validate_timestamps") || strings.Contains(opcache, "validate_timestamps    = 1") {
		t.Error("opcache.ini should disable validate_timestamps in production")
	}

	// MySQL config debe tener parámetros InnoDB
	myCnf := files["docker/mysql/performance.cnf"]
	mysqlChecks := []string{"innodb_buffer_pool_size", "innodb_flush_method", "utf8mb4", "O_DIRECT"}
	for _, c := range mysqlChecks {
		if !strings.Contains(myCnf, c) {
			t.Errorf("performance.cnf missing %q", c)
		}
	}
}

func TestTemplateEngine_GenerateFiles_NoAdditionalFiles(t *testing.T) {
	engine := container.NewTemplateEngine()
	params := container.ProjectParams{
		ProjectName: "mi-api",
		Stack:       "MERN",
		DBName:      "testdb",
		AppPort:     "3000",
	}

	files, err := engine.GenerateFiles(params)
	if err != nil {
		t.Fatalf("GenerateFiles(MERN) error: %v", err)
	}
	// MERN no tiene archivos adicionales
	if len(files) != 1 {
		t.Errorf("MERN should only generate docker-compose.yml, got %d files: %v", len(files), files)
	}
	if _, ok := files["docker-compose.yml"]; !ok {
		t.Error("docker-compose.yml missing from output")
	}
}

func TestTemplateEngine_TraefikLabels_Injected(t *testing.T) {
	engine := container.NewTemplateEngine()
	params := container.ProjectParams{
		ProjectName: "mi-api",
		Stack:       "fastapi-postgres",
		DBName:      "apidb",
		DBUser:      "apiuser",
		DBPassword:  "secret",
		AppPort:     "8000",
		Domain:      "api.ejemplo.com",
	}

	result, err := engine.Generate(params)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	traefikChecks := []string{
		"traefik.enable=true",
		"Host(`api.ejemplo.com`)",
		"entrypoints=websecure",
		"certresolver=letsencrypt",
		"loadbalancer.server.port=8000",
		"entrypoints=web",
		"redirectscheme.scheme=https",
		"traefik-public",
	}
	for _, check := range traefikChecks {
		if !strings.Contains(result, check) {
			t.Errorf("Traefik labels: missing %q\nGot:\n%s", check, result)
		}
	}

	// Must declare the external network
	if !strings.Contains(result, "external: true") {
		t.Error("compose missing 'external: true' for traefik-public network")
	}
}

func TestTemplateEngine_TraefikLabels_NotInjectedWithoutDomain(t *testing.T) {
	engine := container.NewTemplateEngine()
	result, err := engine.Generate(container.ProjectParams{
		ProjectName: "sin-dominio",
		Stack:       "fastapi-postgres",
		DBName:      "db",
		DBUser:      "u",
		DBPassword:  "p",
		AppPort:     "8000",
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if strings.Contains(result, "traefik.enable") {
		t.Error("Traefik labels must NOT appear when no domain is set")
	}
	if strings.Contains(result, "traefik-public") {
		t.Error("traefik-public network must NOT appear when no domain is set")
	}
}

func TestTemplateEngine_TraefikLabels_AllStacksHaveMainService(t *testing.T) {
	engine := container.NewTemplateEngine()
	stacks := engine.ListStacks()

	for _, stack := range stacks {
		name := stack["name"]
		t.Run(name, func(t *testing.T) {
			params := container.ProjectParams{
				ProjectName: "test-dominio",
				Stack:       name,
				DBName:      "db",
				DBUser:      "user",
				DBPassword:  "pass",
				AppPort:     "8080",
				Domain:      "test.ejemplo.com",
			}
			result, err := engine.Generate(params)
			if err != nil {
				t.Fatalf("Generate(%q) error: %v", name, err)
			}
			if !strings.Contains(result, "traefik.enable=true") {
				t.Errorf("stack %q: Traefik labels missing when Domain is set", name)
			}
			if !strings.Contains(result, "traefik-public") {
				t.Errorf("stack %q: traefik-public network missing when Domain is set", name)
			}
		})
	}
}

func TestTemplateEngine_NamedVolumes(t *testing.T) {
	engine := container.NewTemplateEngine()

	for _, stack := range []string{"fastapi-postgres", "LAMP", "data-science"} {
		result, err := engine.Generate(container.ProjectParams{
			ProjectName: "test",
			Stack:       stack,
			DBName:      "db",
			DBUser:      "user",
			DBPassword:  "pass",
			AppPort:     "8080",
		})
		if err != nil {
			t.Fatalf("Generate(%q) error: %v", stack, err)
		}
		if !strings.Contains(result, "volumes:") {
			t.Errorf("stack %q should declare named volumes section", stack)
		}
	}
}
