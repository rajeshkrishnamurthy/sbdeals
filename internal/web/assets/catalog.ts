type CatalogItem = {
  id: number;
  type: string;
  title: string;
  imageUrl?: string;
  currentPriceText: string;
  originalPriceText?: string;
  discountText?: string;
  reserveButtonLabel: string;
  whatsAppMessage: string;
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

type ClickedCreatePayload = {
  itemId: number;
  itemType: string;
  itemTitle: string;
  sourcePage: string;
  sourceRailId: number;
  sourceRailTitle: string;
};

const root = document.getElementById("catalog-root") as HTMLDivElement | null;
const toast = document.getElementById("catalog-toast") as HTMLDivElement | null;
const WHATSAPP_PHONE = "918951395971";
const CTA_TOAST_MESSAGE = "Connecting to WhatsApp...";
const CTA_DEBOUNCE_MS = 1200;
const ctaDebounceUntil = new Map<string, number>();

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

const createWhatsAppIcon = (): SVGElement => {
  const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  svg.setAttribute("viewBox", "0 0 24 24");
  svg.setAttribute("class", "cta-icon");
  svg.setAttribute("aria-hidden", "true");
  const path = document.createElementNS("http://www.w3.org/2000/svg", "path");
  path.setAttribute("d", "M12 2.25c-5.385 0-9.75 4.21-9.75 9.403 0 1.82.542 3.511 1.478 4.942L2.25 21.75l5.374-1.43A9.93 9.93 0 0 0 12 21.05c5.385 0 9.75-4.21 9.75-9.397C21.75 6.46 17.385 2.25 12 2.25Zm0 17.15a7.98 7.98 0 0 1-3.971-1.05l-.285-.165-3.188.845.857-3.04-.188-.3a7.2 7.2 0 0 1-1.125-3.89c0-3.99 3.496-7.234 7.9-7.234 4.404 0 7.9 3.244 7.9 7.234 0 3.99-3.496 7.234-7.9 7.234Zm3.497-5.39c-.19-.09-1.121-.544-1.295-.605-.173-.06-.3-.09-.427.09-.127.18-.49.605-.601.73-.11.12-.221.136-.41.045-.19-.09-.8-.28-1.523-.891-.563-.477-.943-1.066-1.053-1.246-.11-.18-.012-.277.083-.367.086-.081.19-.211.284-.317.095-.105.126-.18.19-.3.063-.12.031-.226-.016-.317-.047-.09-.427-1.022-.585-1.4-.154-.368-.311-.318-.427-.324l-.364-.006c-.126 0-.332.045-.506.225-.173.18-.664.647-.664 1.577s.68 1.83.775 1.956c.095.126 1.34 2.072 3.247 2.906.454.194.807.31 1.083.397.455.145.87.124 1.197.075.365-.054 1.122-.458 1.28-.899.157-.44.157-.817.11-.898-.047-.081-.173-.126-.364-.216Z");
  path.setAttribute("fill", "currentColor");
  svg.appendChild(path);
  return svg;
};

const reserveDebounceKey = (rail: CatalogRail, item: CatalogItem): string => `${rail.id}:${item.type}:${item.id}`;

const shouldSuppressCTA = (key: string): boolean => {
  const now = Date.now();
  const until = ctaDebounceUntil.get(key) ?? 0;
  if (until > now) {
    return true;
  }
  ctaDebounceUntil.set(key, now + CTA_DEBOUNCE_MS);
  return false;
};

const createClicked = async (payload: ClickedCreatePayload): Promise<void> => {
  const response = await fetch("/api/clicked", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
    keepalive: true,
  });
  if (!response.ok) {
    throw new Error("clicked create failed");
  }
};

const waitShort = (ms: number): Promise<void> =>
  new Promise((resolve) => {
    window.setTimeout(resolve, ms);
  });

const openWhatsApp = (message: string): void => {
  const url = `https://wa.me/${WHATSAPP_PHONE}?text=${encodeURIComponent(message)}`;
  window.open(url, "_blank", "noopener");
};

const createCard = (item: CatalogItem, rail: CatalogRail): HTMLElement => {
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
  const ctaContent = document.createElement("span");
  ctaContent.className = "cta-content";
  ctaContent.appendChild(createWhatsAppIcon());
  ctaContent.appendChild(appendText("span", "", item.reserveButtonLabel));
  cta.appendChild(ctaContent);
  cta.addEventListener("click", () => {
    const key = reserveDebounceKey(rail, item);
    if (shouldSuppressCTA(key)) {
      return;
    }

    showToast(CTA_TOAST_MESSAGE);
    const payload: ClickedCreatePayload = {
      itemId: item.id,
      itemType: item.type,
      itemTitle: item.title,
      sourcePage: "catalog",
      sourceRailId: rail.id,
      sourceRailTitle: rail.title,
    };
    const clickedPromise = createClicked(payload).catch(() => undefined);
    void Promise.race([clickedPromise, waitShort(250)]).finally(() => {
      openWhatsApp(item.whatsAppMessage);
    });
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
    row.appendChild(createCard(item, rail));
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
