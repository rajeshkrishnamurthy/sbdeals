package web

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rajeshkrishnamurthy/sbdeals/internal/books"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/bundles"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/clicked"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/rails"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/suppliers"
)

func TestHandleClickedCreate(t *testing.T) {
	stub := &clickedStoreStub{}
	s := newServerWithClickedStore(stub)

	req := httptest.NewRequest(http.MethodPost, "/api/clicked", bytes.NewBufferString(`{"itemId":7,"itemType":"BOOK","itemTitle":"Book One","sourcePage":"catalog","sourceRailId":2,"sourceRailTitle":"Fresh Books"}`))
	rr := httptest.NewRecorder()

	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
	if stub.created.ItemID != 7 || stub.created.ItemType != clicked.ItemTypeBook {
		t.Fatalf("unexpected created payload: %+v", stub.created)
	}
}

func TestHandleClickedCreateValidationAndError(t *testing.T) {
	t.Run("invalid payload", func(t *testing.T) {
		s := newServerWithClickedStore(&clickedStoreStub{})
		req := httptest.NewRequest(http.MethodPost, "/api/clicked", bytes.NewBufferString(`{"itemId":0}`))
		rr := httptest.NewRecorder()

		s.Handler().ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("store failure", func(t *testing.T) {
		s := newServerWithClickedStore(&clickedStoreStub{err: errors.New("boom")})
		req := httptest.NewRequest(http.MethodPost, "/api/clicked", bytes.NewBufferString(`{"itemId":1,"itemType":"BUNDLE","itemTitle":"Bundle","sourcePage":"catalog","sourceRailId":1,"sourceRailTitle":"Weekend Bundles"}`))
		rr := httptest.NewRecorder()

		s.Handler().ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", rr.Code)
		}
	})
}

func TestHandleClickedCreateMethodNotAllowed(t *testing.T) {
	s := newServerWithClickedStore(&clickedStoreStub{})
	req := httptest.NewRequest(http.MethodGet, "/api/clicked", nil)
	rr := httptest.NewRecorder()

	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

type clickedStoreStub struct {
	created clicked.CreateInput
	err     error
}

func (s *clickedStoreStub) CreateClicked(input clicked.CreateInput) (clicked.Enquiry, error) {
	s.created = input
	if s.err != nil {
		return clicked.Enquiry{}, s.err
	}
	return clicked.Enquiry{}, nil
}

func (s *clickedStoreStub) ListByStatus(status clicked.Status) ([]clicked.Enquiry, error) {
	return nil, nil
}

func (s *clickedStoreStub) ConvertToInterested(id int, input clicked.ConvertInput) (clicked.Enquiry, bool, error) {
	return clicked.Enquiry{}, false, nil
}

func newServerWithClickedStore(clickedStore clicked.Store) *Server {
	return NewServerWithStoresAndClicked(
		suppliers.NewMemoryStore(),
		books.NewMemoryStore(),
		bundles.NewMemoryStore(nil, nil),
		rails.NewMemoryStore(),
		clickedStore,
	)
}
