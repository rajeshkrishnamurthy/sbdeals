package web

import (
	"fmt"
	"html/template"
	"strings"
)

func adminNav(activeSection string) template.HTML {
	items := []struct {
		label   string
		path    string
		section string
	}{
		{label: "Suppliers", path: "/admin/suppliers", section: "suppliers"},
		{label: "Books", path: "/admin/books", section: "books"},
		{label: "Bundles", path: "/admin/bundles", section: "bundles"},
		{label: "Rails", path: "/admin/rails", section: "rails"},
		{label: "Enquiries", path: "/admin/enquiries", section: "enquiries"},
		{label: "Customers", path: "/admin/customers", section: "customers"},
	}

	parts := make([]string, 0, len(items)+2)
	parts = append(parts, `<nav class="admin-nav">`)
	for _, item := range items {
		className := "admin-nav-link"
		if item.section == activeSection {
			className += " active"
		}
		parts = append(parts, fmt.Sprintf(`<a href="%s" class="%s">%s</a>`, item.path, className, item.label))
	}
	parts = append(parts, `</nav>`)
	return template.HTML(strings.Join(parts, ""))
}
