const railItemSearch = document.getElementById("rail-item-search");
const railAvailableRows = Array.from(document.querySelectorAll("[data-rail-item-row]"));

if (railItemSearch && railAvailableRows.length > 0) {
  const filterRows = () => {
    const term = railItemSearch.value.trim().toLowerCase();
    railAvailableRows.forEach((row) => {
      const title = (row.dataset.title || "").toLowerCase();
      row.hidden = term !== "" && !title.includes(term);
    });
  };

  railItemSearch.addEventListener("input", filterRows);
  filterRows();
}
