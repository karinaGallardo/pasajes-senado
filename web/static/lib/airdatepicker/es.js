(function () {
  var es = {
    days: ["Domingo", "Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado"],
    daysShort: ["Dom", "Lun", "Mar", "Mie", "Jue", "Vie", "Sab"],
    daysMin: ["Do", "Lu", "Ma", "Mi", "Ju", "Vi", "Sa"],
    months: [
      "Enero",
      "Febrero",
      "Marzo",
      "Abril",
      "Mayo",
      "Junio",
      "Julio",
      "Agosto",
      "Septiembre",
      "Octubre",
      "Noviembre",
      "Diciembre",
    ],
    monthsShort: ["Ene", "Feb", "Mar", "Abr", "May", "Jun", "Jul", "Ago", "Sep", "Oct", "Nov", "Dic"],
    today: "Hoy",
    clear: "Limpiar",
    dateFormat: "dd/MM/yyyy",
    timeFormat: "hh:mm aa",
    firstDay: 1,
  };

  // Registrar en el objeto global de AirDatepicker si existe
  if (typeof AirDatepicker !== "undefined" && AirDatepicker.locale) {
    AirDatepicker.locale.es = es;
  }

  // Registrar también la variable global que la directiva de Alpine está buscando
  window.airDatepickerLocaleEs = es;
})();

/* 
!(function (e) {
  e.fn.datepicker.language.es = {
    days: ["Domingo", "Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado"],
    daysShort: ["Dom", "Lun", "Mar", "Mie", "Jue", "Vie", "Sab"],
    daysMin: ["Do", "Lu", "Ma", "Mi", "Ju", "Vi", "Sa"],
    months: [
      "Enero",
      "Febrero",
      "Marzo",
      "Abril",
      "Mayo",
      "Junio",
      "Julio",
      "Augosto",
      "Septiembre",
      "Octubre",
      "Noviembre",
      "Diciembre",
    ],
    monthsShort: ["Ene", "Feb", "Mar", "Abr", "May", "Jun", "Jul", "Ago", "Sep", "Oct", "Nov", "Dic"],
    today: "Hoy",
    clear: "Limpiar",
    dateFormat: "dd/mm/yyyy",
    timeFormat: "hh:ii aa",
    firstDay: 1,
  };
})(jQuery);
//# sourceMappingURL=datepicker.es.min.js.map
 */
