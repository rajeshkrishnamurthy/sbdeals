package web

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/rajeshkrishnamurthy/sbdeals/internal/clicked"
)

type clickedCreateRequest struct {
	ItemID       int    `json:"itemId"`
	ItemType     string `json:"itemType"`
	ItemTitle    string `json:"itemTitle"`
	SourcePage   string `json:"sourcePage"`
	SourceRailID int    `json:"sourceRailId"`
	SourceRail   string `json:"sourceRailTitle"`
}

func (s *Server) handleClickedCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	var req clickedCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}

	itemType := clicked.ItemType(strings.TrimSpace(req.ItemType))
	if req.ItemID <= 0 || strings.TrimSpace(req.ItemTitle) == "" || strings.TrimSpace(req.SourcePage) == "" || !isValidClickedItemType(itemType) {
		http.Error(w, "invalid clicked payload", http.StatusBadRequest)
		return
	}

	_, err := s.clickedStore.Create(clicked.CreateInput{
		ItemID:       req.ItemID,
		ItemType:     itemType,
		ItemTitle:    strings.TrimSpace(req.ItemTitle),
		SourcePage:   strings.TrimSpace(req.SourcePage),
		SourceRailID: req.SourceRailID,
		SourceRail:   strings.TrimSpace(req.SourceRail),
	})
	if err != nil {
		log.Printf("clicked create failed: %v", err)
		http.Error(w, "failed to create clicked", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func isValidClickedItemType(itemType clicked.ItemType) bool {
	return itemType == clicked.ItemTypeBook || itemType == clicked.ItemTypeBundle
}
