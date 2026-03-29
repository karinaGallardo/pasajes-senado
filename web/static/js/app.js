document.addEventListener("alpine:init", function () {
  /**
   * Global data for Pasaje Modals (Create/Edit)
   */
  Alpine.data("modalPasaje", function (config) {
    return {
      open: true,
      id: config.id || "",
      ruta: config.ruta || "",
      agencia_id: config.agencia_id || "",
      aerolinea_id: config.aerolinea_id || "",
      numero_vuelo: config.numero_vuelo || "",
      fecha_vuelo: config.fecha_vuelo || "",
      fecha_emision: config.fecha_emision || "",
      numero_boleto: config.numero_boleto || "",
      costo: config.costo || "",
      glosa: config.glosa || "",
      codigo_reserva: config.codigo_reserva || "",
      numero_factura: config.numero_factura || "",
      fileName: "",
      loading: false,
      processingImage: false,
      fares: config.fares || {},

      init() {
        console.log("Modal Pasaje Initialized:", this.id ? "Edit" : "Create");
      },

      get tarifaReferencial() {
        if (this.ruta && this.aerolinea_id && this.fares[this.ruta]) {
          return this.fares[this.ruta][this.aerolinea_id] || 0;
        }
        return 0;
      },

      async handleFile(el) {
        if (!el.files || el.files.length === 0) return;

        const file = el.files[0];
        if (!window.checkFileSize(file)) {
          el.value = "";
          this.fileName = "";
          return;
        }
        if (file.type.startsWith("image/")) {
          this.processingImage = true;
          try {
            const resizedFile = await window.resizeImage(file);
            const dataTransfer = new DataTransfer();
            dataTransfer.items.add(resizedFile);
            el.files = dataTransfer.files;
            this.fileName = resizedFile.name;
          } catch (err) {
            console.error("Error resizing image:", err);
            this.fileName = file.name;
          } finally {
            this.processingImage = false;
          }
        } else {
          this.fileName = file.name;
        }
      },

      get canSave() {
        if (this.loading || this.processingImage) return false;
        // If editing, file is not mandatory. If creating, it is.
        return this.id ? true : !!this.fileName;
      },
    };
  });

  Alpine.data("searchableSelect", function (config) {
    return {
      open: false,
      search: "",
      loading: false,
      disabled: config.disabled || false,
      value: config.initialValue || "",
      label: config.initialLabel || config.placeholder || "Seleccione una opción",
      placeholder: config.placeholder || "Seleccione una opción",
      items: config.items || [],
      endpoint: config.endpoint || null,
      abortController: null,

      init() {
        if (this.endpoint) {
          this.$watch("search", (val) => {
            if (val.length >= 3) {
              this.fetchResults();
            } else if (val.length === 0) {
              this.cancelFetch();
              this.items = [];
            }
          });
        }
      },

      cancelFetch() {
        if (this.abortController) {
          this.abortController.abort();
          this.abortController = null;
        }
      },

      fetchResults: async function () {
        this.cancelFetch();
        this.abortController = new AbortController();
        this.loading = true;

        try {
          const res = await fetch(`${this.endpoint}?q=${encodeURIComponent(this.search)}`, {
            signal: this.abortController.signal,
          });
          if (!res.ok) throw new Error("Server error");
          this.items = await res.json();
        } catch (e) {
          if (e.name !== "AbortError") {
            console.error("Error fetching searchable items:", e);
          }
        } finally {
          this.loading = false;
        }
      },

      get filteredItems() {
        if (this.endpoint) return this.items || [];
        if (this.search === "") return this.items || [];
        const q = this.search.toLowerCase();
        return (this.items || []).filter((i) => {
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

  Alpine.data("multiDestinoSelect", function (config) {
    return {
      search: "",
      loading: false,
      allDestinos: config.allDestinos || [],
      selected: config.selected || [],
      open: false,
      endpoint: config.endpoint || null,
      abortController: null,

      init() {
        if (this.endpoint) {
          this.$watch("search", (val) => {
            if (val.length >= 3) {
              this.fetchResults();
            } else if (val.length === 0) {
              this.cancelFetch();
              this.allDestinos = [];
            }
          });
        }
      },

      cancelFetch() {
        if (this.abortController) {
          this.abortController.abort();
          this.abortController = null;
        }
      },

      fetchResults: async function () {
        this.cancelFetch();
        this.abortController = new AbortController();
        this.loading = true;

        try {
          const res = await fetch(`${this.endpoint}?q=${encodeURIComponent(this.search)}`, {
            signal: this.abortController.signal,
          });
          if (!res.ok) throw new Error("Server error");
          const data = await res.json();

          this.allDestinos = data.map((d) => ({
            iata: d.value,
            ciudad: d.label,
          }));
        } catch (e) {
          if (e.name !== "AbortError") {
            console.error("Error fetching multi destinations:", e);
          }
        } finally {
          this.loading = false;
        }
      },

      get filtered() {
        const list = this.allDestinos || [];
        if (this.endpoint) {
          return list.filter((d) => {
            const diata = (d.iata || "").trim().toUpperCase();
            return !this.selected.some((item) => (item.iata || "").trim().toUpperCase() === diata);
          });
        }

        if (!this.search) return list.slice(0, 10);
        const s = this.search.toLowerCase();
        return list
          .filter((d) => (d.ciudad || "").toLowerCase().includes(s) || (d.iata || "").toLowerCase().includes(s))
          .filter((d) => !this.selected.some((item) => (item.iata || "").trim().toUpperCase() === (d.iata || "").trim().toUpperCase()))
          .slice(0, 15);
      },

      add(dest) {
        this.selected.push(dest);
        this.search = "";
        this.open = false;
      },

      remove(iata) {
        this.selected = this.selected.filter((item) => {
          return (item.iata || "").trim().toUpperCase() !== (iata || "").trim().toUpperCase();
        });
      },
    };
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
