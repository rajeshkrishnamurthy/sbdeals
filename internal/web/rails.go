package web

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/rajeshkrishnamurthy/sbdeals/internal/books"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/bundles"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/rails"
)

var railValidationOrder = []string{"title", "type"}

func (s *Server) handleRailsCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.renderRailsList(w, r)
	case http.MethodPost:
		s.createRail(w, r)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) handleRailNew(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	s.renderRailForm(w, railFormViewModel{
		PageTitle:     "Add Rail",
		Action:        "/admin/rails",
		SubmitLabel:   "Save Rail",
		ActiveSection: "rails",
		Input:         railFormInput{},
		TypeOptions:   []rails.RailType{rails.RailTypeBook, rails.RailTypeBundle},
		Errors:        map[string]string{},
		IsCreate:      true,
	})
}

func (s *Server) handleRailItem(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/admin/rails/" {
		http.Redirect(w, r, "/admin/rails", http.StatusMovedPermanently)
		return
	}

	railID, action, ok := parseRailPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	if !s.dispatchRailItemAction(w, r, railID, action) {
		http.NotFound(w, r)
	}
}

func (s *Server) dispatchRailItemAction(w http.ResponseWriter, r *http.Request, railID int, action string) bool {
	switch action {
	case "":
		s.handleRailDetailAction(w, r, railID)
		return true
	case "publish":
		s.handleRailPublishAction(w, r, railID)
		return true
	case "unpublish":
		s.handleRailUnpublishAction(w, r, railID)
		return true
	case "move-up":
		s.handleRailMoveUpAction(w, r, railID)
		return true
	case "move-down":
		s.handleRailMoveDownAction(w, r, railID)
		return true
	case "items/add":
		s.handleRailAddItemAction(w, r, railID)
		return true
	case "items/remove":
		s.handleRailRemoveItemAction(w, r, railID)
		return true
	}
	return false
}

func (s *Server) handleRailDetailAction(w http.ResponseWriter, r *http.Request, railID int) {
	if !methodAllowed(r.Method, http.MethodGet, http.MethodPost) {
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		return
	}
	if r.Method == http.MethodGet {
		s.renderRailDetail(w, r, railID)
		return
	}
	s.updateRail(w, r, railID)
}

func (s *Server) handleRailPublishAction(w http.ResponseWriter, r *http.Request, railID int) {
	if !methodAllowed(r.Method, http.MethodPost, http.MethodPatch) {
		writeMethodNotAllowed(w, http.MethodPost, http.MethodPatch)
		return
	}
	s.publishRail(w, r, railID)
}

func (s *Server) handleRailUnpublishAction(w http.ResponseWriter, r *http.Request, railID int) {
	if !methodAllowed(r.Method, http.MethodPost, http.MethodPatch) {
		writeMethodNotAllowed(w, http.MethodPost, http.MethodPatch)
		return
	}
	s.unpublishRail(w, r, railID)
}

func (s *Server) handleRailMoveUpAction(w http.ResponseWriter, r *http.Request, railID int) {
	if !methodAllowed(r.Method, http.MethodPost) {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}
	s.moveRailUp(w, r, railID)
}

func (s *Server) handleRailMoveDownAction(w http.ResponseWriter, r *http.Request, railID int) {
	if !methodAllowed(r.Method, http.MethodPost) {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}
	s.moveRailDown(w, r, railID)
}

func (s *Server) handleRailAddItemAction(w http.ResponseWriter, r *http.Request, railID int) {
	if !methodAllowed(r.Method, http.MethodPost) {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}
	s.addRailItem(w, r, railID)
}

func (s *Server) handleRailRemoveItemAction(w http.ResponseWriter, r *http.Request, railID int) {
	if !methodAllowed(r.Method, http.MethodPost) {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}
	s.removeRailItem(w, r, railID)
}

func methodAllowed(method string, allowed ...string) bool {
	for _, allow := range allowed {
		if method == allow {
			return true
		}
	}
	return false
}

func (s *Server) renderRailsList(w http.ResponseWriter, r *http.Request) {
	items, err := s.railStore.List()
	if err != nil {
		http.Error(w, "failed to load rails", http.StatusInternalServerError)
		return
	}

	view := railsListViewModel{
		Flash:           r.URL.Query().Get("flash"),
		ValidationToast: strings.TrimSpace(r.URL.Query().Get("error")),
		ActiveSection:   "rails",
		Rails:           items,
	}
	if err := railsListTemplate.Execute(w, view); err != nil {
		http.Error(w, "failed to render rails list", http.StatusInternalServerError)
	}
}

func (s *Server) createRail(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}
	input := readRailCreateInput(r)
	errorsByField := validateRailCreateInput(input)
	if len(errorsByField) > 0 {
		w.WriteHeader(http.StatusBadRequest)
		s.renderRailForm(w, railFormViewModel{
			PageTitle:       "Add Rail",
			Action:          "/admin/rails",
			SubmitLabel:     "Save Rail",
			ActiveSection:   "rails",
			Input:           input,
			TypeOptions:     []rails.RailType{rails.RailTypeBook, rails.RailTypeBundle},
			Errors:          errorsByField,
			ValidationToast: buildValidationToast(errorsByField, railValidationOrder),
			IsCreate:        true,
		})
		return
	}

	_, err := s.railStore.Create(rails.CreateInput{
		Title: input.Title,
		Type:  rails.RailType(input.Type),
	})
	if errors.Is(err, rails.ErrDuplicateTitle) {
		errorsByField := map[string]string{"title": "Rail title must be unique."}
		w.WriteHeader(http.StatusBadRequest)
		s.renderRailForm(w, railFormViewModel{
			PageTitle:       "Add Rail",
			Action:          "/admin/rails",
			SubmitLabel:     "Save Rail",
			ActiveSection:   "rails",
			Input:           input,
			TypeOptions:     []rails.RailType{rails.RailTypeBook, rails.RailTypeBundle},
			Errors:          errorsByField,
			ValidationToast: buildValidationToast(errorsByField, railValidationOrder),
			IsCreate:        true,
		})
		return
	}
	if err != nil {
		http.Error(w, "failed to create rail", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/rails?flash="+url.QueryEscape("Rail created successfully."), http.StatusSeeOther)
}

func (s *Server) renderRailDetail(w http.ResponseWriter, r *http.Request, railID int) {
	railData, err := s.railStore.Get(railID)
	if errors.Is(err, rails.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "failed to load rail", http.StatusInternalServerError)
		return
	}

	availableItems, selectedItems, err := s.buildRailItemsForDetail(railData)
	if err != nil {
		http.Error(w, "failed to load rail items", http.StatusInternalServerError)
		return
	}

	s.renderRailForm(w, railFormViewModel{
		PageTitle:         "View/Edit Rail",
		Action:            fmt.Sprintf("/admin/rails/%d", railData.ID),
		SubmitLabel:       "Save Changes",
		ActiveSection:     "rails",
		Flash:             r.URL.Query().Get("flash"),
		ValidationToast:   strings.TrimSpace(r.URL.Query().Get("error")),
		Input:             railFormInput{Title: railData.Title, Type: string(railData.Type)},
		ImmutableType:     railTypeLabel(railData.Type),
		Errors:            map[string]string{},
		RailID:            railData.ID,
		ShowPublishToggle: true,
		PublishAction:     fmt.Sprintf("/admin/rails/%d/%s?from=edit", railData.ID, toggleActionPath(railData.IsPublished)),
		PublishLabel:      publishStateLabel(railData.IsPublished),
		PublishRecency:    publishRecencyLabel(railData.IsPublished, railData.PublishedAt, railData.UnpublishedAt),
		ShowItemPanel:     true,
		AvailableItems:    availableItems,
		SelectedItems:     selectedItems,
	})
}

func (s *Server) updateRail(w http.ResponseWriter, r *http.Request, railID int) {
	railData, err := s.railStore.Get(railID)
	if errors.Is(err, rails.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "failed to load rail", http.StatusInternalServerError)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.Form.Get("title"))
	errorsByField := map[string]string{}
	if title == "" {
		errorsByField["title"] = "Rail title is required."
	}
	if len(errorsByField) > 0 {
		s.renderRailUpdateValidation(w, railData, title, errorsByField)
		return
	}

	_, err = s.railStore.Update(railID, rails.UpdateInput{Title: title})
	if errors.Is(err, rails.ErrDuplicateTitle) {
		errorsByField["title"] = "Rail title must be unique."
		s.renderRailUpdateValidation(w, railData, title, errorsByField)
		return
	}
	if err != nil {
		if errors.Is(err, rails.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "failed to update rail", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/rails/%d?flash=%s", railID, url.QueryEscape("Rail updated successfully.")), http.StatusSeeOther)
}

func (s *Server) renderRailUpdateValidation(w http.ResponseWriter, railData rails.Rail, title string, errorsByField map[string]string) {
	availableItems, selectedItems, err := s.buildRailItemsForDetail(railData)
	if err != nil {
		http.Error(w, "failed to load rail items", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusBadRequest)
	s.renderRailForm(w, railFormViewModel{
		PageTitle:         "View/Edit Rail",
		Action:            fmt.Sprintf("/admin/rails/%d", railData.ID),
		SubmitLabel:       "Save Changes",
		ActiveSection:     "rails",
		Input:             railFormInput{Title: title, Type: string(railData.Type)},
		ImmutableType:     railTypeLabel(railData.Type),
		Errors:            errorsByField,
		ValidationToast:   buildValidationToast(errorsByField, railValidationOrder),
		RailID:            railData.ID,
		ShowPublishToggle: true,
		PublishAction:     fmt.Sprintf("/admin/rails/%d/%s?from=edit", railData.ID, toggleActionPath(railData.IsPublished)),
		PublishLabel:      publishStateLabel(railData.IsPublished),
		PublishRecency:    publishRecencyLabel(railData.IsPublished, railData.PublishedAt, railData.UnpublishedAt),
		ShowItemPanel:     true,
		AvailableItems:    availableItems,
		SelectedItems:     selectedItems,
	})
}

func (s *Server) publishRail(w http.ResponseWriter, r *http.Request, railID int) {
	if _, err := s.railStore.Publish(railID); err != nil {
		if errors.Is(err, rails.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "failed to publish rail", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, railPublishRedirectPath(r, railID, "Rail published successfully.", ""), http.StatusSeeOther)
}

func (s *Server) unpublishRail(w http.ResponseWriter, r *http.Request, railID int) {
	if _, err := s.railStore.Unpublish(railID); err != nil {
		if errors.Is(err, rails.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "failed to unpublish rail", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, railPublishRedirectPath(r, railID, "Rail unpublished successfully.", ""), http.StatusSeeOther)
}

func (s *Server) moveRailUp(w http.ResponseWriter, r *http.Request, railID int) {
	if err := s.railStore.MoveUp(railID); err != nil {
		if errors.Is(err, rails.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "failed to move rail up", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/rails?flash="+url.QueryEscape("Rail order updated successfully."), http.StatusSeeOther)
}

func (s *Server) moveRailDown(w http.ResponseWriter, r *http.Request, railID int) {
	if err := s.railStore.MoveDown(railID); err != nil {
		if errors.Is(err, rails.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "failed to move rail down", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/rails?flash="+url.QueryEscape("Rail order updated successfully."), http.StatusSeeOther)
}

func (s *Server) addRailItem(w http.ResponseWriter, r *http.Request, railID int) {
	railData, err := s.railStore.Get(railID)
	if errors.Is(err, rails.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "failed to load rail", http.StatusInternalServerError)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, fmt.Sprintf("/admin/rails/%d?error=%s", railID, url.QueryEscape("Please choose a valid item.")), http.StatusSeeOther)
		return
	}

	itemID, ok := parsePositiveID(r.Form.Get("item_id"))
	if !ok {
		http.Redirect(w, r, fmt.Sprintf("/admin/rails/%d?error=%s", railID, url.QueryEscape("Please choose a valid item.")), http.StatusSeeOther)
		return
	}

	valid, typeMismatch, err := s.ensureRailItemMatchesType(railData.Type, itemID)
	if err != nil {
		http.Error(w, "failed to validate item", http.StatusInternalServerError)
		return
	}
	if typeMismatch {
		http.Redirect(w, r, fmt.Sprintf("/admin/rails/%d?error=%s", railID, url.QueryEscape("Type mismatch: item does not match rail type.")), http.StatusSeeOther)
		return
	}
	if !valid {
		http.Redirect(w, r, fmt.Sprintf("/admin/rails/%d?error=%s", railID, url.QueryEscape("Please choose a valid item.")), http.StatusSeeOther)
		return
	}

	if _, err := s.railStore.AddItem(railID, itemID); err != nil {
		if errors.Is(err, rails.ErrDuplicateItem) {
			http.Redirect(w, r, fmt.Sprintf("/admin/rails/%d?error=%s", railID, url.QueryEscape("Item is already added to this rail.")), http.StatusSeeOther)
			return
		}
		if errors.Is(err, rails.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "failed to add item to rail", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/rails/%d?flash=%s", railID, url.QueryEscape("Item added to rail successfully.")), http.StatusSeeOther)
}

func (s *Server) removeRailItem(w http.ResponseWriter, r *http.Request, railID int) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, fmt.Sprintf("/admin/rails/%d?error=%s", railID, url.QueryEscape("Please choose a valid item.")), http.StatusSeeOther)
		return
	}
	itemID, ok := parsePositiveID(r.Form.Get("item_id"))
	if !ok {
		http.Redirect(w, r, fmt.Sprintf("/admin/rails/%d?error=%s", railID, url.QueryEscape("Please choose a valid item.")), http.StatusSeeOther)
		return
	}

	if _, err := s.railStore.RemoveItem(railID, itemID); err != nil {
		if errors.Is(err, rails.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "failed to remove item from rail", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/admin/rails/%d?flash=%s", railID, url.QueryEscape("Item removed from rail successfully.")), http.StatusSeeOther)
}

func (s *Server) ensureRailItemMatchesType(railType rails.RailType, itemID int) (bool, bool, error) {
	switch railType {
	case rails.RailTypeBook:
		_, err := s.bookStore.Get(itemID)
		if err == nil {
			return true, false, nil
		}
		if !errors.Is(err, books.ErrNotFound) {
			return false, false, err
		}
		_, bundleErr := s.bundleStore.Get(itemID)
		if bundleErr == nil {
			return false, true, nil
		}
		if errors.Is(bundleErr, bundles.ErrNotFound) {
			return false, false, nil
		}
		return false, false, bundleErr
	case rails.RailTypeBundle:
		_, err := s.bundleStore.Get(itemID)
		if err == nil {
			return true, false, nil
		}
		if !errors.Is(err, bundles.ErrNotFound) {
			return false, false, err
		}
		_, bookErr := s.bookStore.Get(itemID)
		if bookErr == nil {
			return false, true, nil
		}
		if errors.Is(bookErr, books.ErrNotFound) {
			return false, false, nil
		}
		return false, false, bookErr
	default:
		return false, false, fmt.Errorf("unsupported rail type: %s", railType)
	}
}

func railPublishRedirectPath(r *http.Request, railID int, flash string, errorMessage string) string {
	base := "/admin/rails"
	if r.URL.Query().Get("from") == "edit" {
		base = fmt.Sprintf("/admin/rails/%d", railID)
	}
	if flash != "" {
		return base + "?flash=" + url.QueryEscape(flash)
	}
	if errorMessage != "" {
		return base + "?error=" + url.QueryEscape(errorMessage)
	}
	return base
}

func readRailCreateInput(r *http.Request) railFormInput {
	return railFormInput{
		Title: strings.TrimSpace(r.Form.Get("title")),
		Type:  strings.TrimSpace(r.Form.Get("type")),
	}
}

func validateRailCreateInput(input railFormInput) map[string]string {
	errs := map[string]string{}
	if input.Title == "" {
		errs["title"] = "Rail title is required."
	}
	if !rails.RailType(input.Type).IsValid() {
		errs["type"] = "Rail type is required."
	}
	return errs
}

func parsePositiveID(raw string) (int, bool) {
	id, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func parseRailPath(path string) (int, string, bool) {
	prefix := "/admin/rails/"
	if !strings.HasPrefix(path, prefix) {
		return 0, "", false
	}
	rest := strings.TrimPrefix(path, prefix)
	if rest == "" {
		return 0, "", false
	}
	parts := strings.Split(rest, "/")
	if len(parts) == 0 || len(parts) > 3 {
		return 0, "", false
	}

	id, err := strconv.Atoi(parts[0])
	if err != nil || id <= 0 {
		return 0, "", false
	}

	action, ok := railActionFromParts(parts[1:])
	if !ok {
		return 0, "", false
	}
	return id, action, true
}

func railActionFromParts(parts []string) (string, bool) {
	switch len(parts) {
	case 0:
		return "", true
	case 1:
		action := parts[0]
		if action == "publish" || action == "unpublish" || action == "move-up" || action == "move-down" {
			return action, true
		}
		return "", false
	case 2:
		if parts[0] != "items" {
			return "", false
		}
		if parts[1] == "add" || parts[1] == "remove" {
			return "items/" + parts[1], true
		}
		return "", false
	default:
		return "", false
	}
}

type railItemOption struct {
	ID    int
	Title string
}

func (s *Server) buildRailItemsForDetail(railData rails.Rail) ([]railItemOption, []railItemOption, error) {
	allItems, err := s.loadRailTypeItems(railData.Type)
	if err != nil {
		return nil, nil, err
	}
	byID := make(map[int]railItemOption, len(allItems))
	for _, item := range allItems {
		byID[item.ID] = item
	}

	selected := make([]railItemOption, 0, len(railData.ItemIDs))
	selectedSet := map[int]struct{}{}
	for _, itemID := range railData.ItemIDs {
		if item, ok := byID[itemID]; ok {
			selected = append(selected, item)
		} else {
			selected = append(selected, railItemOption{ID: itemID, Title: fmt.Sprintf("Item #%d", itemID)})
		}
		selectedSet[itemID] = struct{}{}
	}

	available := make([]railItemOption, 0, len(allItems))
	for _, item := range allItems {
		if _, selectedAlready := selectedSet[item.ID]; selectedAlready {
			continue
		}
		available = append(available, item)
	}
	return available, selected, nil
}

func (s *Server) loadRailTypeItems(railType rails.RailType) ([]railItemOption, error) {
	switch railType {
	case rails.RailTypeBook:
		booksList, err := s.bookStore.List()
		if err != nil {
			return nil, err
		}
		out := make([]railItemOption, 0, len(booksList))
		for _, item := range booksList {
			out = append(out, railItemOption{ID: item.ID, Title: item.Title})
		}
		sortRailItems(out)
		return out, nil
	case rails.RailTypeBundle:
		bundlesList, err := s.bundleStore.List()
		if err != nil {
			return nil, err
		}
		out := make([]railItemOption, 0, len(bundlesList))
		for _, item := range bundlesList {
			out = append(out, railItemOption{ID: item.ID, Title: bundleLabel(item.Name, item.ID)})
		}
		sortRailItems(out)
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported rail type: %s", railType)
	}
}

func sortRailItems(items []railItemOption) {
	sort.Slice(items, func(i, j int) bool {
		left := strings.ToLower(items[i].Title)
		right := strings.ToLower(items[j].Title)
		if left == right {
			return items[i].ID < items[j].ID
		}
		return left < right
	})
}

func railTypeLabel(t rails.RailType) string {
	switch t {
	case rails.RailTypeBook:
		return "Books"
	case rails.RailTypeBundle:
		return "Bundles"
	default:
		return string(t)
	}
}

type railsListViewModel struct {
	Flash           string
	ValidationToast string
	ActiveSection   string
	Rails           []rails.ListItem
}

type railFormInput struct {
	Title string
	Type  string
}

type railFormViewModel struct {
	PageTitle         string
	Action            string
	SubmitLabel       string
	ActiveSection     string
	Flash             string
	ValidationToast   string
	Input             railFormInput
	TypeOptions       []rails.RailType
	ImmutableType     string
	Errors            map[string]string
	IsCreate          bool
	RailID            int
	ShowPublishToggle bool
	PublishAction     string
	PublishLabel      string
	PublishRecency    string
	ShowItemPanel     bool
	AvailableItems    []railItemOption
	SelectedItems     []railItemOption
}

func (m railFormViewModel) HasError(field string) bool {
	_, ok := m.Errors[field]
	return ok
}

func (m railFormViewModel) Error(field string) string {
	return m.Errors[field]
}

func (m railFormViewModel) TypeSelected(value rails.RailType) bool {
	return m.Input.Type == string(value)
}

func (s *Server) renderRailForm(w http.ResponseWriter, data railFormViewModel) {
	if err := railFormTemplate.Execute(w, data); err != nil {
		http.Error(w, "failed to render rail form", http.StatusInternalServerError)
	}
}

var railsListTemplate = template.Must(template.New("rails-list").Funcs(template.FuncMap{
	"adminNav":         adminNav,
	"railTypeLabel":    railTypeLabel,
	"publishState":     publishStateLabel,
	"publishRecency":   publishRecencyLabel,
	"toggleActionPath": toggleActionPath,
}).Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Rails</title>
  <style>
    :root { --bg:#f6f8fb; --card:#fff; --line:#d9e1ea; --text:#1f2937; --accent:#0f766e; --muted:#4b5563; }
    * { box-sizing: border-box; }
    body { margin:0; font-family: "Segoe UI", Tahoma, sans-serif; background: var(--bg); color: var(--text); }
    header { background: var(--card); border-bottom:1px solid var(--line); }
    .shell { width:min(1100px, 94vw); margin:0 auto; padding:16px; }
    .admin-nav { display:flex; gap:14px; }
    .admin-nav-link { color: var(--accent); font-weight:600; text-decoration:none; padding:6px 10px; border-radius:8px; }
    .admin-nav-link.active { background:#e6f4f2; color:#0a5f57; }
    .toolbar { display:flex; align-items:center; justify-content:space-between; margin:16px 0; }
    .button { background: var(--accent); color:white; padding:10px 14px; border-radius:8px; text-decoration:none; font-weight:600; border:none; cursor:pointer; }
    table { width:100%; border-collapse:collapse; background: var(--card); border:1px solid var(--line); border-radius:10px; overflow:hidden; }
    th, td { padding:10px; text-align:left; border-bottom:1px solid var(--line); vertical-align:middle; }
    th { font-size:0.9rem; color:var(--muted); }
    .toast-success { position:fixed; top:16px; right:16px; max-width:min(420px, 90vw); z-index:999; margin:0; padding:10px 12px; border-radius:10px; background:#d1fae5; color:#065f46; border:1px solid #6ee7b7; box-shadow:0 8px 24px rgba(0,0,0,0.12); }
    .toast-error { position:fixed; top:16px; right:16px; max-width:min(420px, 90vw); z-index:999; margin:0; padding:10px 12px; border-radius:10px; background:#fee2e2; color:#991b1b; border:1px solid #fecaca; box-shadow:0 8px 24px rgba(0,0,0,0.12); }
    .inline-publish, .inline-order { display:flex; gap:8px; align-items:center; }
    .toggle { padding:5px 9px; border-radius:999px; border:1px solid var(--line); cursor:pointer; font-weight:600; font-size:0.8rem; background:white; }
    .toggle.on { background:#dcfce7; color:#166534; border-color:#86efac; }
    .toggle.off { background:#f3f4f6; color:#374151; border-color:#d1d5db; }
    .recency { color:var(--muted); font-size:0.8rem; }
    .order-btn { padding:5px 8px; border:1px solid var(--line); border-radius:6px; background:white; cursor:pointer; }
    .row-link { color:var(--accent); font-weight:600; }
  </style>
</head>
<body>
  <header>
    <div class="shell">{{adminNav .ActiveSection}}</div>
  </header>
  <main class="shell">
    <div class="toolbar">
      <h1>Rails</h1>
      <a class="button" href="/admin/rails/new">Add Rail</a>
    </div>
    {{if .Flash}}<p class="toast-success">{{.Flash}}</p>{{end}}
    {{if .ValidationToast}}<p class="toast-error" role="alert">{{.ValidationToast}}</p>{{end}}
    <table>
      <thead>
        <tr>
          <th>Rail Title</th>
          <th>Rail Type</th>
          <th># Items</th>
          <th>Status</th>
          <th>Order</th>
          <th>Action</th>
        </tr>
      </thead>
      <tbody>
      {{if .Rails}}
        {{range .Rails}}
        <tr>
          <td>{{.Title}}</td>
          <td>{{railTypeLabel .Type}}</td>
          <td>{{.ItemCount}}</td>
          <td>
            <form class="inline-publish" method="post" action="/admin/rails/{{.ID}}/{{toggleActionPath .IsPublished}}">
              <button class="toggle {{if .IsPublished}}on{{else}}off{{end}}" type="submit">{{publishState .IsPublished}}</button>
              <span class="recency">{{publishRecency .IsPublished .PublishedAt .UnpublishedAt}}</span>
            </form>
          </td>
          <td>
            <div class="inline-order">
              <form method="post" action="/admin/rails/{{.ID}}/move-up"><button class="order-btn" type="submit">Move Up</button></form>
              <form method="post" action="/admin/rails/{{.ID}}/move-down"><button class="order-btn" type="submit">Move Down</button></form>
            </div>
          </td>
          <td><a class="row-link" href="/admin/rails/{{.ID}}">View/Edit</a></td>
        </tr>
        {{end}}
      {{else}}
        <tr><td colspan="6">No rails yet. Click "Add Rail" to create one.</td></tr>
      {{end}}
      </tbody>
    </table>
  </main>
</body>
</html>
`))

var railFormTemplate = template.Must(template.New("rail-form").Funcs(template.FuncMap{
	"adminNav":      adminNav,
	"railTypeLabel": railTypeLabel,
}).Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.PageTitle}}</title>
  <style>
    :root { --bg:#f6f8fb; --card:#fff; --line:#d9e1ea; --text:#1f2937; --accent:#0f766e; --muted:#4b5563; --error:#b91c1c; }
    * { box-sizing: border-box; }
    body { margin:0; font-family: "Segoe UI", Tahoma, sans-serif; background: var(--bg); color: var(--text); }
    header { background: var(--card); border-bottom:1px solid var(--line); }
    .shell { width:min(1100px, 94vw); margin:0 auto; padding:16px; }
    .admin-nav { display:flex; gap:14px; }
    .admin-nav-link { color: var(--accent); font-weight:600; text-decoration:none; padding:6px 10px; border-radius:8px; }
    .admin-nav-link.active { background:#e6f4f2; color:#0a5f57; }
    .card { background:var(--card); border:1px solid var(--line); border-radius:10px; padding:20px; margin-bottom:16px; }
    .field { margin:0 0 14px; }
    label { display:block; font-weight:600; margin-bottom:6px; }
    input, select { width:100%; padding:10px 12px; border:1px solid var(--line); border-radius:8px; font:inherit; }
    .read-only { padding:10px 12px; border:1px solid var(--line); border-radius:8px; background:#f8fafc; font-weight:600; }
    .error { color: var(--error); margin-top:6px; font-size:0.9rem; }
    .toast-success { position:fixed; top:16px; right:16px; max-width:min(420px, 90vw); z-index:999; margin:0; padding:10px 12px; border-radius:10px; background:#d1fae5; color:#065f46; border:1px solid #6ee7b7; box-shadow:0 8px 24px rgba(0,0,0,0.12); }
    .toast-error { position:fixed; top:16px; right:16px; max-width:min(420px, 90vw); z-index:999; margin:0; padding:10px 12px; border-radius:10px; background:#fee2e2; color:#991b1b; border:1px solid #fecaca; box-shadow:0 8px 24px rgba(0,0,0,0.12); }
    .button { background: var(--accent); color:white; padding:10px 14px; border-radius:8px; text-decoration:none; border:none; font-weight:600; cursor:pointer; }
    .row { display:flex; gap:10px; align-items:center; }
    .secondary { color:var(--accent); text-decoration:none; font-weight:600; }
    .publish-box { margin:0 0 14px; padding:12px; background:#f8fafc; border:1px solid var(--line); border-radius:8px; display:flex; gap:10px; align-items:center; }
    .toggle { padding:6px 10px; border-radius:999px; border:1px solid var(--line); cursor:pointer; font-weight:600; font-size:0.85rem; background:white; }
    .toggle.on { background:#dcfce7; color:#166534; border-color:#86efac; }
    .toggle.off { background:#f3f4f6; color:#374151; border-color:#d1d5db; }
    .recency { color:var(--muted); font-size:0.85rem; }
    .picker-grid { display:grid; grid-template-columns: 1fr 1fr; gap:14px; }
    table { width:100%; border-collapse:collapse; border:1px solid var(--line); border-radius:8px; overflow:hidden; }
    th, td { border-bottom:1px solid var(--line); padding:8px; text-align:left; vertical-align:middle; }
    th { background:#f8fafc; color:var(--muted); font-size:0.85rem; }
    .tiny-btn { padding:6px 10px; border:1px solid var(--line); border-radius:6px; background:#fff; cursor:pointer; }
    @media (max-width: 960px) { .picker-grid { grid-template-columns: 1fr; } }
  </style>
</head>
<body>
  <header>
    <div class="shell">{{adminNav .ActiveSection}}</div>
  </header>
  <main class="shell">
    <h1>{{.PageTitle}}</h1>
    {{if .Flash}}<p class="toast-success">{{.Flash}}</p>{{end}}
    {{if .ValidationToast}}<p class="toast-error" role="alert">{{.ValidationToast}}</p>{{end}}

    {{if .ShowPublishToggle}}
    <div class="publish-box">
      <form method="post" action="{{.PublishAction}}">
        <button class="toggle {{if eq .PublishLabel "Published"}}on{{else}}off{{end}}" type="submit">{{.PublishLabel}}</button>
      </form>
      <span class="recency">{{.PublishRecency}}</span>
    </div>
    {{end}}

    <form class="card" method="post" action="{{.Action}}">
      <div class="field">
        <label for="title">Rail Title</label>
        <input id="title" name="title" value="{{.Input.Title}}" required>
        {{if .HasError "title"}}<div class="error">{{.Error "title"}}</div>{{end}}
      </div>

      {{if .IsCreate}}
      <div class="field">
        <label for="type">Rail Type</label>
        <select id="type" name="type" required>
          <option value="">Select rail type</option>
          {{range .TypeOptions}}
          <option value="{{.}}" {{if $.TypeSelected .}}selected{{end}}>{{railTypeLabel .}}</option>
          {{end}}
        </select>
        {{if .HasError "type"}}<div class="error">{{.Error "type"}}</div>{{end}}
      </div>
      {{else}}
      <div class="field">
        <label>Rail Type (immutable)</label>
        <div class="read-only">{{.ImmutableType}}</div>
      </div>
      {{end}}

      <div class="row">
        <button class="button" type="submit">{{.SubmitLabel}}</button>
        <a class="secondary" href="/admin/rails">Back to Rails</a>
      </div>
    </form>

    {{if .ShowItemPanel}}
    <section class="card">
      <h2>Item Assignment</h2>
      <div class="field">
        <label for="rail-item-search">Search available items (title)</label>
        <input id="rail-item-search" placeholder="Search title">
      </div>
      <div class="picker-grid">
        <div>
          <h3>Available items</h3>
          <table>
            <thead><tr><th>Title</th><th></th></tr></thead>
            <tbody>
            {{range .AvailableItems}}
              <tr data-rail-item-row data-title="{{.Title}}">
                <td>{{.Title}}</td>
                <td>
                  <form method="post" action="/admin/rails/{{$.RailID}}/items/add">
                    <input type="hidden" name="item_id" value="{{.ID}}">
                    <button class="tiny-btn" type="submit">Add</button>
                  </form>
                </td>
              </tr>
            {{else}}
              <tr><td colspan="2">No available items.</td></tr>
            {{end}}
            </tbody>
          </table>
        </div>
        <div>
          <h3>Selected items</h3>
          <table>
            <thead><tr><th>Title</th><th></th></tr></thead>
            <tbody>
            {{range .SelectedItems}}
              <tr>
                <td>{{.Title}}</td>
                <td>
                  <form method="post" action="/admin/rails/{{$.RailID}}/items/remove">
                    <input type="hidden" name="item_id" value="{{.ID}}">
                    <button class="tiny-btn" type="submit">Remove</button>
                  </form>
                </td>
              </tr>
            {{else}}
              <tr><td colspan="2">No selected items.</td></tr>
            {{end}}
            </tbody>
          </table>
        </div>
      </div>
    </section>
    {{end}}
  </main>
  {{if .ShowItemPanel}}<script src="/assets/rails-form.js" defer></script>{{end}}
</body>
</html>
`))
