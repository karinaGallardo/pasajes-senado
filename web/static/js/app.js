document.addEventListener("alpine:init", function () {
  /**
   * Global data for Pasaje Modals (Create/Edit)
   */
  Alpine.data("modalPasaje", function (config) {
    return {
      open: true,
      id: config.id || "",
      ruta: config.ruta || "",
      ruta_id: config.ruta_id || "",
      agencia_id: config.agencia_id || "",
      aerolinea_id: config.aerolinea_id || "",
      numero_vuelo: config.numero_vuelo || "",
      fecha_vuelo: config.fecha_vuelo || "",
      fecha_emision: config.fecha_emision || "",
      numero_billete: config.numero_billete || "",
      costo: config.costo || "",
      glosa: config.glosa || "",
      codigo_reserva: config.codigo_reserva || "",
      numero_factura: config.numero_factura || "",
      fileName: "",
      loading: false,
      processingImage: false,
      fares: config.fares || {},

      parseDate(str) {
        if (!str) return new Date(NaN);
        // Standardize separator to T for ISO parsing if it's space
        return new Date(str.replace(" ", "T"));
      },

      init() {
        console.log("Modal Pasaje Initialized:", this.id ? "Edit" : "Create");
      },

      get tarifaReferencial() {
        if (this.ruta_id && this.aerolinea_id && this.fares[this.ruta_id]) {
          return this.fares[this.ruta_id][this.aerolinea_id] || 0;
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
      selectedLabel: config.initialLabel || "",
      placeholder: config.placeholder || "Seleccione una opción",
      items: config.items || [],
      endpoint: config.endpoint || null,
      getExtraParams: config.getExtraParams || null,
      abortController: null,

      init() {
        if (this.selectedLabel) {
          this.search = this.selectedLabel;
        }

        this.$watch("open", (isOpen) => {
          if (isOpen) {
            this.search = ""; // Clear for immediate typing
          } else {
            if (this.search === "" && this.selectedLabel) {
              this.search = this.selectedLabel;
            }
          }
        });

        if (this.endpoint) {
          this.$watch("search", (val) => {
            if (val.length >= 2) {
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
          let url = `${this.endpoint}${this.endpoint.includes("?") ? "&" : "?"}q=${encodeURIComponent(this.search)}`;

          if (this.getExtraParams && typeof this.getExtraParams === "function") {
            const extras = this.getExtraParams();
            Object.keys(extras).forEach((key) => {
              url += `&${key}=${encodeURIComponent(extras[key])}`;
            });
          }

          console.log("SearchableSelect Fetching:", url);
          const res = await fetch(url, {
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
          return (
            (i.label && i.label.toLowerCase().includes(q)) ||
            (i.value && i.value.toLowerCase().includes(q)) ||
            (i.extra && i.extra.toLowerCase().includes(q))
          );
        });
      },

      select(item) {
        this.selectedLabel = item.label;
        this.search = item.label; // Mostrar nombre legible en el input
        this.value = item ? item.value : "";
        this.label = item ? item.label : this.placeholder;
        this.open = false;

        // Ejecutar el callback de selección si existe
        if (config.onSelect && typeof config.onSelect === "function") {
          config.onSelect(this.value);
        }

        // Dispatch unique event to avoid collision with standard "change" events
        this.$el.dispatchEvent(new CustomEvent("searchable-select-change", { detail: item || {}, bubbles: true }));
      },
    };
  });

  /**
   * x-datepicker: Premium Calendar Directive using Air Datepicker v3
   */
  Alpine.directive("datepicker", (el, { expression }, { evaluate }) => {
    let options = {};
    if (typeof expression === "string" && expression.trim() !== "") {
      options = evaluate(expression) || {};
    }

    const defaultOptions = {
      locale: window.airDatepickerLocaleEs || "es", // "es" is loaded from CDN
      timepicker: true,
      timeFormat: "HH:mm",
      dateFormat: "yyyy-MM-dd",
      minutesStep: 5,
      autoClose: false,
      position: "bottom center",
      navTitles: {
        days: "<strong>MMMM</strong> <i>yyyy</i>",
      },
      prevHtml: '<i class="ph ph-caret-left"></i>',
      nextHtml: '<i class="ph ph-caret-right"></i>',
      buttons: [
        {
          content: "Hoy",
          className: "custom-button-today",
          onClick: (dp) => {
            const now = new Date();
            dp.setViewDate(now);
            dp.selectDate(now);
          },
        },
        "clear",
        {
          content: "Aceptar",
          className: "custom-button-apply",
          onClick: (dp) => {
            dp.hide();
          },
        },
      ],
      onSelect({ date, formattedDate, datepicker }) {
        // Sync back with input for HTMX/Alpine
        el.value = formattedDate;
        el.dispatchEvent(new Event("input", { bubbles: true }));
        el.dispatchEvent(new Event("change", { bubbles: true }));
      },
    };

    // Force readonly to prevent manual bypass and ensure data integrity
    el.readOnly = true;
    el.classList.add("cursor-pointer");

    const mergedOptions = { ...defaultOptions, ...options };

    // Auto-detect container to avoid scrolling issues in modals
    if (!mergedOptions.container) {
      mergedOptions.container = document.querySelector("#modal-container") || "body";
    }

    // Setup Air Datepicker
    const dp = new AirDatepicker(el, mergedOptions);

    // Clean up when element is removed
    el._dp_instance = dp;
    el._dp_cleanup = Alpine.effect(() => {
      const val = el.getAttribute("value") || el.value;
      if (val && dp.selectedDates.length === 0) {
        dp.selectDate(val, { silent: true });
      }
    });

    // Cleanup on destroy
    return () => {
      if (dp) dp.destroy();
    };
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
            if (val.length >= 2) {
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
          .filter(
            (d) => !this.selected.some((item) => (item.iata || "").trim().toUpperCase() === (d.iata || "").trim().toUpperCase()),
          )
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

  Alpine.data("itineraryRow", (config) => ({
    esDevolucion: config.esDevo,
    esModificacion: config.esMod,
    monto: config.monto || 0,
    moneda: config.moneda || "BOB",
    route_id: config.route_id || "",
    tramo_nombre: config.tramo_nombre || "",
    origen_iata: config.origen_iata || "",
    destino_iata: config.destino_iata || "",
    billete: config.billete || "",
    vuelo: config.vuelo || "",
    pase: config.pase || "",
    isFullTicket: false,
    fileName: "",
    localUrl: "",
    processingImage: false,

    init() {
      this.$watch("esDevolucion", (val) => {
        if (val) {
          this.esModificacion = false;
          this.isFullTicket = true;
          this.monto = 0; // Reset manual amount if full refund
        } else {
          this.isFullTicket = false;
        }
        this.$dispatch("update-totals", { billete: this.billete });
      });

      this.$watch("esModificacion", (val) => {
        if (val) {
          this.esDevolucion = false;
          // When modifying connection, reset isFullTicket
          this.isFullTicket = false;
          // Sync other scales of the same ticket to also clear their Devolucion status
          if (this.billete) {
            this.$dispatch("ticket-mod-changed", { billete: this.billete, state: true });
          }
        }
        this.$dispatch("update-totals", { billete: this.billete });
      });
    },

    checkFullStatus() {
      // Legacy or called from outside, just sync isFullTicket with esDevolucion
      this.isFullTicket = this.esDevolucion;
    },

    get bindFade() {
      return {
        "x-transition:enter": "transition ease-out duration-300",
        "x-transition:enter-start": "opacity-0 transform -translate-y-2",
        "x-transition:enter-end": "opacity-100 transform translate-y-0",
        ":class": "{ 'bg-warning-50/30': esModificacion, 'bg-danger-50/30': esDevolucion }",
      };
    },

    get bindInput() {
      return {
        ":readonly": "esDevolucion || esModificacion",
        ":class": "{ 'bg-neutral-50 text-neutral-400 cursor-not-allowed opacity-60': esDevolucion || esModificacion }",
      };
    },

    get bindVuelo() {
      return {
        // Only disable if it's an original segment being toggled for modification
        ":disabled": "esDevolucion",
        ":class": "{ 'bg-neutral-50 text-neutral-400 cursor-not-allowed opacity-60': esDevolucion }",
      };
    },
    get bindPase() {
      return {
        ":disabled": "esDevolucion",
        ":class": "{ 'bg-neutral-50 text-neutral-400 cursor-not-allowed opacity-60': esDevolucion }",
      };
    },

    get bindArchivo() {
      return {
        ":disabled": "esDevolucion || esModificacion",
        ":class": "{ 'bg-danger-50/10 cursor-not-allowed': esDevolucion, 'bg-warning-50/10 cursor-not-allowed': esModificacion }",
      };
    },
  }));

  Alpine.data("routePicker", (items = []) => ({
    open: false,
    searchOrig: "",
    searchDest: "",
    currentRowId: null,
    selectedOrig: null,
    selectedDest: null,
    items: (items || []).map((d) => ({
      value: d.iata || d.value,
      cityName: d.nombre || d.label || d.ciudad,
      label: (d.iata || d.value) + " - " + (d.nombre || d.label || d.ciudad),
    })),

    get filteredOrig() {
      if (!this.searchOrig) return this.items || [];
      const q = this.searchOrig.toLowerCase();
      return (this.items || []).filter(
        (i) => (i.label || "").toLowerCase().includes(q) || (i.cityName || "").toLowerCase().includes(q),
      );
    },

    get filteredDest() {
      if (!this.searchDest) return this.items || [];
      const q = this.searchDest.toLowerCase();
      return (this.items || []).filter(
        (i) => (i.label || "").toLowerCase().includes(q) || (i.cityName || "").toLowerCase().includes(q),
      );
    },

    openModal(rowId) {
      this.currentRowId = rowId;
      this.searchOrig = "";
      this.searchDest = "";
      this.selectedOrig = null;
      this.selectedDest = null;
      this.open = true;
    },

    confirm() {
      if (this.selectedOrig && this.selectedDest) {
        window.dispatchEvent(
          new CustomEvent("route-picked", {
            detail: {
              rowId: this.currentRowId,
              orig: this.selectedOrig,
              dest: this.selectedDest,
            },
          }),
        );
        this.open = false;
      }
    },
  }));

  /**
   * oficialFormHandler: Logic for Official Travel Request Modal
   */
  Alpine.data("oficialFormHandler", () => ({
    open: true,
    loading: false,
    tipo: "COMISION",
    ambito: "NACIONAL",
    motivo: "",
    autorizacion: "",
    aerolinea: "",
    tramos: [], // { id, tipo, origen, destino, fecha_salida }

    init() {
      console.log("OficialFormHandler Initialized");
    },

    addTramo(tipo) {
      let lastDest = "";
      if (this.tramos.length > 0) {
        lastDest = this.tramos[this.tramos.length - 1].destino;
      }

      this.tramos.push({
        id: String(Date.now() + Math.random()),
        tipo: tipo,
        origen: lastDest,
        destino: "",
        fecha_salida: "",
      });
    },

    removeTramo(id) {
      this.tramos = this.tramos.filter((t) => t.id !== id);
    },

    get isValid() {
      // Validar campos generales básicos
      if (!this.autorizacion || !this.tipo || !this.ambito) return false;

      // Validar que haya al menos un tramo y todos estén llenos
      if (this.tramos.length === 0) return false;
      return this.tramos.every((t) => t.origen && t.destino && t.fecha_salida);
    },

    async submitForm() {
      if (this.loading || !this.isValid) return;

      this.loading = true;

      try {
        if (this.$refs.mainForm) {
          this.$refs.mainForm.submit();
        } else {
          throw new Error("Referencia a formulario principal no encontrada");
        }
      } catch (e) {
        console.error("Submit Error:", e);
        this.loading = false;
        alert("Hubo un error al intentar enviar el formulario");
      }
    },
  }));
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
    text: data.text || data.message || "",
    confirmButtonColor: "#03738C",
    customClass: {
      popup: "rounded-xl font-sans",
      title: "text-lg font-bold text-neutral-900",
      htmlContainer: "text-sm text-neutral-600",
      confirmButton: "rounded-lg px-6 py-2.5 text-sm font-medium shadow-sm transition-all duration-200",
    },
  });
});

/**
 * Global HTMX Loading Handlers
 * Shows a loading state on buttons and prevents multiple submissions
 */
document.addEventListener("htmx:configRequest", (event) => {
  const element = event.detail.elt;
  // Solo para botones o enlaces que tengan hx-get/post/etc o sean submits de forms htmx
  if (element && (element.tagName === "BUTTON" || element.tagName === "A")) {
    // Evitar procesar botones de navegacion simple o que tienen x-ignore
    if (element.hasAttribute("x-ignore") || element.classList.contains("no-disable")) return;

    element.disabled = true;
    const originalContent = element.innerHTML;
    element.setAttribute("data-original-content", originalContent);

    // Si tiene icono, intentar mantener una estructura similar pero con spinner
    if (originalContent.includes("<i")) {
      element.innerHTML = '<i class="ph ph-circle-notch animate-spin"></i> <span class="ml-1">...</span>';
    } else {
      element.innerHTML = '<i class="ph ph-circle-notch animate-spin mr-2"></i>';
    }
    element.classList.add("opacity-75", "cursor-wait");
  }
});

document.addEventListener("htmx:afterRequest", (event) => {
  const element = event.detail.elt;
  if (element && (element.tagName === "BUTTON" || element.tagName === "A")) {
    element.disabled = false;
    element.classList.remove("opacity-75", "cursor-wait");
    const originalContent = element.getAttribute("data-original-content");
    if (originalContent) {
      element.innerHTML = originalContent;
    }
  }
});

document.addEventListener("htmx:beforeSend", (event) => {
  // Opcional: mostrar barra de progreso global NProgress si estuviera disponible
});

/**
 * Global Standard Form Submit Handlers
 * Similar to HTMX handlers but for normal POST forms
 */
document.addEventListener("submit", (event) => {
  const form = event.target;
  const submitButton = form.querySelector('button[type="submit"]');

  if (submitButton && !submitButton.hasAttribute("x-ignore") && !submitButton.classList.contains("no-disable")) {
    // Check if it's already being handled by HTMX to avoid double processing
    if (
      submitButton.hasAttribute("hx-post") ||
      submitButton.hasAttribute("hx-put") ||
      submitButton.hasAttribute("hx-get") ||
      form.hasAttribute("hx-post")
    ) {
      return;
    }

    submitButton.disabled = true;
    const originalContent = submitButton.innerHTML;
    submitButton.setAttribute("data-original-content", originalContent);

    if (originalContent.includes("<i")) {
      submitButton.innerHTML = '<i class="ph ph-circle-notch animate-spin"></i> <span class="ml-1">...</span>';
    } else {
      submitButton.innerHTML = '<i class="ph ph-circle-notch animate-spin mr-2"></i>';
    }
    submitButton.classList.add("opacity-75", "cursor-wait");

    // Optional: timeout to restore button if navigation takes too long or fails
    setTimeout(() => {
      if (submitButton.disabled) {
        submitButton.disabled = false;
        submitButton.classList.remove("opacity-75", "cursor-wait");
        if (submitButton.getAttribute("data-original-content")) {
          submitButton.innerHTML = submitButton.getAttribute("data-original-content");
        }
      }
    }, 10000); // 10s fallback
  }
});

/**
 * Redimensiona una imagen si es muy pesada o muy grande.
 */
window.resizeImage = async function (file, maxWidth = 1600, maxHeight = 1600, quality = 0.7) {
  return new Promise((resolve) => {
    if (!file || !file.type.startsWith("image/") || file.size < 2 * 1024 * 1024) {
      resolve(file);
      return;
    }
    const reader = new FileReader();
    reader.onload = (e) => {
      const img = new Image();
      img.onload = () => {
        let width = img.width;
        let height = img.height;
        if (width > height) {
          if (width > maxWidth) {
            height *= maxWidth / width;
            width = maxWidth;
          }
        } else {
          if (height > maxHeight) {
            width *= maxHeight / height;
            height = maxHeight;
          }
        }
        const canvas = document.createElement("canvas");
        canvas.width = width;
        canvas.height = height;
        const ctx = canvas.getContext("2d");
        ctx.drawImage(img, 0, 0, width, height);
        canvas.toBlob(
          (blob) => {
            let newName = file.name;
            const extension = newName.split(".").pop().toLowerCase();
            if (extension !== "jpg" && extension !== "jpeg") {
              newName = newName.replace(/\.[^/.]+$/, "") + ".jpg";
            }
            const resizedFile = new File([blob], newName, { type: "image/jpeg", lastModified: Date.now() });
            resolve(resizedFile);
          },
          "image/jpeg",
          quality,
        );
      };
      img.onerror = () => resolve(file);
      img.src = e.target.result;
    };
    reader.onerror = () => resolve(file);
    reader.readAsDataURL(file);
  });
};

/**
 * Valida si el archivo cumple con el límite de tamaño.
 */
window.checkFileSize = function (file, maxSizeMB = 8) {
  if (!file) return true;
  const maxSizeBytes = maxSizeMB * 1024 * 1024;
  if (file.size > maxSizeBytes) {
    if (typeof Swal !== "undefined") {
      Swal.fire({
        icon: "warning",
        title: "Archivo demasiado pesado",
        text: `El archivo supera el límite permitido de ${maxSizeMB}MB.`,
        confirmButtonColor: "#3085d6",
      });
    } else {
      alert(`El archivo supera el límite permitido de ${maxSizeMB}MB.`);
    }
    return false;
  }
  return true;
};

/**
 * Valida y procesa un archivo (redimensiona si es imagen).
 */
window.validateAndProcessFile = async function (file, maxSizeMB = 8) {
  if (!window.checkFileSize(file, maxSizeMB)) return null;

  if (file.type.startsWith("image/")) {
    try {
      return await window.resizeImage(file);
    } catch (err) {
      console.error("Error redimensionando:", err);
      return file;
    }
  }
  return file;
};

/**
 * Manejador global para una sola carga de archivo (ej. Pases de abordar).
 * Soporta actualización de Alpine si se pasa el objeto de datos.
 */
window.handleSingleUpload = async function (event, alpineData = null) {
  const input = event.target;
  if (!input.files || input.files.length === 0) return;

  const file = input.files[0];
  if (alpineData) alpineData.processingImage = true;

  try {
    const processed = await window.validateAndProcessFile(file);
    if (!processed) {
      input.value = "";
      if (alpineData) alpineData.fileName = "";
      return;
    }

    const dataTransfer = new DataTransfer();
    dataTransfer.items.add(processed);
    input.files = dataTransfer.files;

    if (alpineData) {
      alpineData.fileName = processed.name;
    }

    // --- NUEVO: Subida automática al servidor ---
    const formData = new FormData();
    formData.append("file", processed);
    // Intentar sacar el token CSRF si existe en el documento
    const csrfToken = document.querySelector('input[name="_csrf"]')?.value;
    if (csrfToken) formData.append("_csrf", csrfToken);

    const response = await fetch("/uploads/single", {
      method: "POST",
      body: formData,
    });

    if (response.ok) {
      const data = await response.json();
      if (data.path) {
        // Buscar el input hidden correspondiente para guardar la ruta del server
        // El input hidden debe tener nombre 'tramo_archivo_existente_{{index}}' o similar
        // Intentamos encontrarlo por nombre basándonos en el nombre del input file
        const indexMatch = input.name.match(/tramo_archivo_(.+)/);
        if (indexMatch && indexMatch[1]) {
          const index = indexMatch[1];
          const hiddenInput = document.querySelector(`input[name="tramo_archivo_existente_${index}"]`);
          if (hiddenInput) {
            hiddenInput.value = data.path;
            console.log(`Archivo subido y vinculado: ${data.path}`);
          }
        }
      }
    } else {
      console.error("Error en la subida automática:", response.statusText);
    }
    // ---------------------------------------------
  } finally {
    if (alpineData) alpineData.processingImage = false;
  }
};

/**
 * Manejador global para carga múltiple de archivos (ej. Anexos).
 */
window.handleMultipleUpload = async function (event, alpineData = null) {
  const input = event.target;
  if (!input.files || input.files.length === 0) return;

  if (alpineData) alpineData.processingImage = true;

  try {
    const dataTransfer = new DataTransfer();
    for (let i = 0; i < input.files.length; i++) {
      const file = input.files[i];
      const processed = await window.validateAndProcessFile(file);
      if (processed) {
        dataTransfer.items.add(processed);
      }
    }
    input.files = dataTransfer.files;
    // Disparar evento para que Alpine/HTMX detecten el cambio
    input.dispatchEvent(new Event("change", { bubbles: true }));
  } finally {
    if (alpineData) alpineData.processingImage = false;
  }
};
