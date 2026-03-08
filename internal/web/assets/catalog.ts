type CatalogItem = {
  id: number;
  type: string;
  title: string;
  imageUrl?: string;
  currentPriceText: string;
  originalPriceText?: string;
  discountText?: string;
  reserveButtonLabel: string;
};

type CatalogRail = {
  id: number;
  title: string;
  type: string;
  items: CatalogItem[];
};

type CatalogResponse = {
  rails: CatalogRail[];
};

const root = document.getElementById("catalog-root") as HTMLDivElement | null;
const toast = document.getElementById("catalog-toast") as HTMLDivElement | null;

const setRootState = (className: string): void => {
  if (!root) {
    return;
  }
  root.className = className;
};

const clearNode = (node: HTMLElement): void => {
  while (node.firstChild) {
    node.removeChild(node.firstChild);
  }
};

const showToast = (message: string): void => {
  if (!toast) {
    return;
  }
  toast.textContent = message;
  toast.classList.add("visible");
  window.setTimeout(() => {
    toast.classList.remove("visible");
  }, 1800);
};

const appendText = (tagName: string, className: string, text: string): HTMLElement => {
  const node = document.createElement(tagName);
  node.className = className;
  node.textContent = text;
  return node;
};

const createCard = (item: CatalogItem): HTMLElement => {
  const article = document.createElement("article");
  article.className = "catalog-card";

  const media = document.createElement("div");
  media.className = "catalog-card-media";
  if (item.imageUrl) {
    const image = document.createElement("img");
    image.src = item.imageUrl;
    image.alt = item.title;
    media.appendChild(image);
  } else {
    media.appendChild(appendText("span", "catalog-card-placeholder", "No image"));
  }
  article.appendChild(media);

  article.appendChild(appendText("h3", "catalog-card-title", item.title));

  const pricing = document.createElement("div");
  pricing.className = "catalog-price";
  pricing.appendChild(appendText("span", "catalog-price-current", item.currentPriceText));
  if (item.originalPriceText) {
    pricing.appendChild(appendText("span", "catalog-price-original", item.originalPriceText));
  }
  if (item.discountText) {
    pricing.appendChild(appendText("span", "catalog-price-discount", item.discountText));
  }
  article.appendChild(pricing);

  const cta = document.createElement("button");
  cta.className = "cta";
  cta.type = "button";
  cta.textContent = item.reserveButtonLabel;
  cta.addEventListener("click", () => {
    showToast("Coming soon");
  });
  article.appendChild(cta);

  return article;
};

const createRail = (rail: CatalogRail): HTMLElement => {
  const section = document.createElement("section");
  section.className = "rail-section";

  const header = document.createElement("div");
  header.className = "rail-header";
  header.appendChild(appendText("h2", "rail-title", rail.title));
  header.appendChild(appendText("span", "rail-kind", rail.type));
  section.appendChild(header);

  if (rail.items.length === 0) {
    const empty = document.createElement("div");
    empty.className = "rail-empty";
    empty.textContent = "Items for this rail will appear here when active listings are available.";
    section.appendChild(empty);
    return section;
  }

  const row = document.createElement("div");
  row.className = "rail-row";
  for (const item of rail.items) {
    row.appendChild(createCard(item));
  }
  section.appendChild(row);
  return section;
};

const renderRails = (payload: CatalogResponse): void => {
  if (!root) {
    return;
  }
  clearNode(root);
  setRootState("rail-list");

  for (const rail of payload.rails) {
    root.appendChild(createRail(rail));
  }
};

const renderError = (): void => {
  if (!root) {
    return;
  }
  clearNode(root);
  setRootState("catalog-error");

  root.appendChild(appendText("p", "", "Catalog could not be loaded right now."));
  const retry = document.createElement("button");
  retry.type = "button";
  retry.textContent = "Retry";
  retry.addEventListener("click", () => {
    void loadCatalog();
  });
  root.appendChild(retry);
};

const loadCatalog = async (): Promise<void> => {
  if (!root) {
    return;
  }
  clearNode(root);
  setRootState("catalog-loading");
  root.appendChild(appendText("p", "", "Loading curated rails..."));

  try {
    const response = await fetch(root.dataset.endpoint ?? "", {
      headers: { Accept: "application/json" },
    });
    if (!response.ok) {
      throw new Error("catalog fetch failed");
    }
    const payload = (await response.json()) as CatalogResponse;
    renderRails(payload);
  } catch {
    renderError();
  }
};

if (root) {
  void loadCatalog();
}
