(function () {
  "use strict";

  // 1. Mobile drawer toggle & close
  function setDrawerOpen(open) {
    var drawer = document.getElementById("mobile-drawer");
    var backdrop = document.getElementById("mobile-drawer-backdrop");
    var toggles = document.querySelectorAll("[data-drawer-toggle]");
    if (!drawer || !backdrop) return;
    if (open) {
      drawer.removeAttribute("hidden");
      backdrop.removeAttribute("hidden");
    } else {
      drawer.setAttribute("hidden", "");
      backdrop.setAttribute("hidden", "");
    }
    toggles.forEach(function (btn) {
      btn.setAttribute("aria-expanded", open ? "true" : "false");
    });
  }

  // 2. Modal open & close
  function openModal(selector) {
    var modal = document.querySelector(selector);
    if (!modal) return;
    modal.removeAttribute("hidden");
    initQuizWizard(modal);
  }

  function closeModal(modal) {
    if (!modal) return;
    modal.setAttribute("hidden", "");
    if (modal._quizTimer) {
      clearInterval(modal._quizTimer);
      modal._quizTimer = null;
    }
  }

  // 3. Daily Quiz one-way step wizard + 45s timer per question
  function initQuizWizard(root) {
    var form = root.querySelector(".quiz-form");
    if (!form) return;
    var steps = Array.prototype.slice.call(form.querySelectorAll("[data-quiz-step]"));
    if (!steps.length) return;

    var current = 0;
    var remaining = 45;
    var progressEl = form.querySelector("[data-quiz-progress]");
    var barEl = form.querySelector("[data-quiz-progress-bar]");
    var timerEl = form.querySelector("[data-quiz-timer]");
    var nextBtn = form.querySelector("[data-quiz-next]");
    var submitBtn = form.querySelector("[data-quiz-submit]");

    function renderStep() {
      steps.forEach(function (step, idx) {
        if (idx === current) {
          step.classList.add("is-active");
          step.removeAttribute("hidden");
        } else {
          step.classList.remove("is-active");
          step.setAttribute("hidden", "");
        }
      });
      if (progressEl) {
        progressEl.textContent = "Soal " + (current + 1) + " dari " + steps.length;
      }
      if (barEl) {
        var pct = Math.round(((current + 1) / steps.length) * 100);
        barEl.style.width = pct + "%";
      }
      if (nextBtn && submitBtn) {
        if (current >= steps.length - 1) {
          nextBtn.setAttribute("hidden", "");
          submitBtn.removeAttribute("hidden");
        } else {
          nextBtn.removeAttribute("hidden");
          submitBtn.setAttribute("hidden", "");
        }
      }
      remaining = 45;
      if (timerEl) timerEl.textContent = remaining + "s";
    }

    if (root._quizTimer) clearInterval(root._quizTimer);
    root._quizTimer = setInterval(function () {
      if (root.hasAttribute("hidden") || !document.body.contains(form)) {
        clearInterval(root._quizTimer);
        return;
      }
      remaining -= 1;
      if (timerEl) timerEl.textContent = Math.max(0, remaining) + "s";
      if (remaining <= 0) {
        if (current < steps.length - 1) {
          current += 1;
          renderStep();
        } else {
          clearInterval(root._quizTimer);
          if (form.requestSubmit) form.requestSubmit();
          else form.submit();
        }
      }
    }, 1000);

    if (nextBtn && !nextBtn._bound) {
      nextBtn._bound = true;
      nextBtn.addEventListener("click", function () {
        if (current < steps.length - 1) {
          current += 1;
          renderStep();
        }
      });
    }

    renderStep();
  }

  // 4. CBT Exam Kiosk question navigator + anti-cheat focus guard & fullscreen
  function initCbtKiosk() {
    var kiosk = document.querySelector("[data-cbt-kiosk]");
    if (!kiosk || kiosk._bound) return;
    kiosk._bound = true;

    // Enter kiosk isolation mode (hides ERP sidebar and topbar)
    document.body.classList.add("is-cbt-active");

    var steps = Array.prototype.slice.call(kiosk.querySelectorAll("[data-cbt-question]"));
    var prevBtn = kiosk.querySelector("[data-cbt-prev]");
    var nextBtn = kiosk.querySelector("[data-cbt-next]");
    var timerEl = kiosk.querySelector("[data-cbt-timer]");
    var fsGate = document.querySelector("[data-cbt-fullscreen-gate]");
    var enterFsBtn = document.querySelector("[data-cbt-enter-fullscreen]");
    var reenterFsBtn = document.querySelector("[data-cbt-reenter-fullscreen]");
    var questionsModal = document.querySelector("[data-cbt-questions-dialog]");
    var toggleQuestionsBtn = kiosk.querySelector("[data-cbt-toggle-questions]");
    var closeQuestionsBtns = Array.prototype.slice.call(document.querySelectorAll("[data-cbt-close-questions]"));
    var warnForm = document.getElementById("cbt-violation-form");
    var reasonInput = warnForm ? warnForm.querySelector("input[name='reason']") : null;
    var currentIdx = 0;
    var violationTriggered = false;

    // Helper: detect if browser is currently in fullscreen
    function isFullscreen() {
      return Boolean(
        document.fullscreenElement ||
        document.webkitFullscreenElement ||
        document.mozFullScreenElement ||
        document.msFullscreenElement
      );
    }

    // Helper: request fullscreen mode
    function enterFullscreen() {
      var docEl = document.documentElement;
      var req =
        docEl.requestFullscreen ||
        docEl.webkitRequestFullscreen ||
        docEl.mozRequestFullScreen ||
        docEl.msRequestFullscreen;
      if (req) {
        try {
          var p = req.call(docEl);
          if (p && p.then) {
            p.then(function () {
              if (fsGate) fsGate.setAttribute("hidden", "");
            }).catch(function (err) {
              console.warn("Fullscreen request error:", err);
            });
          } else if (fsGate) {
            fsGate.setAttribute("hidden", "");
          }
        } catch (e) {
          console.warn("Fullscreen call exception:", e);
        }
      }
    }

    if (enterFsBtn) {
      enterFsBtn.addEventListener("click", function () {
        enterFullscreen();
      });
    }
    if (reenterFsBtn) {
      reenterFsBtn.addEventListener("click", function () {
        enterFullscreen();
      });
    }

    // Toggle Questions dialog / modal
    var toggleQuestionsBtns = Array.prototype.slice.call(
      document.querySelectorAll("[data-cbt-toggle-questions]")
    );
    toggleQuestionsBtns.forEach(function (btn) {
      btn.addEventListener("click", function () {
        if (questionsModal) questionsModal.removeAttribute("hidden");
      });
    });
    closeQuestionsBtns.forEach(function (btn) {
      btn.addEventListener("click", function () {
        if (questionsModal) questionsModal.setAttribute("hidden", "");
      });
    });

    // Helper: submit anti-cheat violation to lock exam
    function triggerAntiCheat(reason) {
      var locked = kiosk.getAttribute("data-cbt-locked") === "true";
      if (locked || violationTriggered) return;
      violationTriggered = true;
      if (reasonInput) {
        reasonInput.value = reason;
      }
      if (warnForm) {
        if (warnForm.requestSubmit) warnForm.requestSubmit();
        else warnForm.submit();
      }
    }

    // Update navigation buttons, question badges, and answer letters
    function updatePalette() {
      var totalAnswered = 0;
      steps.forEach(function (stepEl, i) {
        var answered = false;
        var choice = "-";
        if (stepEl) {
          var checked = stepEl.querySelector("input[type='radio']:checked");
          var area = stepEl.querySelector("textarea");
          if (checked) {
            answered = true;
            choice = checked.value;
          } else if (area && area.value.trim().length > 0) {
            answered = true;
            choice = "✓";
          }
        }
        if (answered) totalAnswered++;

        var matchingBtns = document.querySelectorAll("[data-cbt-goto='" + i + "']");
        matchingBtns.forEach(function (btn) {
          btn.classList.toggle("is-current", i === currentIdx);
          btn.classList.toggle("is-answered", answered);
          var badge = btn.querySelector("[data-cbt-ans-badge]");
          if (badge) {
            badge.textContent = choice;
          }
        });
      });

      var countBadges = document.querySelectorAll(
        "[data-cbt-answered-count], [data-cbt-answered-badge]"
      );
      countBadges.forEach(function (b) {
        b.textContent = totalAnswered + "/" + steps.length + " Terjawab";
      });
    }

    function showQuestion(idx) {
      if (!steps.length) return;
      currentIdx = Math.max(0, Math.min(steps.length - 1, idx));
      steps.forEach(function (el, i) {
        if (i === currentIdx) {
          el.removeAttribute("hidden");
          el.classList.add("is-active");
        } else {
          el.setAttribute("hidden", "");
          el.classList.remove("is-active");
        }
      });
      if (prevBtn) prevBtn.disabled = currentIdx === 0;
      if (nextBtn) nextBtn.disabled = currentIdx >= steps.length - 1;
      updatePalette();
    }

    // Attach click listener to all question buttons (both in aside and in modal)
    var allGotoBtns = Array.prototype.slice.call(
      document.querySelectorAll("[data-cbt-goto]")
    );
    allGotoBtns.forEach(function (btn) {
      btn.addEventListener("click", function () {
        var targetIdx = parseInt(btn.getAttribute("data-cbt-goto"), 10);
        if (!isNaN(targetIdx)) {
          showQuestion(targetIdx);
          if (questionsModal) questionsModal.setAttribute("hidden", "");
        }
      });
    });

    if (prevBtn) {
      prevBtn.addEventListener("click", function () {
        showQuestion(currentIdx - 1);
      });
    }
    if (nextBtn) {
      nextBtn.addEventListener("click", function () {
        showQuestion(currentIdx + 1);
      });
    }
    kiosk.addEventListener("change", function () {
      updatePalette();
    });
    kiosk.addEventListener("input", function () {
      updatePalette();
    });

    showQuestion(0);

    // Initial fullscreen status check
    if (isFullscreen()) {
      if (fsGate) fsGate.setAttribute("hidden", "");
    } else {
      if (fsGate) fsGate.removeAttribute("hidden");
    }

    // Fullscreen exit detection
    var everFullscreen = isFullscreen();
    function onFullscreenChange() {
      if (isFullscreen()) {
        everFullscreen = true;
        if (fsGate) fsGate.setAttribute("hidden", "");
      } else {
        if (fsGate) fsGate.removeAttribute("hidden");
        if (everFullscreen) {
          triggerAntiCheat(
            "Keluar dari mode Layar Penuh (Fullscreen Exit) tanpa izin pengawas"
          );
        }
      }
    }
    document.addEventListener("fullscreenchange", onFullscreenChange);
    document.addEventListener("webkitfullscreenchange", onFullscreenChange);
    document.addEventListener("mozfullscreenchange", onFullscreenChange);

    // Anti-cheat window blur guard (catches switching to other apps or windows)
    var blurTimer = null;
    window.addEventListener("blur", function () {
      if (!everFullscreen && !isFullscreen()) return;
      blurTimer = setTimeout(function () {
        triggerAntiCheat("Pindah aplikasi / jendela lain (Window Blur) terdeteksi");
      }, 1000);
    });
    window.addEventListener("focus", function () {
      if (blurTimer) {
        clearTimeout(blurTimer);
        blurTimer = null;
      }
    });

    // Anti-cheat tab switch detector (visibilitychange)
    document.addEventListener("visibilitychange", function () {
      if (document.hidden) {
        triggerAntiCheat("Pindah tab peramban (Tab Switch) terdeteksi");
      }
    });

    // Prevent prohibited shortcuts & DevTools keys
    document.addEventListener("keydown", function (e) {
      var locked = kiosk.getAttribute("data-cbt-locked") === "true";
      if (locked || violationTriggered) return;

      var key = e.key;
      var ctrlOrMeta = e.ctrlKey || e.metaKey;

      // Block F12, Ctrl+Shift+I, Ctrl+Shift+J, Ctrl+Shift+C, Ctrl+U (Inspect / DevTools)
      if (
        key === "F12" ||
        (ctrlOrMeta &&
          e.shiftKey &&
          (key === "I" ||
            key === "i" ||
            key === "J" ||
            key === "j" ||
            key === "C" ||
            key === "c")) ||
        (ctrlOrMeta && (key === "u" || key === "U"))
      ) {
        e.preventDefault();
        e.stopPropagation();
        triggerAntiCheat(
          "Mencoba membuka Developer Tools / Inspeksi kode sumber (" + key + ")"
        );
        return false;
      }

      // Block Alt+Tab or Alt+F4
      if (e.altKey && (key === "Tab" || key === "F4")) {
        e.preventDefault();
        triggerAntiCheat("Mencoba beralih aplikasi (Alt+" + key + ")");
        return false;
      }

      // Block Ctrl+N, Ctrl+T, Ctrl+W
      if (
        ctrlOrMeta &&
        (key === "n" ||
          key === "N" ||
          key === "t" ||
          key === "T" ||
          key === "w" ||
          key === "W")
      ) {
        e.preventDefault();
        triggerAntiCheat("Mencoba navigasi tab peramban (" + key + ")");
        return false;
      }

      // Block Copy/Cut/Paste shortcuts outside essay textarea
      if (
        ctrlOrMeta &&
        (key === "c" ||
          key === "C" ||
          key === "x" ||
          key === "X" ||
          key === "v" ||
          key === "V" ||
          key === "a" ||
          key === "A")
      ) {
        var tag =
          e.target && e.target.tagName ? e.target.tagName.toLowerCase() : "";
        if (tag !== "textarea") {
          e.preventDefault();
          return false;
        }
      }

      // Prevent F11 native browser toggle
      if (key === "F11") {
        e.preventDefault();
        return false;
      }
    });

    // Disable context menu (right click)
    document.addEventListener("contextmenu", function (e) {
      e.preventDefault();
      return false;
    });

    // Block clipboard copy/cut outside textarea
    document.addEventListener("copy", function (e) {
      var tag =
        e.target && e.target.tagName ? e.target.tagName.toLowerCase() : "";
      if (tag !== "textarea") {
        e.preventDefault();
        return false;
      }
    });
    document.addEventListener("cut", function (e) {
      var tag =
        e.target && e.target.tagName ? e.target.tagName.toLowerCase() : "";
      if (tag !== "textarea") {
        e.preventDefault();
        return false;
      }
    });

    // Confirmation on accidental tab close / navigation
    window.addEventListener("beforeunload", function (e) {
      var locked = kiosk.getAttribute("data-cbt-locked") === "true";
      if (!locked && !violationTriggered) {
        e.preventDefault();
        e.returnValue = "Ujian sedang berlangsung! Yakin ingin keluar?";
        return e.returnValue;
      }
    });

    // Countdown timer (default 90 minutes)
    if (timerEl) {
      var secs = parseInt(timerEl.getAttribute("data-seconds") || "5400", 10);
      setInterval(function () {
        if (secs > 0) secs -= 1;
        var m = Math.floor(secs / 60);
        var s = secs % 60;
        timerEl.textContent =
          (m < 10 ? "0" + m : m) + ":" + (s < 10 ? "0" + s : s);
      }, 1000);
    }
  }

  // 5. Rewind Story slides
  function initRewindSlides() {
    var container = document.querySelector("[data-rewind-deck]");
    if (!container || container._bound) return;
    container._bound = true;
    var slides = Array.prototype.slice.call(container.querySelectorAll("[data-rewind-slide]"));
    var dots = Array.prototype.slice.call(container.querySelectorAll("[data-rewind-dot]"));
    var prev = container.querySelector("[data-rewind-prev]");
    var next = container.querySelector("[data-rewind-next]");
    var idx = 0;

    function show(n) {
      if (!slides.length) return;
      idx = (n + slides.length) % slides.length;
      slides.forEach(function (s, i) {
        if (i === idx) s.removeAttribute("hidden");
        else s.setAttribute("hidden", "");
      });
      dots.forEach(function (d, i) {
        d.classList.toggle("is-active", i === idx);
      });
    }
    if (prev) prev.addEventListener("click", function () { show(idx - 1); });
    if (next) next.addEventListener("click", function () { show(idx + 1); });
    dots.forEach(function (d, i) {
      d.addEventListener("click", function () { show(i); });
    });
    show(0);
  }

  // Global delegated click listener
  document.addEventListener("click", function (e) {
    var toggleBtn = e.target.closest("[data-drawer-toggle]");
    if (toggleBtn) {
      var drawer = document.getElementById("mobile-drawer");
      var isHidden = !drawer || drawer.hasAttribute("hidden");
      setDrawerOpen(isHidden);
      return;
    }

    var closeDrawerBtn = e.target.closest("[data-drawer-close]");
    if (closeDrawerBtn) {
      setDrawerOpen(false);
      return;
    }

    var openModalBtn = e.target.closest("[data-modal-open]");
    if (openModalBtn) {
      openModal(openModalBtn.getAttribute("data-modal-open"));
      return;
    }

    var closeModalBtn = e.target.closest("[data-modal-close]");
    if (closeModalBtn) {
      closeModal(closeModalBtn.closest(".modal"));
      return;
    }

    // Close open <details class="user-menu"> when clicking outside
    document.querySelectorAll("details.user-menu[open]").forEach(function (menu) {
      if (!menu.contains(e.target)) {
        menu.removeAttribute("open");
      }
    });
  });

  document.addEventListener("keydown", function (e) {
    if (e.key === "Escape") {
      setDrawerOpen(false);
      document.querySelectorAll(".modal:not([hidden])").forEach(closeModal);
      document.querySelectorAll("details.user-menu[open]").forEach(function (m) {
        m.removeAttribute("open");
      });
    }
  });

  document.addEventListener("DOMContentLoaded", function () {
    initCbtKiosk();
    initRewindSlides();
  });

  document.addEventListener("htmx:afterSwap", function () {
    initCbtKiosk();
    initRewindSlides();
  });
})();
