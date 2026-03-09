# URRACA

URRACA es una base inicial de desarrollo para una TUI en Go orientada a **API hunting**, **detección de OAuth/OIDC sin autenticación** y **recorrido automático de pipeline dirigido por hallazgos**.

## Estado actual

Este scaffold incluye:

- nombre final: **URRACA**
- splash ASCII grueso en negrita
- interfaz TUI con layout:
  - **2 paneles arriba**
  - **1 panel abajo**
- color principal **celeste**
- hallazgos destacados en **rojo**
- pipeline automático interno
- scheduler básico con cola de jobs
- stages iniciales:
  - bootstrap
  - hunt
  - swagger
  - auth
  - js
  - map

## Estructura

```text
urraca/
├── cmd/urraca/main.go
├── internal/app/app.go
├── internal/engine/
├── internal/pipeline/
├── internal/ui/
└── go.mod
```

## Ejecutar

```bash
cd urraca
# asegurarse de tener Go instalado (Kali: `sudo apt install golang` o similar)
go mod tidy
# las importaciones usan el módulo completo definido en go.mod
# por ejemplo: github.com/yourusername/urraca

go run ./cmd/urraca --target https://example.com
```
## Notas

- Esta versión es un **scaffold funcional de UI + engine**, no el hunter final.
- Los hallazgos actuales están simulados por heurística interna para fijar UX, layout y propagación entre stages.
- El siguiente paso natural es reemplazar cada stage por lógica HTTP real, timeouts, clasificación de respuestas y parsing de artefactos OpenAPI/JS.
