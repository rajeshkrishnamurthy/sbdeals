package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/rajeshkrishnamurthy/sbdeals/internal/books"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/bundles"
	"github.com/rajeshkrishnamurthy/sbdeals/internal/rails"
)

type catalogPageViewModel struct {
	DataEndpoint string
}

type catalogResponse struct {
	Rails []catalogRailResponse `json:"rails"`
}

type catalogRailResponse struct {
	ID    int                   `json:"id"`
	Title string                `json:"title"`
	Type  string                `json:"type"`
	Items []catalogItemResponse `json:"items"`
}

type catalogItemResponse struct {
	ID                 int    `json:"id"`
	Type               string `json:"type"`
	Title              string `json:"title"`
	ImageURL           string `json:"imageUrl"`
	CurrentPriceText   string `json:"currentPriceText"`
	OriginalPriceText  string `json:"originalPriceText,omitempty"`
	DiscountText       string `json:"discountText,omitempty"`
	ReserveButtonLabel string `json:"reserveButtonLabel"`
	WhatsAppMessage    string `json:"whatsAppMessage"`
}

func (s *Server) renderCatalogPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	data := catalogPageViewModel{DataEndpoint: "/api/catalog"}
	if err := catalogPageTemplate.Execute(w, data); err != nil {
		http.Error(w, "failed to render catalog page", http.StatusInternalServerError)
	}
}

func (s *Server) handleCatalogData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	payload, err := s.buildCatalogResponse()
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to load catalog"})
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "failed to write catalog response", http.StatusInternalServerError)
	}
}

func (s *Server) buildCatalogResponse() (catalogResponse, error) {
	railItems, err := s.railStore.List()
	if err != nil {
		return catalogResponse{}, err
	}

	response := catalogResponse{Rails: make([]catalogRailResponse, 0, len(railItems))}
	for _, item := range railItems {
		if !item.IsPublished {
			continue
		}

		railData, err := s.railStore.Get(item.ID)
		if err != nil {
			return catalogResponse{}, err
		}

		items, err := s.buildCatalogItemsForRail(railData)
		if err != nil {
			return catalogResponse{}, err
		}

		response.Rails = append(response.Rails, catalogRailResponse{
			ID:    railData.ID,
			Title: railData.Title,
			Type:  string(railData.Type),
			Items: items,
		})
	}

	return response, nil
}

func (s *Server) buildCatalogItemsForRail(railData rails.Rail) ([]catalogItemResponse, error) {
	items := make([]catalogItemResponse, 0, len(railData.ItemIDs))
	for _, itemID := range railData.ItemIDs {
		item, visible, err := s.buildCatalogItem(railData.Type, itemID)
		if err != nil {
			if isCatalogItemNotFound(err) {
				continue
			}
			return nil, err
		}
		if !visible {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Server) buildCatalogItem(railType rails.RailType, itemID int) (catalogItemResponse, bool, error) {
	switch railType {
	case rails.RailTypeBook:
		return s.buildCatalogBookItem(itemID)
	case rails.RailTypeBundle:
		return s.buildCatalogBundleItem(itemID)
	default:
		return catalogItemResponse{}, false, fmt.Errorf("unsupported rail type: %s", railType)
	}
}

func (s *Server) buildCatalogBookItem(itemID int) (catalogItemResponse, bool, error) {
	book, err := s.bookStore.Get(itemID)
	if err != nil {
		return catalogItemResponse{}, false, err
	}
	if !book.IsPublished || !book.InStock {
		return catalogItemResponse{}, false, nil
	}

	item := catalogItemResponse{
		ID:                 book.ID,
		Type:               string(rails.RailTypeBook),
		Title:              strings.TrimSpace(book.Title),
		CurrentPriceText:   catalogMoney(book.MyPrice),
		ReserveButtonLabel: "I'm interested",
		WhatsAppMessage:    buildBookWhatsAppMessage(book.Title),
	}
	if strings.TrimSpace(book.CoverMimeType) != "" {
		item.ImageURL = fmt.Sprintf("/admin/books/%d/cover", book.ID)
	}
	if book.MRP > 0 {
		item.OriginalPriceText = catalogMoney(book.MRP)
		item.DiscountText = formatRoundedDiscount(books.ComputeDiscount(book.MRP, book.MyPrice))
	}
	return item, true, nil
}

func (s *Server) buildCatalogBundleItem(itemID int) (catalogItemResponse, bool, error) {
	bundle, err := s.bundleStore.Get(itemID)
	if err != nil {
		return catalogItemResponse{}, false, err
	}
	if !bundle.IsPublished || !bundleBooksInStock(bundle.Books) {
		return catalogItemResponse{}, false, nil
	}

	item := catalogItemResponse{
		ID:                 bundle.ID,
		Type:               string(rails.RailTypeBundle),
		Title:              bundleLabel(bundle.Name, bundle.ID),
		CurrentPriceText:   catalogMoney(bundle.BundlePrice),
		ReserveButtonLabel: "I'm interested",
		WhatsAppMessage:    buildBundleWhatsAppMessage(bundle.Books),
	}
	if strings.TrimSpace(bundle.ImageMimeType) != "" {
		item.ImageURL = fmt.Sprintf("/admin/bundles/%d/image", bundle.ID)
	}
	bundleMRP := sumCatalogBundleMRP(bundle.Books)
	if bundleMRP > 0 {
		item.OriginalPriceText = catalogMoney(bundleMRP)
		item.DiscountText = formatBundleDiscountPercent(bundleMRP, bundle.BundlePrice)
	}
	return item, true, nil
}

func isCatalogItemNotFound(err error) bool {
	return errors.Is(err, books.ErrNotFound) || errors.Is(err, bundles.ErrNotFound)
}

func bundleBooksInStock(items []bundles.BundleBook) bool {
	for _, item := range items {
		if !item.InStock {
			return false
		}
	}
	return true
}

func sumCatalogBundleMRP(items []bundles.BundleBook) float64 {
	total := 0.0
	for _, item := range items {
		total += item.MRP
	}
	return total
}

func catalogMoney(value float64) string {
	return fmt.Sprintf("Rs. %.2f", value)
}

func buildBookWhatsAppMessage(title string) string {
	return fmt.Sprintf("Hi Srikar, I'm interested in this book: %s.", strings.TrimSpace(title))
}

func buildBundleWhatsAppMessage(items []bundles.BundleBook) string {
	titles := make([]string, 0, 3)
	for _, item := range items {
		if len(titles) == 3 {
			break
		}
		title := strings.TrimSpace(item.Title)
		if title == "" {
			continue
		}
		titles = append(titles, title)
	}
	return fmt.Sprintf("Hi Srikar, I'm interested in this bundle containing: %s.", strings.Join(titles, ", "))
}

var catalogPageTemplate = template.Must(template.New("catalog-page").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Srikar Book Deals</title>
  <style>
    :root {
      --bg:#f5efe6;
      --paper:#fffaf2;
      --ink:#1f2933;
      --muted:#6b7280;
      --line:#d9cdb9;
      --accent:#0f766e;
      --accent-strong:#115e59;
      --danger:#b91c1c;
      --card-shadow:0 14px 34px rgba(31,41,51,0.08);
    }
    * { box-sizing:border-box; }
    body {
      margin:0;
      font-family:"Avenir Next", "Segoe UI", sans-serif;
      color:var(--ink);
      background:
        radial-gradient(circle at top, rgba(15,118,110,0.10), transparent 28%),
        linear-gradient(180deg, #fcf7ef 0%, var(--bg) 100%);
    }
    .page { width:min(1220px, 94vw); margin:0 auto; padding:24px 0 64px; }
    .hero {
      background:linear-gradient(135deg, rgba(255,250,242,0.96), rgba(246,238,225,0.96));
      border:1px solid rgba(217,205,185,0.9);
      border-radius:24px;
      padding:28px;
      box-shadow:var(--card-shadow);
    }
    .eyebrow {
      margin:0 0 8px;
      color:var(--accent-strong);
      text-transform:uppercase;
      letter-spacing:0.18em;
      font-size:0.74rem;
      font-weight:700;
    }
    h1 {
      margin:0;
      font-size:clamp(2rem, 5vw, 3.6rem);
      line-height:1;
      letter-spacing:-0.03em;
    }
    .hero-copy {
      max-width:48rem;
      margin:14px 0 0;
      color:var(--muted);
      font-size:1rem;
      line-height:1.6;
    }
    .catalog-region { margin-top:28px; }
    .catalog-loading, .catalog-error, .catalog-empty {
      background:var(--paper);
      border:1px solid var(--line);
      border-radius:20px;
      padding:22px;
      box-shadow:var(--card-shadow);
    }
    .catalog-error p, .catalog-empty p, .catalog-loading p { margin:0; }
    .catalog-error { display:grid; gap:14px; }
    .catalog-error button, .catalog-loading button, .catalog-empty button, .cta {
      appearance:none;
      border:none;
      border-radius:999px;
      background:var(--accent);
      color:#fff;
      cursor:pointer;
      font:inherit;
      font-weight:700;
      padding:12px 16px;
    }
    .catalog-error button:hover, .cta:hover { background:var(--accent-strong); }
    .cta-content { display:inline-flex; align-items:center; gap:8px; }
    .cta-icon { width:16px; height:16px; display:block; }
    .rail-list { display:grid; gap:28px; }
    .rail-section { display:grid; gap:14px; }
    .rail-header { display:flex; justify-content:space-between; align-items:flex-end; gap:16px; }
    .rail-title { margin:0; font-size:1.4rem; letter-spacing:-0.02em; }
    .rail-kind { color:var(--muted); font-size:0.85rem; text-transform:uppercase; letter-spacing:0.12em; }
    .rail-row {
      display:grid;
      grid-auto-flow:column;
      grid-auto-columns:minmax(220px, 260px);
      gap:14px;
      overflow-x:auto;
      padding:4px 2px 8px;
      scrollbar-width:thin;
      scroll-snap-type:x proximity;
      -webkit-overflow-scrolling:touch;
    }
    .rail-empty {
      background:rgba(255,250,242,0.86);
      border:1px dashed var(--line);
      border-radius:18px;
      padding:18px;
      color:var(--muted);
      min-height:140px;
      display:flex;
      align-items:center;
    }
    .catalog-card {
      background:var(--paper);
      border:1px solid rgba(217,205,185,0.9);
      border-radius:20px;
      padding:14px;
      display:grid;
      gap:14px;
      min-height:100%;
      box-shadow:var(--card-shadow);
      scroll-snap-align:start;
    }
    .catalog-card-media {
      aspect-ratio:4 / 5;
      border-radius:16px;
      overflow:hidden;
      background:linear-gradient(180deg, rgba(15,118,110,0.08), rgba(17,94,89,0.18));
      border:1px solid rgba(217,205,185,0.8);
      display:flex;
      align-items:center;
      justify-content:center;
    }
    .catalog-card-media img {
      width:100%;
      height:100%;
      object-fit:contain;
      display:block;
      background:#fff;
    }
    .catalog-card-placeholder {
      color:#fff;
      font-weight:700;
      letter-spacing:0.12em;
      text-transform:uppercase;
      font-size:0.75rem;
    }
    .catalog-card-title {
      margin:0;
      font-size:1rem;
      line-height:1.35;
      min-height:2.7em;
    }
    .catalog-price { display:flex; flex-wrap:wrap; gap:8px 10px; align-items:baseline; }
    .catalog-price-current { font-size:1.15rem; font-weight:800; letter-spacing:-0.02em; }
    .catalog-price-original { color:var(--muted); text-decoration:line-through; }
    .catalog-price-discount {
      color:var(--accent-strong);
      font-weight:700;
      background:rgba(15,118,110,0.10);
      padding:4px 8px;
      border-radius:999px;
      font-size:0.8rem;
    }
    .catalog-toast {
      position:fixed;
      left:50%;
      bottom:22px;
      transform:translateX(-50%) translateY(12px);
      background:#1f2933;
      color:#fff;
      padding:12px 16px;
      border-radius:999px;
      box-shadow:0 12px 28px rgba(0,0,0,0.22);
      opacity:0;
      pointer-events:none;
      transition:opacity 160ms ease, transform 160ms ease;
    }
    .catalog-toast.visible {
      opacity:1;
      transform:translateX(-50%) translateY(0);
    }
    @media (max-width: 720px) {
      .page { width:min(96vw, 100%); padding-top:16px; }
      .hero { padding:20px; border-radius:20px; }
      .rail-row { grid-auto-columns:minmax(190px, 78vw); }
    }
  </style>
</head>
<body>
  <main class="page">
    <section class="hero">
      <p class="eyebrow">Curated Deals</p>
      <h1>Srikar Book Deals</h1>
      <p class="hero-copy">Browse curated rails across books and bundles, and tap I’m interested to continue the conversation on WhatsApp.</p>
    </section>
    <section class="catalog-region">
      <div id="catalog-root" data-endpoint="{{.DataEndpoint}}" class="catalog-loading" aria-live="polite">
        <p>Loading curated rails...</p>
      </div>
    </section>
  </main>
  <div id="catalog-toast" class="catalog-toast" role="status" aria-live="polite"></div>
  <script src="/assets/catalog.js" defer></script>
</body>
</html>
`))
