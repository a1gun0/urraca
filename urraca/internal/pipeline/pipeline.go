package pipeline

import (
	"fmt"
	"strings"
	"time"

	"github.com/yourusername/urraca/internal/engine"
)

type Results struct {
	Events   []engine.Event
	Findings []engine.Finding
	NextJobs []engine.Job
}

type StageFunc func(engine.Job, engine.Config) Results

type Definition struct {
	Stages map[string]StageFunc
}

func Default() Definition {
	return Definition{
		Stages: map[string]StageFunc{
			"bootstrap": stageBootstrap,
			"hunt":      stageHunt,
			"swagger":   stageSwagger,
			"auth":      stageAuth,
			"js":        stageJS,
			"map":       stageMap,
		},
	}
}

func stageBootstrap(job engine.Job, cfg engine.Config) Results {
	now := time.Now()
	target := cfg.Target

	return Results{
		Events: []engine.Event{
			{Kind: engine.EventLog, Message: "normalizando target y preparando scheduler", CreatedAt: now},
		},
		Findings: []engine.Finding{
			{
				ID:         "bootstrap-target",
				Target:     target,
				Module:     "bootstrap",
				Category:   "target",
				Subtype:    "normalized",
				Value:      target,
				URL:        target,
				Confidence: 100,
				Timestamp:  now,
				Severity:   "info",
				Evidence:   "target aceptado por el parser",
			},
		},
		NextJobs: []engine.Job{
			newJob("hunt", target, map[string]string{"target": target}, 90),
			newJob("auth", target, map[string]string{"target": target}, 80),
			newJob("js", target, map[string]string{"target": target}, 70),
		},
	}
}

func stageHunt(job engine.Job, cfg engine.Config) Results {
	now := time.Now()
	target := cfg.Target

	paths := []string{
		"/api",
		"/api/v1",
		"/swagger.json",
		"/openapi.json",
		"/docs",
		"/graphql",
		"/.well-known/openid-configuration",
	}

	findings := []engine.Finding{
		{
			ID:         "hunt-api-root",
			Target:     target,
			Module:     "hunt",
			Category:   "api",
			Subtype:    "rest-root",
			Value:      "/api",
			URL:        target + "/api",
			Status:     200,
			Confidence: 60,
			Timestamp:  now,
			Severity:   "medium",
			Evidence:   "ruta candidata del pipeline semilla",
		},
	}

	var next []engine.Job
	for _, p := range paths {
		if strings.Contains(p, "swagger") || strings.Contains(p, "openapi") {
			next = append(next, newJob("swagger", target, map[string]string{"url": target + p, "path": p}, 95))
		}
		if strings.Contains(p, "openid") {
			next = append(next, newJob("auth", target, map[string]string{"url": target + p, "path": p}, 100))
		}
	}

	return Results{
		Events: []engine.Event{
			{Kind: engine.EventLog, Message: "surface hunt inicial completo", CreatedAt: now},
		},
		Findings: findings,
		NextJobs: append(next, newJob("map", target, map[string]string{"target": target}, 40)),
	}
}

func stageSwagger(job engine.Job, cfg engine.Config) Results {
	now := time.Now()
	url := job.Input["url"]
	if url == "" {
		url = cfg.Target + "/swagger.json"
	}

	f := engine.Finding{
		ID:         "swagger-openapi",
		Target:     cfg.Target,
		Module:     "swagger",
		Category:   "doc",
		Subtype:    "openapi-json",
		Value:      url,
		URL:        url,
		Status:     200,
		Confidence: 85,
		Timestamp:  now,
		Severity:   "high",
		Evidence:   "artefacto OpenAPI candidato detectado por heurística",
	}

	return Results{
		Events: []engine.Event{
			{Kind: engine.EventLog, Message: "parseando artefacto OpenAPI candidato", CreatedAt: now},
		},
		Findings: []engine.Finding{f},
		NextJobs: []engine.Job{
			newJob("auth", cfg.Target, map[string]string{"url": url, "source": "openapi"}, 90),
			newJob("map", cfg.Target, map[string]string{"url": url, "source": "openapi"}, 60),
		},
	}
}

func stageAuth(job engine.Job, cfg engine.Config) Results {
	now := time.Now()
	url := job.Input["url"]
	if url == "" {
		url = cfg.Target + "/.well-known/openid-configuration"
	}

	findings := []engine.Finding{
		{
			ID:         "auth-bearer",
			Target:     cfg.Target,
			Module:     "auth",
			Category:   "auth",
			Subtype:    "bearer-challenge",
			Value:      "WWW-Authenticate: Bearer",
			URL:        cfg.Target + "/api",
			Status:     401,
			Confidence: 75,
			Timestamp:  now,
			Severity:   "high",
			Evidence:   "challenge observable sin autenticación",
		},
		{
			ID:         "auth-oidc",
			Target:     cfg.Target,
			Module:     "auth",
			Category:   "auth",
			Subtype:    "oidc-metadata",
			Value:      url,
			URL:        url,
			Status:     200,
			Confidence: 95,
			Timestamp:  now,
			Severity:   "critical",
			Evidence:   "endpoint well-known candidato para OIDC",
		},
	}

	return Results{
		Events: []engine.Event{
			{Kind: engine.EventLog, Message: "correlando señales de OAuth/OIDC", CreatedAt: now},
		},
		Findings: findings,
		NextJobs: []engine.Job{
			newJob("map", cfg.Target, map[string]string{"url": url, "source": "auth"}, 80),
		},
	}
}

func stageJS(job engine.Job, cfg engine.Config) Results {
	now := time.Now()

	findings := []engine.Finding{
		{
			ID:         "js-client-id",
			Target:     cfg.Target,
			Module:     "js",
			Category:   "auth",
			Subtype:    "client-id-public",
			Value:      "client_id público referenciado en JS",
			URL:        cfg.Target + "/assets/app.js",
			Status:     200,
			Confidence: 55,
			Timestamp:  now,
			Severity:   "medium",
			Evidence:   "string coincidente con patrón client_id",
		},
	}

	return Results{
		Events: []engine.Event{
			{Kind: engine.EventLog, Message: "analizando JavaScript público", CreatedAt: now},
		},
		Findings: findings,
		NextJobs: []engine.Job{
			newJob("auth", cfg.Target, map[string]string{"url": cfg.Target + "/assets/app.js", "source": "js"}, 65),
			newJob("map", cfg.Target, map[string]string{"url": cfg.Target + "/assets/app.js", "source": "js"}, 50),
		},
	}
}

func stageMap(job engine.Job, cfg engine.Config) Results {
	now := time.Now()
	src := job.Input["source"]
	if src == "" {
		src = "pipeline"
	}
	return Results{
		Events: []engine.Event{
			{Kind: engine.EventLog, Message: fmt.Sprintf("actualizando grafo desde %s", src), CreatedAt: now},
		},
		Findings: nil,
		NextJobs: nil,
	}
}

func newJob(stage string, target string, input map[string]string, priority int) engine.Job {
	if input == nil {
		input = make(map[string]string)
	}
	return engine.Job{
		ID:        fmt.Sprintf("%s-%d", stage, time.Now().UnixNano()),
		Stage:     stage,
		Target:    target,
		Priority:  priority,
		CreatedAt: time.Now(),
		Timeout:   8 * time.Second, // TODO: use cfg or constant
		Input:     input,
	}
}
