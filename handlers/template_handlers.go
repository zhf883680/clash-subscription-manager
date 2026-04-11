package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"clash-subscription-manager/models"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
)

type templatePayload struct {
	Name                    string   `json:"name"`
	Content                 string   `json:"content"`
	SelectedSubscriptionIDs []string `json:"selected_subscription_ids"`
	UseAllSubscriptions     *bool    `json:"use_all_subscriptions"`
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
			template.SelectedSubscriptionIDs = payload.SelectedSubscriptionIDs
			template.UseAllSubscriptions = payload.UseAllSubscriptions
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

	h.writeRenderedTemplate(w, r, item, templateRenderModeProviders)
}

func (h *Handler) RenderTemplateProxiesHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	item, err := GetTemplate(id, filepath.Join(h.config.DataDir, "templates.json"))
	if err != nil {
		h.respondJSON(w, http.StatusNotFound, Response{
			Success: false,
			Error:   fmt.Sprintf("Template not found: %v", err),
		})
		return
	}

	h.writeRenderedTemplate(w, r, item, templateRenderModeProxies)
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

	h.writeRenderedTemplate(w, r, item, templateRenderModeProviders)
}

func (h *Handler) RenderDefaultTemplateProxiesHandler(w http.ResponseWriter, r *http.Request) {
	item, err := GetDefaultTemplate(filepath.Join(h.config.DataDir, "templates.json"))
	if err != nil {
		h.respondJSON(w, http.StatusNotFound, Response{
			Success: false,
			Error:   fmt.Sprintf("Default template not found: %v", err),
		})
		return
	}

	h.writeRenderedTemplate(w, r, item, templateRenderModeProxies)
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
		Name:                    payload.Name,
		Content:                 payload.Content,
		SelectedSubscriptionIDs: payload.SelectedSubscriptionIDs,
		UseAllSubscriptions:     payload.UseAllSubscriptions,
		UpdatedAt:               time.Now(),
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

func (h *Handler) writeRenderedTemplate(w http.ResponseWriter, r *http.Request, item *models.Template, mode templateRenderMode) {
	rendered, err := h.renderTemplateContent(r, item, mode)
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

type templateRenderMode string

const (
	templateRenderModeProviders templateRenderMode = "providers"
	templateRenderModeProxies   templateRenderMode = "proxies"
)

func (h *Handler) renderTemplateContent(r *http.Request, item *models.Template, mode templateRenderMode) ([]byte, error) {
	subscriptions, err := ListSubscriptions(filepath.Join(h.config.DataDir, "subscriptions.json"))
	if err != nil {
		return nil, fmt.Errorf("load subscriptions: %w", err)
	}
	subscriptions = selectTemplateSubscriptions(item, subscriptions)

	var rendered string
	switch mode {
	case templateRenderModeProxies:
		proxiesBlock, err := h.renderExpandedProxiesBlock(subscriptions)
		if err != nil {
			return nil, err
		}
		rendered = replaceYAMLSection(item.Content, "proxy-providers", "")
		rendered = replaceYAMLSection(rendered, "proxies", proxiesBlock)
	default:
		rendered = replaceYAMLSection(item.Content, "proxy-providers", renderProxyProvidersBlock(h.buildTemplateProviders(r, subscriptions)))
	}
	return []byte(rendered), nil
}

func selectTemplateSubscriptions(item *models.Template, subscriptions []models.Subscription) []models.Subscription {
	if item == nil || item.UseAllSubscriptions == nil || *item.UseAllSubscriptions {
		return subscriptions
	}
	if len(item.SelectedSubscriptionIDs) == 0 {
		return []models.Subscription{}
	}

	selected := make(map[string]struct{}, len(item.SelectedSubscriptionIDs))
	for _, id := range item.SelectedSubscriptionIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		selected[id] = struct{}{}
	}

	filtered := make([]models.Subscription, 0, len(selected))
	for _, subscription := range subscriptions {
		if _, ok := selected[subscription.ID]; ok {
			filtered = append(filtered, subscription)
		}
	}
	return filtered
}

func (h *Handler) renderExpandedProxiesBlock(subscriptions []models.Subscription) (string, error) {
	proxies, err := h.collectExpandedProxyNodes(subscriptions)
	if err != nil {
		return "", err
	}
	return renderProxiesBlock(proxies)
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

func renderProxiesBlock(proxies []*yaml.Node) (string, error) {
	root := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "proxies"},
			{Kind: yaml.SequenceNode, Tag: "!!seq", Content: proxies},
		},
	}

	out, err := yaml.Marshal(root)
	if err != nil {
		return "", fmt.Errorf("marshal proxies: %w", err)
	}
	return decodeYAMLUnicodeEscapes(string(out)), nil
}

func replaceYAMLSection(raw string, key string, block string) string {
	lines := strings.Split(raw, "\n")
	start := -1
	end := len(lines)
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		keyPrefix := key + ":"
		if !isTopLevelSectionLine(line) {
			continue
		}
		if trimmed == keyPrefix || strings.HasPrefix(trimmed, keyPrefix+" ") {
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

	if start == -1 {
		if strings.TrimSpace(block) == "" {
			return strings.TrimRight(raw, "\n")
		}
		block = strings.TrimRight(block, "\n") + "\n"
		trimmed := strings.TrimRight(raw, "\n")
		if trimmed == "" {
			return strings.TrimRight(block, "\n")
		}
		return trimmed + "\n" + block
	}

	before := strings.Join(lines[:start], "\n")
	after := strings.Join(lines[end:], "\n")
	if strings.TrimSpace(block) == "" {
		if strings.TrimSpace(after) == "" {
			return strings.TrimRight(before, "\n")
		}
		if strings.TrimSpace(before) == "" {
			return strings.TrimLeft(after, "\n")
		}
		return strings.TrimRight(before, "\n") + "\n" + strings.TrimLeft(after, "\n")
	}
	block = strings.TrimRight(block, "\n") + "\n"
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

func isTopLevelSectionLine(line string) bool {
	return !startsWithIndent(line)
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
var yamlUnicodeEscapePattern = regexp.MustCompile(`\\U[0-9A-Fa-f]{8}|\\u[0-9A-Fa-f]{4}`)

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

func (h *Handler) collectExpandedProxyNodes(subscriptions []models.Subscription) ([]*yaml.Node, error) {
	var proxies []*yaml.Node
	for _, subscription := range subscriptions {
		if subscription.FilePath == "" {
			continue
		}

		var matcher *regexp.Regexp
		filter := strings.TrimSpace(subscription.Filter)
		if filter != "" {
			compiled, err := regexp.Compile(filter)
			if err != nil {
				return nil, fmt.Errorf("compile filter for subscription %q: %w", subscription.Name, err)
			}
			matcher = compiled
		}

		filePath := filepath.Join(h.config.DataDir, subscription.FilePath)
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("read subscription file for %q: %w", subscription.Name, err)
		}

		items, err := extractProxyNodes(data, matcher)
		if err != nil {
			return nil, fmt.Errorf("extract proxies for %q: %w", subscription.Name, err)
		}
		proxies = append(proxies, items...)
	}
	return proxies, nil
}

func extractProxyNodes(data []byte, matcher *regexp.Regexp) ([]*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse subscription yaml: %w", err)
	}

	root := doc.Content
	if len(root) == 0 || root[0] == nil || root[0].Kind != yaml.MappingNode {
		return nil, nil
	}

	proxiesNode := findMappingValue(root[0], "proxies")
	if proxiesNode == nil || proxiesNode.Kind != yaml.SequenceNode {
		return nil, nil
	}

	matches := make([]*yaml.Node, 0, len(proxiesNode.Content))
	for _, proxyNode := range proxiesNode.Content {
		if matcher != nil {
			name := findProxyName(proxyNode)
			if name == "" || !matcher.MatchString(name) {
				continue
			}
		}
		matches = append(matches, cloneYAMLNode(proxyNode))
	}
	return matches, nil
}

func findMappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for index := 0; index+1 < len(node.Content); index += 2 {
		if node.Content[index].Value == key {
			return node.Content[index+1]
		}
	}
	return nil
}

func findProxyName(node *yaml.Node) string {
	valueNode := findMappingValue(node, "name")
	if valueNode == nil {
		return ""
	}
	return strings.TrimSpace(valueNode.Value)
}

func cloneYAMLNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	cloned := *node
	if len(node.Content) > 0 {
		cloned.Content = make([]*yaml.Node, len(node.Content))
		for index, child := range node.Content {
			cloned.Content[index] = cloneYAMLNode(child)
		}
	}
	return &cloned
}

func decodeYAMLUnicodeEscapes(value string) string {
	return yamlUnicodeEscapePattern.ReplaceAllStringFunc(value, func(match string) string {
		hex := match[2:]
		codePoint, err := strconv.ParseInt(hex, 16, 32)
		if err != nil {
			return match
		}
		r := rune(codePoint)
		if !utf8.ValidRune(r) {
			return match
		}
		return string(r)
	})
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

	selectedIDs := make([]string, 0, len(payload.SelectedSubscriptionIDs))
	seen := make(map[string]struct{}, len(payload.SelectedSubscriptionIDs))
	for _, id := range payload.SelectedSubscriptionIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		selectedIDs = append(selectedIDs, id)
	}
	payload.SelectedSubscriptionIDs = selectedIDs

	if payload.UseAllSubscriptions == nil {
		useAll := len(payload.SelectedSubscriptionIDs) == 0
		payload.UseAllSubscriptions = &useAll
	}

	return payload, nil
}
