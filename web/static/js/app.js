document.addEventListener("alpine:init", function () {
  Alpine.data("searchableSelect", function (config) {
    return {
      open: false,
      search: "",
      disabled: config.disabled || false,
      value: config.initialValue || "",
      label: config.initialLabel || config.placeholder || "Seleccione una opción",
      placeholder: config.placeholder || "Seleccione una opción",
      items: config.items || [],

      get filteredItems() {
        if (this.search === "") return this.items;
        const q = this.search.toLowerCase();
        return this.items.filter(function (i) {
          return (i.label && i.label.toLowerCase().includes(q)) || (i.value && i.value.toLowerCase().includes(q)) || (i.extra && i.extra.toLowerCase().includes(q));
        });
      },

      select: function (item) {
        this.value = item ? item.value : "";
        this.label = item ? item.label : this.placeholder;
        this.open = false;
        this.search = "";
      },
    };
  });

  Alpine.directive("flatpickr", (el, { expression }, { evaluate }) => {
    let options = {};
    if (typeof expression === "string" && expression.trim() !== "") {
      options = evaluate(expression) || {};
    }
    const defaultOptions = {
      locale: "es",
      dateFormat: "Y-m-d H:i", // Keep server format in 24h
      enableTime: true,
      time_24hr: false, // Enable AM/PM toggle
      altInput: true,
      altFormat: "d/m/Y h:i K", // Formato estándar Bolivia: 01/03/2026 02:30 PM
      allowInput: true,
      static: true, // Crucial for components inside modals
      position: "auto right",
      plugins: [
        new confirmDatePlugin({
          confirmText: "Aceptar",
          showAlways: false,
          theme: "light",
        }),
      ],
    };

    const fp = flatpickr(el, {
      ...defaultOptions,
      ...options,
      onChange: (selectedDates, dateStr) => {
        el.value = dateStr;
        el.dispatchEvent(new Event("input", { bubbles: true }));
        el.dispatchEvent(new Event("change", { bubbles: true }));
      },
    });

    // Watch for external value changes
    el._fp_cleanup = Alpine.effect(() => {
      const val = el.getAttribute("value") || el.value;
      if (val && fp.input.value !== val) {
        fp.setDate(val, false);
      }
    });
  });

  Alpine.directive("tooltip", (el, { expression, modifiers }) => {
    let placement = "top";
    if (modifiers.includes("left")) placement = "left";
    if (modifiers.includes("right")) placement = "right";
    if (modifiers.includes("bottom")) placement = "bottom";

    tippy(el, {
      content: expression,
      allowHTML: true,
      animation: "shift-away",
      theme: "translucent",
      placement: placement,
      interactive: true,
    });
  });
});

/**
 * HTMX Confirmation with SweetAlert2
 */
document.addEventListener("htmx:confirm", function (e) {
  if (!e.detail.question) return;

  e.preventDefault();

  Swal.fire({
    title: "¿Estás seguro?",
    text: e.detail.question,
    icon: "warning",
    showCancelButton: true,
    confirmButtonColor: "#03738C",
    cancelButtonColor: "#ef4444",
    confirmButtonText: "Sí, proceder",
    cancelButtonText: "Cancelar",
    reverseButtons: true,
    customClass: {
      popup: "rounded-xl font-sans",
      title: "text-lg font-bold text-neutral-900",
      htmlContainer: "text-sm text-neutral-600",
      confirmButton: "rounded-lg px-4 py-2 text-sm font-medium shadow-sm transition-all duration-200",
      cancelButton: "rounded-lg px-4 py-2 text-sm font-medium shadow-sm transition-all duration-200",
    },
  }).then(function (result) {
    if (result.isConfirmed) {
      e.detail.issueRequest(true);
    }
  });
});

/**
 * Global event listener for showAlert trigger
 */
document.addEventListener("showAlert", function (evt) {
  const data = evt.detail;
  Swal.fire({
    icon: data.icon || "info",
    title: data.title || "",
    text: data.text || "",
    confirmButtonColor: "#03738C",
    customClass: {
      popup: "rounded-xl font-sans",
      confirmButton: "rounded-lg px-4 py-2 text-sm font-medium shadow-sm",
    },
  });
});
