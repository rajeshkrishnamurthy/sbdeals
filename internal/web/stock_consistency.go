package web

import (
	"errors"

	"github.com/rajeshkrishnamurthy/sbdeals/internal/books"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/bundles"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/rails"
)

type bundleStockSyncer interface {
	SyncDerivedStockByBook(bookID int, inStock bool) error
}

func (s *Server) enforceBundleStockFromBook(bookID int, inStock bool) error {
	syncer, ok := s.bundleStore.(bundleStockSyncer)
	if !ok {
		return nil
	}
	return syncer.SyncDerivedStockByBook(bookID, inStock)
}

func (s *Server) enforceRailPublicationConsistency() error {
	items, err := s.railStore.List()
	if err != nil {
		return err
	}

	for _, item := range items {
		if !item.IsPublished {
			continue
		}
		railData, err := s.railStore.Get(item.ID)
		if errors.Is(err, rails.ErrNotFound) {
			continue
		}
		if err != nil {
			return err
		}

		hasPublishedItems, err := s.railHasPublishedItems(railData)
		if err != nil {
			return err
		}
		if hasPublishedItems {
			continue
		}
		if _, err := s.railStore.Unpublish(railData.ID); err != nil && !errors.Is(err, rails.ErrNotFound) {
			return err
		}
	}
	return nil
}

func (s *Server) railHasPublishedItems(railData rails.Rail) (bool, error) {
	for _, itemID := range railData.ItemIDs {
		switch railData.Type {
		case rails.RailTypeBook:
			book, err := s.bookStore.Get(itemID)
			if errors.Is(err, books.ErrNotFound) {
				continue
			}
			if err != nil {
				return false, err
			}
			if book.IsPublished {
				return true, nil
			}
		case rails.RailTypeBundle:
			bundle, err := s.bundleStore.Get(itemID)
			if errors.Is(err, bundles.ErrNotFound) {
				continue
			}
			if err != nil {
				return false, err
			}
			if bundle.IsPublished {
				return true, nil
			}
		}
	}
	return false, nil
}
