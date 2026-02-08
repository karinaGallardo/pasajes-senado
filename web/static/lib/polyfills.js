/**
 * Polyfills básicos para compatibilidad con navegadores antiguos (Windows 7 / Chrome ~2020)
 * Incluye: Promise, fetch, Object.assign, closest, remove, forEach
 */

// 1. NodeList.prototype.forEach
if (window.NodeList && !NodeList.prototype.forEach) {
  NodeList.prototype.forEach = Array.prototype.forEach;
}

// 2. Element.prototype.closest
if (!Element.prototype.closest) {
  Element.prototype.closest = function (s) {
    var el = this;
    do {
      if (Element.prototype.matches.call(el, s)) return el;
      el = el.parentElement || el.parentNode;
    } while (el !== null && el.nodeType === 1);
    return null;
  };
}

// 3. Element.prototype.matches
if (!Element.prototype.matches) {
  Element.prototype.matches = Element.prototype.msMatchesSelector || Element.prototype.webkitMatchesSelector;
}

// 4. Element.prototype.remove
(function (arr) {
  arr.forEach(function (item) {
    if (Object.prototype.hasOwnProperty.call(item, "remove")) {
      return;
    }
    Object.defineProperty(item, "remove", {
      configurable: true,
      enumerable: true,
      writable: true,
      value: function remove() {
        if (this.parentNode === null) {
          return;
        }
        this.parentNode.removeChild(this);
      },
    });
  });
})([Element.prototype, CharacterData.prototype, DocumentType.prototype]);

// 5. Object.assign
if (typeof Object.assign !== "function") {
  Object.defineProperty(Object, "assign", {
    value: function assign(target, varArgs) {
      "use strict";
      if (target === null || target === undefined) {
        throw new TypeError("Cannot convert undefined or null to object");
      }
      var to = Object(target);
      for (var index = 1; index < arguments.length; index++) {
        var nextSource = arguments[index];
        if (nextSource !== null && nextSource !== undefined) {
          for (var nextKey in nextSource) {
            if (Object.prototype.hasOwnProperty.call(nextSource, nextKey)) {
              to[nextKey] = nextSource[nextKey];
            }
          }
        }
      }
      return to;
    },
    writable: true,
    configurable: true,
  });
}

// 6. AbortController (mínimo para fetch si es necesario, básico)
if (typeof window.AbortController === "undefined") {
  window.AbortController = function () {
    this.signal = { aborted: false };
    this.abort = function () {
      this.signal.aborted = true;
    };
  };
}
