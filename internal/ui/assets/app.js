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

  // 4. CBT Exam Kiosk question navigator + anti-cheat focus guard
  function initCbtKiosk() {
    var kiosk = document.querySelector("[data-cbt-kiosk]");
    if (!kiosk || kiosk._bound) return;
    kiosk._bound = true;

    var steps = Array.prototype.slice.call(kiosk.querySelectorAll("[data-cbt-question]"));
    var navBtns = Array.prototype.slice.call(kiosk.querySelectorAll("[data-cbt-goto]"));
    var prevBtn = kiosk.querySelector("[data-cbt-prev]");
    var nextBtn = kiosk.querySelector("[data-cbt-next]");
    var timerEl = kiosk.querySelector("[data-cbt-timer]");
    var currentIdx = 0;

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
      navBtns.forEach(function (btn, i) {
        btn.classList.toggle("is-current", i === currentIdx);
        var stepEl = steps[i];
        var answered = false;
        if (stepEl) {
          var checked = stepEl.querySelector("input[type='radio']:checked");
          var area = stepEl.querySelector("textarea");
          answered = Boolean(checked || (area && area.value.trim().length > 0));
        }
        btn.classList.toggle("is-answered", answered);
      });
      if (prevBtn) prevBtn.disabled = currentIdx === 0;
      if (nextBtn) nextBtn.disabled = currentIdx >= steps.length - 1;
    }

    navBtns.forEach(function (btn, i) {
      btn.addEventListener("click", function () {
        showQuestion(i);
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
      showQuestion(currentIdx);
    });
    kiosk.addEventListener("input", function () {
      showQuestion(currentIdx);
    });
    showQuestion(0);

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

    // Anti-cheat tab switch detector (>1.5s blur triggers proctor status)
    var blurTimer = null;
    document.addEventListener("visibilitychange", function () {
      var locked = kiosk.getAttribute("data-cbt-locked") === "true";
      if (locked) return;
      if (document.hidden) {
        blurTimer = setTimeout(function () {
          var warnForm = document.getElementById("cbt-violation-form");
          if (warnForm) {
            if (warnForm.requestSubmit) warnForm.requestSubmit();
            else warnForm.submit();
          }
        }, 1500);
      } else if (blurTimer) {
        clearTimeout(blurTimer);
        blurTimer = null;
      }
    });
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
