const coverInput = document.getElementById("cover");
const coverPreview = document.getElementById("book-cover-preview");
const coverPlaceholder = document.getElementById("book-cover-placeholder");
const mrpInput = document.getElementById("mrp");
const myPriceInput = document.getElementById("my_price");
const discountInput = document.getElementById("discount");

const discountText = (mrpRaw, myPriceRaw) => {
  const mrp = Number.parseFloat(mrpRaw);
  const myPrice = Number.parseFloat(myPriceRaw);
  if (!Number.isFinite(mrp) || !Number.isFinite(myPrice) || mrp <= 0) {
    return "0%";
  }
  const discount = ((mrp - myPrice) / mrp) * 100;
  return `${Math.round(discount)}%`;
};

const updateDiscount = () => {
  if (!discountInput || !mrpInput || !myPriceInput) {
    return;
  }
  discountInput.value = discountText(mrpInput.value, myPriceInput.value);
};

if (mrpInput && myPriceInput && discountInput) {
  mrpInput.addEventListener("input", updateDiscount);
  myPriceInput.addEventListener("input", updateDiscount);
  updateDiscount();
}

if (coverInput && coverPreview && coverPlaceholder) {
  coverInput.addEventListener("change", () => {
    const file = coverInput.files && coverInput.files[0];
    if (!file) {
      return;
    }
    const objectURL = URL.createObjectURL(file);
    coverPreview.src = objectURL;
    coverPreview.classList.remove("hidden");
    coverPlaceholder.classList.add("hidden");
  });
}
