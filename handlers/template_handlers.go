package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"clash-subscription-manager/models"

	"github.com/gorilla/mux"
)

type templatePayload struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

func (h *Handler) ListTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	templates, err := ListTemplates(filepath.Join(h.config.DataDir, "templates.json"))
	if err != nil {
		h.respondJSON(w, http.StatusInternalServerError, Response{
			Success: false,
			Error:   fmt.Sprintf("Failed to load templates: %v", err),
		})
		return
	}

	h.respondJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    templates,
	})
}

func (h *Handler) TemplatesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.createTemplate(w, r)
	default:
		h.respondJSON(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "Method not allowed",
		})
	}
}

func (h *Handler) TemplateHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	dataFile := filepath.Join(h.config.DataDir, "templates.json")

	switch r.Method {
	case http.MethodGet:
		item, err := GetTemplate(id, dataFile)
		if err != nil {
			h.respondJSON(w, http.StatusNotFound, Response{
				Success: false,
				Error:   fmt.Sprintf("Template not found: %v", err),
			})
			return
		}
		h.respondJSON(w, http.StatusOK, Response{Success: true, Data: item})
	case http.MethodPut:
		payload, err := decodeTemplatePayload(r)
		if err != nil {
			h.respondJSON(w, http.StatusBadRequest, Response{
				Success: false,
				Error:   fmt.Sprintf("Invalid request body: %v", err),
			})
			return
		}
		item, err := UpdateTemplate(id, dataFile, func(template *models.Template) error {
			template.Name = payload.Name
			template.Content = payload.Content
			return nil
		})
		if err != nil {
			h.respondJSON(w, http.StatusNotFound, Response{
				Success: false,
				Error:   fmt.Sprintf("Failed to update template: %v", err),
			})
			return
		}
		h.respondJSON(w, http.StatusOK, Response{Success: true, Message: "Template updated successfully", Data: item})
	case http.MethodDelete:
		if err := DeleteTemplate(id, dataFile); err != nil {
			h.respondJSON(w, http.StatusNotFound, Response{
				Success: false,
				Error:   fmt.Sprintf("Failed to delete template: %v", err),
			})
			return
		}
		h.respondJSON(w, http.StatusOK, Response{Success: true, Message: "Template deleted successfully"})
	default:
		h.respondJSON(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "Method not allowed",
		})
	}
}

func (h *Handler) SetDefaultTemplateHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	item, err := SetDefaultTemplate(id, filepath.Join(h.config.DataDir, "templates.json"))
	if err != nil {
		h.respondJSON(w, http.StatusNotFound, Response{
			Success: false,
			Error:   fmt.Sprintf("Failed to set default template: %v", err),
		})
		return
	}

	h.respondJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "Template set as default",
		Data:    item,
	})
}

func (h *Handler) RenderTemplateHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	item, err := GetTemplate(id, filepath.Join(h.config.DataDir, "templates.json"))
	if err != nil {
		h.respondJSON(w, http.StatusNotFound, Response{
			Success: false,
			Error:   fmt.Sprintf("Template not found: %v", err),
		})
		return
	}

	h.writeRenderedTemplate(w, r, item)
}

func (h *Handler) RenderDefaultTemplateHandler(w http.ResponseWriter, r *http.Request) {
	item, err := GetDefaultTemplate(filepath.Join(h.config.DataDir, "templates.json"))
	if err != nil {
		h.respondJSON(w, http.StatusNotFound, Response{
			Success: false,
			Error:   fmt.Sprintf("Default template not found: %v", err),
		})
		return
	}

	h.writeRenderedTemplate(w, r, item)
}

func (h *Handler) createTemplate(w http.ResponseWriter, r *http.Request) {
	payload, err := decodeTemplatePayload(r)
	if err != nil {
		h.respondJSON(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	item, err := AddTemplate(models.Template{
		Name:      payload.Name,
		Content:   payload.Content,
		UpdatedAt: time.Now(),
	}, filepath.Join(h.config.DataDir, "templates.json"))
	if err != nil {
		h.respondJSON(w, http.StatusInternalServerError, Response{
			Success: false,
			Error:   fmt.Sprintf("Failed to add template: %v", err),
		})
		return
	}

	h.respondJSON(w, http.StatusCreated, Response{
		Success: true,
		Message: "Template added successfully",
		Data:    item,
	})
}

func (h *Handler) writeRenderedTemplate(w http.ResponseWriter, r *http.Request, item *models.Template) {
	rendered, err := h.renderTemplateContent(r, item.Content)
	if err != nil {
		h.respondJSON(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   fmt.Sprintf("Failed to render template: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.yaml\"", sanitizeFilename(item.Name)))
	_, _ = w.Write(rendered)
}

func (h *Handler) renderTemplateContent(r *http.Request, raw string) ([]byte, error) {
	subscriptions, err := ListSubscriptions(filepath.Join(h.config.DataDir, "subscriptions.json"))
	if err != nil {
		return nil, fmt.Errorf("load subscriptions: %w", err)
	}

	rendered := replaceProxyProvidersSection(raw, renderProxyProvidersBlock(h.buildTemplateProviders(r, subscriptions)))
	return []byte(rendered), nil
}

func (h *Handler) buildTemplateProviders(r *http.Request, subscriptions []models.Subscription) []templateProvider {
	providers := make([]templateProvider, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		if subscription.ID == "" {
			continue
		}
		fileName := subscription.FilePath
		if fileName == "" {
			fileName = sanitizeFilename(subscription.Name) + ".yaml"
		}
		providers = append(providers, templateProvider{
			Name:             subscription.Name,
			URL:              absoluteDownloadURL(r, subscription.ID),
			Path:             "./proxies/" + path.Base(fileName),
			Filter:           strings.TrimSpace(subscription.Filter),
			AdditionalPrefix: sanitizeProviderLabel(subscription.Name) + " |",
		})
	}
	return providers
}

type templateProvider struct {
	Name             string
	URL              string
	Path             string
	Filter           string
	AdditionalPrefix string
}

func renderProxyProvidersBlock(providers []templateProvider) string {
	if len(providers) == 0 {
		return "proxy-providers: {}\n"
	}

	sort.SliceStable(providers, func(i, j int) bool {
		return providers[i].Name < providers[j].Name
	})

	var builder strings.Builder
	builder.WriteString("proxy-providers:\n")
	for _, provider := range providers {
		builder.WriteString("  ")
		builder.WriteString(renderYAMLKey(provider.Name))
		builder.WriteString(":\n")
		builder.WriteString("    type: http\n")
		builder.WriteString("    url: ")
		builder.WriteString(renderPlainScalar(provider.URL))
		builder.WriteString("\n")
		builder.WriteString("    path: ")
		builder.WriteString(renderPlainScalar(provider.Path))
		builder.WriteString("\n")
		if provider.Filter != "" {
			builder.WriteString("    filter: ")
			builder.WriteString(renderQuotedScalar(provider.Filter))
			builder.WriteString("\n")
		}
		builder.WriteString("    interval: 86400\n")
		builder.WriteString("    health-check:\n")
		builder.WriteString("      enable: true\n")
		builder.WriteString("      url: https://cp.cloudflare.com\n")
		builder.WriteString("      interval: 600\n")
		builder.WriteString("    override:\n")
		builder.WriteString("      ip-version: ipv4\n")
		builder.WriteString("      additional-prefix: ")
		builder.WriteString(renderQuotedScalar(provider.AdditionalPrefix))
		builder.WriteString("\n")
	}
	return builder.String()
}

func replaceProxyProvidersSection(raw string, block string) string {
	lines := strings.Split(raw, "\n")
	start := -1
	end := len(lines)
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "proxy-providers:" || strings.HasPrefix(trimmed, "proxy-providers: ") {
			start = index
			for next := index + 1; next < len(lines); next++ {
				nextTrimmed := strings.TrimSpace(lines[next])
				if nextTrimmed == "" {
					continue
				}
				if !startsWithIndent(lines[next]) {
					end = next
					break
				}
			}
			break
		}
	}

	block = strings.TrimRight(block, "\n") + "\n"
	if start == -1 {
		trimmed := strings.TrimRight(raw, "\n")
		if trimmed == "" {
			return strings.TrimRight(block, "\n")
		}
		return trimmed + "\n" + block
	}

	before := strings.Join(lines[:start], "\n")
	after := strings.Join(lines[end:], "\n")
	if strings.TrimSpace(after) == "" {
		return strings.TrimRight(before, "\n") + "\n" + block
	}
	if strings.TrimSpace(before) == "" {
		return block + strings.TrimLeft(after, "\n")
	}
	return strings.TrimRight(before, "\n") + "\n" + block + strings.TrimLeft(after, "\n")
}

func startsWithIndent(line string) bool {
	return strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")
}

func renderYAMLKey(value string) string {
	if yamlPlainSafePattern.MatchString(value) {
		return value
	}
	return renderQuotedScalar(value)
}

func renderPlainScalar(value string) string {
	if yamlPlainSafePattern.MatchString(value) {
		return value
	}
	return renderQuotedScalar(value)
}

func renderQuotedScalar(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return `"` + replacer.Replace(value) + `"`
}

var yamlPlainSafePattern = regexp.MustCompile(`^[\p{L}\p{N}_./:@%+\-| ]+$`)

func absoluteDownloadURL(r *http.Request, id string) string {
	return fmt.Sprintf("%s://%s/download/%s", requestScheme(r), r.Host, id)
}

func requestScheme(r *http.Request) string {
	if forwardedProto := r.Header.Get("X-Forwarded-Proto"); forwardedProto != "" {
		return forwardedProto
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func sanitizeProviderLabel(name string) string {
	fields := strings.Fields(strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == ' ' || r == '-' || r == '_':
			return ' '
		default:
			return -1
		}
	}, name))
	if len(fields) == 0 {
		return "provider"
	}
	return strings.Join(fields, " ")
}

func decodeTemplatePayload(r *http.Request) (templatePayload, error) {
	var payload templatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return templatePayload{}, err
	}

	payload.Name = strings.TrimSpace(payload.Name)
	payload.Content = strings.TrimSpace(payload.Content)
	if payload.Name == "" {
		return templatePayload{}, fmt.Errorf("name is required")
	}
	if payload.Content == "" {
		return templatePayload{}, fmt.Errorf("content is required")
	}
	if !strings.Contains(payload.Content, "proxy-providers:") {
		payload.Content = strings.TrimRight(payload.Content, "\n") + "\n\nproxy-providers:\n"
	}
	return payload, nil
}
