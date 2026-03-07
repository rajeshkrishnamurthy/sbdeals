// Rail picker filtering is deterministic and server-driven.
const railPickerSearch = document.getElementById("rail-item-search");

if (railPickerSearch && window.location.hash === "#rail-picker") {
  railPickerSearch.focus();
}
