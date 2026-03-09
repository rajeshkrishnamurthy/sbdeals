type ConvertModalState = {
  enquiryID: string;
  customerID?: string;
  quickName?: string;
  quickMobile?: string;
  note?: string;
  search?: string;
};

type OrderModalState = {
  enquiryID: string;
  customerName: string;
  customerMobile: string;
  hasAddress: boolean;
  requireAddress?: boolean;
  amount?: string;
  note?: string;
  address?: string;
};

const initConvertModal = (): void => {
  const dialog = document.getElementById("convert-dialog") as HTMLDialogElement | null;
  const form = document.getElementById("convert-form") as HTMLFormElement | null;
  const cancel = document.getElementById("cancel-convert") as HTMLButtonElement | null;
  const search = document.getElementById("customer-search") as HTMLInputElement | null;
  const select = document.getElementById("customer-id") as HTMLSelectElement | null;
  const quickName = document.getElementById("quick-customer-name") as HTMLInputElement | null;
  const quickMobile = document.getElementById("quick-customer-mobile") as HTMLInputElement | null;
  const note = document.getElementById("enquiry-note") as HTMLTextAreaElement | null;

  if (!dialog || !form || !cancel || !search || !select || !quickName || !quickMobile || !note) {
    return;
  }

  const originalOptions = Array.from(select.querySelectorAll<HTMLOptionElement>("option"));

  const filterOptions = (): void => {
    const query = search.value.toLowerCase().trim();
    select.innerHTML = "";

    originalOptions.forEach((option) => {
      if (!option.value) {
        select.appendChild(option.cloneNode(true));
        return;
      }
      const label = (option.getAttribute("data-customer-label") ?? "").toLowerCase();
      if (!query || label.includes(query)) {
        select.appendChild(option.cloneNode(true));
      }
    });
  };

  const openConvertDialog = (state: ConvertModalState): void => {
    if (!state.enquiryID) {
      return;
    }
    form.setAttribute("action", `/admin/enquiries/${state.enquiryID}/convert`);
    form.reset();
    search.value = state.search ?? "";
    quickName.value = state.quickName ?? "";
    quickMobile.value = state.quickMobile ?? "";
    note.value = state.note ?? "";
    filterOptions();
    if (state.customerID) {
      select.value = state.customerID;
    }
    dialog.showModal();
  };

  document.querySelectorAll<HTMLButtonElement>(".open-convert-dialog").forEach((button) => {
    button.addEventListener("click", () => {
      const enquiryID = button.getAttribute("data-enquiry-id") ?? "";
      openConvertDialog({ enquiryID });
    });
  });

  cancel.addEventListener("click", () => {
    dialog.close();
  });

  search.addEventListener("input", filterOptions);

  if (form.getAttribute("data-open-on-load") === "1") {
    openConvertDialog({
      enquiryID: form.getAttribute("data-enquiry-id") ?? "",
      customerID: form.getAttribute("data-customer-id") ?? "",
      quickName: form.getAttribute("data-quick-customer-name") ?? "",
      quickMobile: form.getAttribute("data-quick-customer-mobile") ?? "",
      note: form.getAttribute("data-note") ?? "",
      search: form.getAttribute("data-customer-search") ?? "",
    });
  }
};

const initOrderModal = (): void => {
  const dialog = document.getElementById("order-dialog") as HTMLDialogElement | null;
  const form = document.getElementById("order-form") as HTMLFormElement | null;
  const cancel = document.getElementById("cancel-order") as HTMLButtonElement | null;
  const customerName = document.getElementById("order-customer-name") as HTMLInputElement | null;
  const customerMobile = document.getElementById("order-customer-mobile") as HTMLInputElement | null;
  const customerNameHidden = document.getElementById("order-customer-name-hidden") as HTMLInputElement | null;
  const customerMobileHidden = document.getElementById("order-customer-mobile-hidden") as HTMLInputElement | null;
  const hasAddressHidden = document.getElementById("order-customer-has-address-hidden") as HTMLInputElement | null;
  const amount = document.getElementById("order-amount") as HTMLInputElement | null;
  const note = document.getElementById("order-note") as HTMLTextAreaElement | null;
  const addressField = document.getElementById("order-address-field") as HTMLDivElement | null;
  const address = document.getElementById("order-address") as HTMLTextAreaElement | null;

  if (
    !dialog ||
    !form ||
    !cancel ||
    !customerName ||
    !customerMobile ||
    !customerNameHidden ||
    !customerMobileHidden ||
    !hasAddressHidden ||
    !amount ||
    !note ||
    !addressField ||
    !address
  ) {
    return;
  }

  const setAddressFieldState = (hasAddress: boolean, requireAddress: boolean): void => {
    const shouldRequire = !hasAddress || requireAddress;
    if (shouldRequire) {
      addressField.classList.remove("hidden");
      address.required = true;
    } else {
      addressField.classList.add("hidden");
      address.required = false;
      address.value = "";
    }
  };

  const openOrderDialog = (state: OrderModalState): void => {
    if (!state.enquiryID) {
      return;
    }
    form.setAttribute("action", `/admin/enquiries/${state.enquiryID}/order`);
    form.reset();

    customerName.value = state.customerName;
    customerMobile.value = state.customerMobile;
    customerNameHidden.value = state.customerName;
    customerMobileHidden.value = state.customerMobile;
    hasAddressHidden.value = state.hasAddress ? "1" : "0";
    amount.value = state.amount ?? "";
    note.value = state.note ?? "";
    address.value = state.address ?? "";

    setAddressFieldState(state.hasAddress, state.requireAddress ?? false);
    dialog.showModal();
  };

  document.querySelectorAll<HTMLButtonElement>(".open-order-dialog").forEach((button) => {
    button.addEventListener("click", () => {
      openOrderDialog({
        enquiryID: button.getAttribute("data-enquiry-id") ?? "",
        customerName: button.getAttribute("data-customer-name") ?? "",
        customerMobile: button.getAttribute("data-customer-mobile") ?? "",
        hasAddress: (button.getAttribute("data-has-address") ?? "0") === "1",
      });
    });
  });

  cancel.addEventListener("click", () => {
    dialog.close();
  });

  if (form.getAttribute("data-open-on-load") === "1") {
    openOrderDialog({
      enquiryID: form.getAttribute("data-enquiry-id") ?? "",
      customerName: form.getAttribute("data-customer-name") ?? "",
      customerMobile: form.getAttribute("data-customer-mobile") ?? "",
      hasAddress: (form.getAttribute("data-has-address") ?? "0") === "1",
      requireAddress: (form.getAttribute("data-require-address") ?? "0") === "1",
      amount: form.getAttribute("data-order-amount") ?? "",
      note: form.getAttribute("data-note") ?? "",
      address: form.getAttribute("data-address") ?? "",
    });
  }
};

initConvertModal();
initOrderModal();
