const railItemSearch = document.getElementById("rail-item-search") as HTMLInputElement | null;
const railAvailableRows = Array.from(document.querySelectorAll<HTMLTableRowElement>("[data-rail-item-row]"));

if (railItemSearch && railAvailableRows.length > 0) {
  const filterRows = (): void => {
    const term = railItemSearch.value.trim().toLowerCase();
    railAvailableRows.forEach((row) => {
      const title = (row.dataset.title ?? "").toLowerCase();
      row.hidden = term !== "" && !title.includes(term);
    });
  };

  railItemSearch.addEventListener("input", filterRows);
  filterRows();
}
