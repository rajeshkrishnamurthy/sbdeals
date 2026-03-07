const supplierSelect = document.getElementById("supplier_id");
const categorySelect = document.getElementById("category");
const conditionInputs = Array.from(document.querySelectorAll('input[name="allowed_conditions"]'));
const searchInput = document.getElementById("bundle-book-search");
const bundlePriceInput = document.getElementById("bundle_price");
const pickerBody = document.getElementById("bundle-picker-body");
const selectedBody = document.getElementById("bundle-selected-body");
const hiddenBookIDs = document.getElementById("bundle-book-ids");
const totalMRP = document.getElementById("bundle-total-mrp");
const totalMyPrice = document.getElementById("bundle-total-my-price");
const totalMyBundle = document.getElementById("bundle-total-my-bundle");
const totalDiscount = document.getElementById("bundle-total-discount");

const isReady =
  supplierSelect &&
  categorySelect &&
  conditionInputs.length > 0 &&
  searchInput &&
  bundlePriceInput &&
  pickerBody &&
  selectedBody &&
  hiddenBookIDs &&
  totalMRP &&
  totalMyPrice &&
  totalMyBundle &&
  totalDiscount;

if (isReady) {
  const booksByID = new Map();
  const selectedBookIDs = [];

  const money = (value) => value.toFixed(2);
  const discountText = (bundleMRP, bundlePriceRaw) => {
    const bundlePrice = Number.parseFloat(bundlePriceRaw);
    if (!Number.isFinite(bundlePrice) || bundleMRP <= 0) {
      return "0%";
    }
    const discount = ((bundleMRP - bundlePrice) / bundleMRP) * 100;
    return `${Math.round(discount)}%`;
  };

  const hasSelected = (bookID) => selectedBookIDs.includes(bookID);

  const selectedConditionSet = () => {
    const values = conditionInputs.filter((item) => item.checked).map((item) => item.value);
    return new Set(values);
  };

  const selectedSupplierID = () => Number.parseInt(supplierSelect.value, 10);
  const selectedCategory = () => categorySelect.value.trim();

  const isBookEligible = (book) => {
    const supplierID = selectedSupplierID();
    if (!Number.isFinite(supplierID) || supplierID <= 0 || book.supplierID !== supplierID) {
      return false;
    }

    const category = selectedCategory();
    if (!category || book.category !== category) {
      return false;
    }

    const conditions = selectedConditionSet();
    if (conditions.size === 0 || !conditions.has(book.condition)) {
      return false;
    }

    return true;
  };

  const searchTerm = () => searchInput.value.trim().toLowerCase();

  const matchesSearch = (book) => {
    const term = searchTerm();
    if (!term) {
      return true;
    }
    return book.title.toLowerCase().includes(term) || book.author.toLowerCase().includes(term);
  };

  const refreshHiddenBookIDs = () => {
    hiddenBookIDs.innerHTML = "";
    selectedBookIDs.forEach((bookID) => {
      const input = document.createElement("input");
      input.type = "hidden";
      input.name = "book_ids";
      input.value = String(bookID);
      hiddenBookIDs.appendChild(input);
    });
  };

  const refreshTotals = () => {
    let bundleMRP = 0;
    let sumMyPrice = 0;
    let sumMyBundle = 0;

    selectedBookIDs.forEach((bookID) => {
      const book = booksByID.get(bookID);
      if (!book) {
        return;
      }
      bundleMRP += book.mrp;
      sumMyPrice += book.myPrice;
      sumMyBundle += book.effectiveBundlePrice;
    });

    totalMRP.textContent = money(bundleMRP);
    totalMyPrice.textContent = money(sumMyPrice);
    totalMyBundle.textContent = money(sumMyBundle);
    totalDiscount.textContent = discountText(bundleMRP, bundlePriceInput.value);
  };

  const textCell = (value) => {
    const cell = document.createElement("td");
    cell.textContent = value;
    return cell;
  };

  const selectedRow = (book) => {
    const row = document.createElement("tr");
    row.setAttribute("data-selected-book", String(book.id));
    row.appendChild(textCell(book.title));
    row.appendChild(textCell(book.author));
    row.appendChild(textCell(book.condition));
    row.appendChild(textCell(money(book.mrp)));
    row.appendChild(textCell(money(book.myPrice)));

    const action = document.createElement("td");
    const button = document.createElement("button");
    button.className = "tiny-btn";
    button.type = "button";
    button.dataset.removeBook = String(book.id);
    button.textContent = "Remove";
    action.appendChild(button);
    row.appendChild(action);
    return row;
  };

  const refreshSelectedBooks = () => {
    selectedBody.innerHTML = "";
    selectedBookIDs.forEach((bookID) => {
      const book = booksByID.get(bookID);
      if (!book) {
        return;
      }
      selectedBody.appendChild(selectedRow(book));
    });
    refreshHiddenBookIDs();
    refreshTotals();
  };

  const refreshPicker = () => {
    booksByID.forEach((book) => {
      const visible = isBookEligible(book) && matchesSearch(book);
      book.row.hidden = !visible;

      if (!book.addButton) {
        return;
      }
      const isAdded = hasSelected(book.id);
      book.addButton.disabled = isAdded;
      book.addButton.textContent = isAdded ? "Added" : "Add";
    });
  };

  const addSelected = (bookID) => {
    if (!booksByID.has(bookID) || hasSelected(bookID)) {
      return;
    }
    selectedBookIDs.push(bookID);
    refreshSelectedBooks();
    refreshPicker();
  };

  const removeSelected = (bookID) => {
    const idx = selectedBookIDs.indexOf(bookID);
    if (idx < 0) {
      return;
    }
    selectedBookIDs.splice(idx, 1);
    refreshSelectedBooks();
    refreshPicker();
  };

  const revalidateSelectedBooks = () => {
    for (let i = selectedBookIDs.length - 1; i >= 0; i -= 1) {
      const book = booksByID.get(selectedBookIDs[i]);
      if (!book || !isBookEligible(book)) {
        selectedBookIDs.splice(i, 1);
      }
    }
  };

  const parseNumber = (value) => {
    const parsed = Number.parseFloat(value);
    if (!Number.isFinite(parsed)) {
      return 0;
    }
    return parsed;
  };

  Array.from(pickerBody.querySelectorAll("tr[data-picker-book-row]")).forEach((row) => {
    const id = Number.parseInt(row.dataset.bookId || "", 10);
    if (!Number.isFinite(id) || id <= 0) {
      return;
    }
    const book = {
      id,
      title: row.dataset.title || "",
      author: row.dataset.author || "",
      supplierID: Number.parseInt(row.dataset.supplierId || "", 10),
      category: row.dataset.category || "",
      condition: row.dataset.condition || "",
      mrp: parseNumber(row.dataset.mrp || "0"),
      myPrice: parseNumber(row.dataset.myPrice || "0"),
      effectiveBundlePrice: parseNumber(row.dataset.bundleEffective || "0"),
      row,
      addButton: row.querySelector("[data-add-book]"),
    };
    booksByID.set(book.id, book);
  });

  Array.from(hiddenBookIDs.querySelectorAll('input[name="book_ids"]')).forEach((input) => {
    const id = Number.parseInt(input.value, 10);
    if (!Number.isFinite(id) || id <= 0 || hasSelected(id)) {
      return;
    }
    selectedBookIDs.push(id);
  });

  pickerBody.addEventListener("click", (event) => {
    const target = event.target;
    if (!target) {
      return;
    }
    const button = target.closest("[data-add-book]");
    if (!button) {
      return;
    }
    const id = Number.parseInt(button.dataset.addBook || "", 10);
    if (!Number.isFinite(id) || id <= 0) {
      return;
    }
    addSelected(id);
  });

  selectedBody.addEventListener("click", (event) => {
    const target = event.target;
    if (!target) {
      return;
    }
    const button = target.closest("[data-remove-book]");
    if (!button) {
      return;
    }
    const id = Number.parseInt(button.dataset.removeBook || "", 10);
    if (!Number.isFinite(id) || id <= 0) {
      return;
    }
    removeSelected(id);
  });

  searchInput.addEventListener("input", refreshPicker);
  supplierSelect.addEventListener("change", () => {
    revalidateSelectedBooks();
    refreshSelectedBooks();
    refreshPicker();
  });
  categorySelect.addEventListener("change", () => {
    revalidateSelectedBooks();
    refreshSelectedBooks();
    refreshPicker();
  });
  conditionInputs.forEach((input) =>
    input.addEventListener("change", () => {
      revalidateSelectedBooks();
      refreshSelectedBooks();
      refreshPicker();
    }),
  );
  bundlePriceInput.addEventListener("input", refreshTotals);

  revalidateSelectedBooks();
  refreshSelectedBooks();
  refreshPicker();
}
