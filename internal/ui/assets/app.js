(function () {
  "use strict";

  // 0. Connection status in the shell topbar.
  function updateConnectionStatus() {
    var offline = navigator.onLine === false;
    document.querySelectorAll("[data-connection-status]").forEach(function (chip) {
      chip.hidden = !offline;
    });
  }

  window.addEventListener("online", updateConnectionStatus);
  window.addEventListener("offline", updateConnectionStatus);
  updateConnectionStatus();

  // 0b. The public landing menu is a <details>; close it once a link is picked
  // so the panel does not stay on top of the section it just scrolled to.
  function closeHomeMenu() {
    document.querySelectorAll(".home-menu[open], .sf-menu[open], .sf-drop[open]").forEach(function (menu) {
      menu.removeAttribute("open");
    });
  }

  document.addEventListener("click", function (event) {
    var target = event.target;
    if (!target || !target.closest) return;
    var menu = target.closest(".home-menu, .sf-menu, .sf-drop");
    if (menu) {
      if (target.closest(".home-menu-panel a, .sf-menu-panel a, .sf-drop-panel a")) {
        menu.removeAttribute("open");
      }
      // clicking outside any open menu closes the rest
      document.querySelectorAll(".home-menu[open], .sf-menu[open], .sf-drop[open]").forEach(function (open) {
        if (open !== menu) open.removeAttribute("open");
      });
      return;
    }
    closeHomeMenu();
  });

  document.addEventListener("keydown", function (event) {
    if (event.key === "Escape") closeHomeMenu();
  });

  // 0c. The floating landing controls tuck away while the page is read
  // downwards and return on the first upward scroll, so they never sit on a
  // heading (on a phone the menu button otherwise lands on body copy).
  (function () {
    var nav = document.querySelector(".home-nav");
    var menu = document.querySelector(".home-menu");
    var login = document.querySelector(".home-dock-login");
    var controls = [nav, menu, login].filter(Boolean);
    if (!controls.length) return;
    var lastY = window.scrollY;
    var queued = false;

    function setTucked(tucked) {
      controls.forEach(function (control) {
        if (tucked && control === menu && control.hasAttribute("open")) return;
        control.classList.toggle("is-tucked", tucked);
      });
    }

    function update() {
      queued = false;
      var y = window.scrollY;
      if (y > lastY + 4 && y > 200) {
        setTucked(true);
      } else if (y < lastY - 4 || y < 80) {
        setTucked(false);
      }
      lastY = y;
    }

    window.addEventListener(
      "scroll",
      function () {
        if (queued) return;
        queued = true;
        window.requestAnimationFrame(update);
      },
      { passive: true }
    );
  })();

  // 1. Mobile drawer toggle & close
  function setDrawerOpen(open) {
    var drawer = document.getElementById("mobile-drawer");
    var backdrop = document.getElementById("mobile-drawer-backdrop");
    var toggles = document.querySelectorAll("[data-drawer-toggle]");
    if (!drawer || !backdrop) return;
    if (open) {
      if (!drawer._returnFocus) drawer._returnFocus = document.activeElement;
      drawer.removeAttribute("hidden");
      backdrop.removeAttribute("hidden");
      document.body.classList.add("drawer-open");
      var firstItem = drawer.querySelector("a[href], button:not([disabled])");
      if (firstItem) firstItem.focus();
    } else {
      drawer.setAttribute("hidden", "");
      backdrop.setAttribute("hidden", "");
      document.body.classList.remove("drawer-open");
      if (drawer._returnFocus && document.contains(drawer._returnFocus)) {
        drawer._returnFocus.focus();
      }
      drawer._returnFocus = null;
    }
    toggles.forEach(function (btn) {
      btn.setAttribute("aria-expanded", open ? "true" : "false");
    });
  }

  // 2. Modal open & close. The modal is a native <dialog>, so the browser
  // provides the backdrop, inert page, Esc handling and focus return.
  function openModal(selector) {
    var modal = document.querySelector(selector);
    if (!modal) return;
    if (typeof modal.showModal === "function") {
      if (!modal.open) modal.showModal();
    } else {
      modal.setAttribute("open", "");
    }
    initQuizWizard(modal);
  }

  function closeModal(modal) {
    if (!modal) return;
    if (typeof modal.close === "function") {
      if (modal.open) modal.close();
    } else {
      modal.removeAttribute("open");
    }
    stopModalTimers(modal);
  }

  function stopModalTimers(modal) {
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

  // 4. CBT Exam Kiosk question navigator + cross-platform anti-cheat (HP, Tablet, Laptop)
  function initCbtKiosk() {
    var kiosk = document.querySelector("[data-cbt-kiosk]");
    if (!kiosk || kiosk._bound) return;
    kiosk._bound = true;

    // Enter kiosk isolation mode (hides ERP sidebar and topbar)
    document.body.classList.add("is-cbt-active");

    var examForm = document.getElementById("cbt-exam-form") || kiosk.querySelector("form");
    var steps = Array.prototype.slice.call(kiosk.querySelectorAll("[data-cbt-question]"));
    var prevBtn = kiosk.querySelector("[data-cbt-prev]");
    var nextBtn = kiosk.querySelector("[data-cbt-next]");
    var submitBtn = kiosk.querySelector("[data-cbt-submit-btn]");
    var navIndicator = kiosk.querySelector("[data-cbt-nav-indicator]");
    var timerEl = kiosk.querySelector("[data-cbt-timer]");
    var fsGate = document.querySelector("[data-cbt-fullscreen-gate]");
    var enterFsBtn = document.querySelector("[data-cbt-enter-fullscreen]");
    var reenterFsBtn = document.querySelector("[data-cbt-reenter-fullscreen]");
    var questionsModal = document.querySelector("[data-cbt-questions-dialog]");
    var modalSubmitBtn = document.querySelector("[data-cbt-trigger-submit]");
    var strikeModal = document.querySelector("[data-cbt-strike-dialog]");
    var strikeBadge = document.querySelector("[data-cbt-strike-badge]");
    var strikeTitle = document.querySelector("[data-cbt-strike-title]");
    var strikeDesc = document.querySelector("[data-cbt-strike-desc]");
    var strikeResumeBtn = document.querySelector("[data-cbt-strike-resume]");
    var warnForm = document.getElementById("cbt-violation-form");
    var reasonInput = warnForm ? warnForm.querySelector("input[name='reason']") : null;

    var currentIdx = 0;
    var violationTriggered = false;
    var MAX_STRIKES = 3;
    var everFullscreen = false;
    var examStarted = false;
    var lastViolationAt = 0;

    var examId = kiosk.getAttribute("data-cbt-exam-id") || "exam-sch-01";
    var storageKey = "wesmk_strikes_" + examId;
    var serverWarnCount = parseInt(kiosk.getAttribute("data-cbt-warn-count") || "0", 10);
    var savedStrikes = 0;
    try {
      savedStrikes = parseInt(sessionStorage.getItem(storageKey) || "0", 10) || 0;
    } catch (e) {}
    var strikeCount = Math.max(serverWarnCount, savedStrikes);

    function unlockKioskView() {
      everFullscreen = true;
      examStarted = true;
      kiosk.classList.remove("is-gate-locked");
      if (fsGate) fsGate.setAttribute("hidden", "");
    }

    function lockKioskView() {
      kiosk.classList.add("is-gate-locked");
    }

    // Helper: detect if browser is currently in native fullscreen
    function isFullscreen() {
      return Boolean(
        document.fullscreenElement ||
        document.webkitFullscreenElement ||
        document.mozFullScreenElement ||
        document.msFullscreenElement
      );
    }

    // Helper: enter fullscreen with fallback for mobile browsers (iOS Safari, Android WebApp)
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
              unlockKioskView();
            }).catch(function () {
              unlockKioskView();
            });
          } else {
            unlockKioskView();
          }
        } catch (e) {
          unlockKioskView();
        }
      } else {
        unlockKioskView();
        window.scrollTo(0, 0);
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

    // Toggle Questions dialog / bottom sheet
    var toggleQuestionsBtns = Array.prototype.slice.call(
      document.querySelectorAll("[data-cbt-toggle-questions]")
    );
    toggleQuestionsBtns.forEach(function (btn) {
      btn.addEventListener("click", function () {
        if (questionsModal) questionsModal.removeAttribute("hidden");
      });
    });

    var closeQuestionsBtns = Array.prototype.slice.call(
      document.querySelectorAll("[data-cbt-close-questions]")
    );
    closeQuestionsBtns.forEach(function (btn) {
      btn.addEventListener("click", function () {
        if (questionsModal) questionsModal.setAttribute("hidden", "");
      });
    });

    // Modal submit button triggers exam submission
    if (modalSubmitBtn && examForm) {
      modalSubmitBtn.addEventListener("click", function () {
        if (confirm("Yakin ingin mengumpulkan seluruh jawaban ujian sekarang?")) {
          violationTriggered = true;
          try { sessionStorage.removeItem(storageKey); } catch (e) {}
          if (examForm.requestSubmit) examForm.requestSubmit();
          else examForm.submit();
        }
      });
    }

    if (examForm) {
      examForm.addEventListener("submit", function () {
        violationTriggered = true;
        try { sessionStorage.removeItem(storageKey); } catch (e) {}
      });
    }

    // Resume button from anti-cheat warning
    if (strikeResumeBtn) {
      strikeResumeBtn.addEventListener("click", function () {
        if (strikeModal) strikeModal.setAttribute("hidden", "");
        enterFullscreen();
      });
    }

    // Persist warning strike to server asynchronously so page refresh cannot reset strikeCount
    function reportWarnToServer(reason) {
      if (!warnForm) return;
      try {
        var csrfEl = warnForm.querySelector("input[name='_csrf']");
        var body = new URLSearchParams();
        if (csrfEl) body.set("_csrf", csrfEl.value);
        body.set("status", "WARN");
        body.set("reason", reason);
        body.set("async", "1");
        fetch("/student/proctor-status", {
          method: "POST",
          headers: {
            "Content-Type": "application/x-www-form-urlencoded",
            "X-Requested-With": "XMLHttpRequest"
          },
          body: body.toString(),
          keepalive: true
        }).catch(function () {});
      } catch (e) {}
    }

    // Lock exam permanently via server form
    function triggerAntiCheat(reason) {
      var locked = kiosk.getAttribute("data-cbt-locked") === "true";
      if (locked || violationTriggered) return;
      violationTriggered = true;
      lockKioskView();
      if (reasonInput) {
        reasonInput.value = reason;
      }
      if (warnForm) {
        if (warnForm.requestSubmit) warnForm.requestSubmit();
        else warnForm.submit();
      }
    }

    // Strike management system (warn on strike 1 & 2, permanent lock on strike 3 or critical)
    function handleViolation(reason, isCritical) {
      var locked = kiosk.getAttribute("data-cbt-locked") === "true";
      if (locked || violationTriggered) return;

      var now = Date.now();
      if (!isCritical && now - lastViolationAt < 1200) {
        return;
      }
      lastViolationAt = now;

      lockKioskView();

      if (isCritical) {
        triggerAntiCheat(reason);
        return;
      }

      strikeCount++;
      try {
        sessionStorage.setItem(storageKey, String(strikeCount));
      } catch (e) {}

      if (strikeCount >= MAX_STRIKES) {
        triggerAntiCheat(
          "Batas pelanggaran tercapai (" +
            strikeCount +
            "/" +
            MAX_STRIKES +
            "): " +
            reason
        );
      } else {
        reportWarnToServer(reason);
        if (strikeModal) {
          if (strikeBadge) {
            strikeBadge.textContent = "Pelanggaran " + strikeCount + "/" + MAX_STRIKES;
          }
          if (strikeTitle) {
            strikeTitle.textContent = "Perpindahan Fokus Terdeteksi!";
          }
          if (strikeDesc) {
            strikeDesc.textContent = reason;
          }
          strikeModal.removeAttribute("hidden");
        }
      }
    }

    // Update navigation palette, question badges, answer letters & submit states
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
        b.textContent = totalAnswered + "/" + steps.length;
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

      var isLast = currentIdx >= steps.length - 1;
      if (nextBtn) {
        if (isLast) nextBtn.setAttribute("hidden", "");
        else nextBtn.removeAttribute("hidden");
      }
      if (submitBtn) {
        if (isLast) submitBtn.removeAttribute("hidden");
        else submitBtn.setAttribute("hidden", "");
      }
      if (navIndicator) {
        navIndicator.textContent = "Soal " + (currentIdx + 1) + "/" + steps.length;
      }

      updatePalette();
      window.scrollTo(0, 0);
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
      unlockKioskView();
    } else {
      lockKioskView();
      if (fsGate) fsGate.removeAttribute("hidden");
    }

    // Native fullscreen exit detection
    function onFullscreenChange() {
      if (isFullscreen()) {
        unlockKioskView();
      } else {
        lockKioskView();
        if (fsGate) fsGate.removeAttribute("hidden");
        if (everFullscreen || examStarted) {
          handleViolation("Keluar dari mode Layar Penuh tanpa izin pengawas", false);
        }
      }
    }
    document.addEventListener("fullscreenchange", onFullscreenChange);
    document.addEventListener("webkitfullscreenchange", onFullscreenChange);
    document.addEventListener("mozfullscreenchange", onFullscreenChange);

    // Cross-Platform Anti-Cheat: Visibility & App-Switch Detection (HP, Tablet, Laptop)
    var hideStartTime = null;
    document.addEventListener("visibilitychange", function () {
      if (document.hidden) {
        hideStartTime = Date.now();
        lockKioskView();
      } else {
        if (hideStartTime) {
          var elapsedSec = (Date.now() - hideStartTime) / 1000;
          hideStartTime = null;
          if (elapsedSec > 3.0) {
            handleViolation(
              "Meninggalkan aplikasi ujian selama " + Math.round(elapsedSec) + " detik (berpindah aplikasi/tab)",
              true
            );
          } else {
            handleViolation("Terdeteksi beralih aplikasi / tab peramban", false);
          }
        }
      }
    });

    // Window Blur Guard (catches notification drawer pull, floating calculator, or alt-tab)
    var blurTimer = null;
    window.addEventListener("blur", function () {
      if (!everFullscreen && !examStarted && !isFullscreen()) return;
      blurTimer = setTimeout(function () {
        handleViolation("Pindah jendela / aplikasi lain (Window Blur) terdeteksi", false);
      }, 600);
    });
    window.addEventListener("focus", function () {
      if (blurTimer) {
        clearTimeout(blurTimer);
        blurTimer = null;
      }
    });

    // Mobile / Tablet Split-Screen Detection on Resize
    window.addEventListener("resize", function () {
      var activeEl = document.activeElement;
      var isInput = activeEl && (activeEl.tagName === "INPUT" || activeEl.tagName === "TEXTAREA");
      if (!isInput && window.innerHeight > 0) {
        if (window.screen && window.screen.availHeight > 500) {
          var ratio = window.innerHeight / window.screen.availHeight;
          if (ratio < 0.55 && window.innerHeight < 380) {
            handleViolation("Mode Layar Terbagi (Split-Screen / Floating Window) terdeteksi", false);
          }
        }
      }
    });

    // Mobile & Touch Gesture Trap: Prevent back gesture navigation
    try {
      history.pushState({ wesmkCbt: true }, "", location.href);
      window.addEventListener("popstate", function () {
        history.pushState({ wesmkCbt: true }, "", location.href);
        handleViolation("Tombol navigasi Kembali (Back) dinonaktifkan selama ujian", false);
      });
    } catch (e) {}

    // Disable text selection, context menu, drag-and-drop
    document.addEventListener("selectstart", function (e) {
      var tag = e.target && e.target.tagName ? e.target.tagName.toLowerCase() : "";
      if (tag !== "textarea" && tag !== "input") {
        e.preventDefault();
        return false;
      }
    });

    document.addEventListener("contextmenu", function (e) {
      e.preventDefault();
      return false;
    });

    document.addEventListener("dragstart", function (e) {
      e.preventDefault();
      return false;
    });

    document.addEventListener("drop", function (e) {
      e.preventDefault();
      return false;
    });

    // Block keyboard shortcuts, PrintScreen, Print, Save, Find & DevTools
    document.addEventListener("keydown", function (e) {
      var locked = kiosk.getAttribute("data-cbt-locked") === "true";
      if (locked || violationTriggered) return;

      var key = e.key;
      var ctrlOrMeta = e.ctrlKey || e.metaKey;

      // Block F12, Ctrl+Shift+I, Ctrl+Shift+J, Ctrl+Shift+C, Ctrl+U
      if (
        key === "F12" ||
        (ctrlOrMeta &&
          e.shiftKey &&
          (key === "I" || key === "i" || key === "J" || key === "j" || key === "C" || key === "c")) ||
        (ctrlOrMeta && (key === "u" || key === "U"))
      ) {
        e.preventDefault();
        e.stopPropagation();
        handleViolation("Mencoba membuka Developer Tools (" + key + ")", true);
        return false;
      }

      // Block PrintScreen / Screenshot keys
      if (key === "PrintScreen" || (ctrlOrMeta && e.shiftKey && (key === "S" || key === "s"))) {
        e.preventDefault();
        if (navigator.clipboard && navigator.clipboard.writeText) {
          navigator.clipboard.writeText("").catch(function () {});
        }
        handleViolation("Mencoba melakukan tangkapan layar (Screenshot)", false);
        return false;
      }

      // Block Print (Ctrl+P), Save (Ctrl+S), Find (Ctrl+F / Ctrl+G)
      if (
        ctrlOrMeta &&
        (key === "p" || key === "P" || key === "s" || key === "S" || key === "f" || key === "F" || key === "g" || key === "G")
      ) {
        e.preventDefault();
        return false;
      }

      // Block Alt+Tab or Alt+F4
      if (e.altKey && (key === "Tab" || key === "F4")) {
        e.preventDefault();
        handleViolation("Mencoba beralih aplikasi (Alt+" + key + ")", false);
        return false;
      }

      // Block Ctrl+N, Ctrl+T, Ctrl+W, Ctrl+R, F5
      if (
        (ctrlOrMeta && (key === "n" || key === "N" || key === "t" || key === "T" || key === "w" || key === "W" || key === "r" || key === "R")) ||
        key === "F5"
      ) {
        e.preventDefault();
        handleViolation("Mencoba navigasi / muat ulang halaman (" + key + ")", false);
        return false;
      }

      // Block Copy/Cut/Paste shortcuts (even in essay textarea, block external paste)
      if (
        ctrlOrMeta &&
        (key === "c" || key === "C" || key === "x" || key === "X" || key === "v" || key === "V")
      ) {
        e.preventDefault();
        return false;
      }

      // Prevent F11 native browser toggle
      if (key === "F11") {
        e.preventDefault();
        return false;
      }
    });

    // Block clipboard copy, cut, and external paste (prevents pasting AI/web answers into essay)
    document.addEventListener("copy", function (e) {
      e.preventDefault();
      return false;
    });
    document.addEventListener("cut", function (e) {
      e.preventDefault();
      return false;
    });
    document.addEventListener("paste", function (e) {
      e.preventDefault();
      handleViolation("Menempelkan teks dari luar (Paste) dilarang selama ujian", false);
      return false;
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

  // 6. Gacha Mystery Orb for Average Exam Score with 2-Stage Click & Aura Lighting Tiers
  function initMysteryOrb() {
    var container = document.querySelector("[data-cbt-orb-container]");
    if (!container || container._bound) return;
    container._bound = true;

    var orbBtn = container.querySelector("[data-cbt-mystery-orb]");
    var symbolEl = container.querySelector("[data-cbt-orb-symbol]");
    var scoreEl = container.querySelector("[data-cbt-orb-score]");
    var hintEl = container.querySelector("[data-cbt-orb-hint]");
    var pillEl = container.querySelector("[data-cbt-tier-pill]");
    var tierNameEl = container.querySelector("[data-cbt-tier-name]");
    var subjectScoresCard = container.querySelector("[data-cbt-subject-scores]");
    var toggleScoresBtn = container.querySelector("[data-cbt-toggle-scores]");

    var currentTier = container.getAttribute("data-tier") || "stellar";
    var currentScore = parseInt(container.getAttribute("data-score") || "0", 10);
    var clickStage = 0; // 0: netral ?, 1: sedikit warna & berputar, 2: baru berwarna penuh & reveal
    var isScoresMasked = false;

    var tierInfo = {
      stellar: {
        cls: "cbt-tier-stellar",
        pillCls: "pill-stellar",
        name: "✦ STELLAR JADE (91 - 100) ✦",
        spinHint: "✨ Ketuk sekali lagi untuk membuka nilai!",
        fullHint: "Rata-rata ujian: " + currentScore + "/100"
      },
      gold: {
        cls: "cbt-tier-gold",
        pillCls: "pill-gold",
        name: "★ EMAS (81 - 90) ★",
        spinHint: "✨ Ketuk sekali lagi untuk membuka nilai!",
        fullHint: "Rata-rata ujian: " + currentScore + "/100"
      },
      blue: {
        cls: "cbt-tier-blue",
        pillCls: "pill-blue",
        name: "◆ BIRU (71 - 80) ◆",
        spinHint: "✨ Ketuk sekali lagi untuk membuka nilai!",
        fullHint: "Rata-rata ujian: " + currentScore + "/100"
      },
      green: {
        cls: "cbt-tier-green",
        pillCls: "pill-green",
        name: "● HIJAU (1 - 70) ●",
        spinHint: "✨ Ketuk sekali lagi untuk membuka nilai!",
        fullHint: "Rata-rata ujian: " + currentScore + "/100"
      }
    };

    function applyTier(tier, score) {
      currentTier = tier;
      currentScore = score;
      container.classList.remove("cbt-tier-stellar", "cbt-tier-gold", "cbt-tier-blue", "cbt-tier-green");
      var info = tierInfo[currentTier] || tierInfo.stellar;
      container.classList.add(info.cls);
      if (scoreEl && currentScore > 0) scoreEl.textContent = currentScore;
    }

    applyTier(currentTier, currentScore);

    var shakeTimer = null;
    function triggerClick(e) {
      if (e) {
        e.preventDefault();
        e.stopPropagation();
      }

      // 1. STRICT LOCK CHECK: Dicek dinamis setiap klik! Tak bisa dibuka jika belum semua ujian selesai.
      var isUnlocked = container.getAttribute("data-unlocked") === "true";
      if (!isUnlocked) {
        if (orbBtn) {
          orbBtn.classList.remove("is-shaking");
          void orbBtn.offsetWidth; // force reflow
          orbBtn.classList.add("is-shaking");
          if (shakeTimer) clearTimeout(shakeTimer);
          shakeTimer = setTimeout(function () {
            if (orbBtn) orbBtn.classList.remove("is-shaking");
          }, 600);
        }
        if (navigator.vibrate) {
          navigator.vibrate([80, 40, 80]);
        }
        if (hintEl) {
          hintEl.textContent = "Belum dapat dibuka. Selesaikan seluruh ujian terlebih dahulu.";
          hintEl.classList.add("text-danger");
          setTimeout(function () {
            if (hintEl) {
              hintEl.classList.remove("text-danger");
              hintEl.textContent = "Selesaikan seluruh ujian (4 mapel) untuk membuka nilai.";
            }
          }, 3000);
        }
        return; // Hentikan di sini!
      }

      var info = tierInfo[currentTier] || tierInfo.stellar;

      if (clickStage === 0) {
        // Klik 1: "di klik pertama hanya ada sedikit warna dan berputar"
        clickStage = 1;
        orbBtn.classList.remove("is-aura-full", "is-revealed");
        orbBtn.classList.add("is-spinning-aura");
        if (symbolEl) symbolEl.removeAttribute("hidden"); // Tanda tanya ? tetap terlihat
        if (scoreEl) scoreEl.setAttribute("hidden", "");
        if (pillEl) pillEl.setAttribute("hidden", "");
        if (hintEl) hintEl.textContent = info.spinHint;
        if (navigator.vibrate) navigator.vibrate(60);
      } else if (clickStage === 1) {
        // Klik 2: "setelah klik dua kali baru berwarna" & reveal nilai rata-rata + mapel
        clickStage = 2;
        orbBtn.classList.remove("is-spinning-aura");
        orbBtn.classList.add("is-aura-full", "is-revealed");
        if (symbolEl) symbolEl.setAttribute("hidden", "");
        if (scoreEl) {
          scoreEl.textContent = currentScore;
          scoreEl.removeAttribute("hidden");
        }
        if (pillEl) {
          pillEl.removeAttribute("hidden");
          pillEl.className = "cbt-tier-pill " + info.pillCls;
        }
        if (tierNameEl) tierNameEl.textContent = info.name;
        if (hintEl) hintEl.textContent = "Rata-rata ujian: " + currentScore + "/100";
        if (navigator.vibrate) navigator.vibrate([100, 50, 120]);

        // Munculkan rincian nilai per mapel di bawahnya
        if (subjectScoresCard) {
          subjectScoresCard.removeAttribute("hidden");
          subjectScoresCard.classList.add("is-revealed");
        }
      } else {
        // Klik 3: Reset agar siswa bisa mengulang animasinya
        clickStage = 0;
        orbBtn.classList.remove("is-spinning-aura", "is-aura-full", "is-revealed");
        if (symbolEl) symbolEl.removeAttribute("hidden");
        if (scoreEl) scoreEl.setAttribute("hidden", "");
        if (pillEl) pillEl.setAttribute("hidden", "");
        if (hintEl) hintEl.textContent = "Ketuk lingkaran untuk melihat aura & nilai rata-rata.";
      }
    }

    if (orbBtn) {
      orbBtn.addEventListener("click", triggerClick);
    }

    // Toggle Privacy: Hide / Show Individual Subject Scores
    if (toggleScoresBtn) {
      toggleScoresBtn.addEventListener("click", function (e) {
        e.preventDefault();
        e.stopPropagation();
        isScoresMasked = !isScoresMasked;

        var valEls = Array.prototype.slice.call(container.querySelectorAll("[data-cbt-score-val]"));
        var maskEls = Array.prototype.slice.call(container.querySelectorAll("[data-cbt-score-mask]"));
        var iconVisible = toggleScoresBtn.querySelector("[data-icon-visible]");
        var iconHidden = toggleScoresBtn.querySelector("[data-icon-hidden]");
        var labelEl = toggleScoresBtn.querySelector("[data-cbt-toggle-text]");

        valEls.forEach(function (el) {
          if (isScoresMasked) el.setAttribute("hidden", "");
          else el.removeAttribute("hidden");
        });

        maskEls.forEach(function (el) {
          if (isScoresMasked) el.removeAttribute("hidden");
          else el.setAttribute("hidden", "");
        });

        if (labelEl) {
          labelEl.textContent = isScoresMasked ? "Tampilkan" : "Sembunyikan";
        }
        if (iconVisible && iconHidden) {
          if (isScoresMasked) {
            iconVisible.setAttribute("hidden", "");
            iconHidden.removeAttribute("hidden");
          } else {
            iconVisible.removeAttribute("hidden");
            iconHidden.setAttribute("hidden", "");
          }
        }
      });
    }
  }

  // Turn ordinary data tables into labelled cards on narrow screens.
  // Header text is copied into data-label so CSS can render it above each value.
  function initMobileCards() {
    var tables = document.querySelectorAll("table.data-table");
    tables.forEach(function (table) {
      if (table.classList.contains("rollover-table")) return;

      var head = table.tHead;
      var headRow = head && head.rows[head.rows.length - 1];
      if (!headRow) return;

      var headers = Array.prototype.slice.call(headRow.cells);
      if (headers.length < 2) return;
      if (headers.some(function (cell) { return cell.colSpan > 1; })) return;

      var labels = headers.map(function (cell) {
        return (cell.textContent || "").replace(/\s+/g, " ").trim();
      });
      if (labels.some(function (label) { return !label; })) return;

      table.classList.add("mobile-cards");
      var wrap = table.closest(".table-wrap");
      if (wrap) wrap.classList.add("has-mobile-cards");

      Array.prototype.forEach.call(table.tBodies, function (body) {
        Array.prototype.forEach.call(body.rows, function (row) {
          var cells = row.cells;
          if (cells.length === 1 && cells[0].colSpan > 1) {
            row.classList.add("is-empty-row");
            return;
          }
          var column = 0;
          Array.prototype.forEach.call(cells, function (cell) {
            if (cell.colSpan > 1) {
              column += cell.colSpan;
              return;
            }
            var label = labels[column];
            column += 1;
            if (label && !cell.hasAttribute("data-label")) {
              cell.setAttribute("data-label", label);
            }
          });
        });
      });
    });
  }

  // Auto-submit a filter form when its control changes. Inline onchange
  // handlers are blocked by CSP, so controls declare data-submit-on-change.
  document.addEventListener("change", function (e) {
    var control = e.target.closest("[data-submit-on-change]");
    if (!control || !control.form) return;
    if (control.form.requestSubmit) control.form.requestSubmit();
    else control.form.submit();
  });

  // Confirmation for destructive forms. CSP blocks inline onsubmit handlers,
  // so forms declare data-confirm="..." instead.
  document.addEventListener("submit", function (e) {
    var form = e.target;
    if (!form || !form.getAttribute) return;
    var message = form.getAttribute("data-confirm");
    if (message && !window.confirm(message)) {
      e.preventDefault();
      e.stopPropagation();
    }
  }, true);

  // 8. Failure feedback: htmx errors should be visible, not only logged.
  var appToast = null;
  var appToastTimer = null;
  var appToastAction = null;
  var lastFailedElt = null;

  function hideAppToast() {
    clearTimeout(appToastTimer);
    if (!appToast) return;
    appToast.classList.remove("is-visible");
    appToast.setAttribute("aria-hidden", "true");
  }

  function getAppToast() {
    if (appToast) return appToast;
    appToast = document.createElement("div");
    appToast.className = "app-toast";
    appToast.setAttribute("role", "alert");
    appToast.setAttribute("aria-live", "assertive");
    appToast.setAttribute("aria-atomic", "true");
    appToast.setAttribute("aria-hidden", "true");
    appToast.innerHTML = '<span class="app-toast-mark" aria-hidden="true">!</span>' +
      '<div class="app-toast-copy">' +
      '<p class="app-toast-message"></p>' +
      '<button type="button" class="app-toast-action" hidden></button>' +
      '</div>' +
      '<button type="button" class="app-toast-close" aria-label="Tutup notifikasi">\u00d7</button>';
    appToast.querySelector(".app-toast-close").addEventListener("click", hideAppToast);
    appToast.querySelector(".app-toast-action").addEventListener("click", function () {
      if (!appToastAction || !appToastAction.handler) return;
      appToastAction.handler();
    });
    (document.body || document.documentElement).appendChild(appToast);
    return appToast;
  }

  function setAppToastAction(action) {
    var button = appToast.querySelector(".app-toast-action");
    appToastAction = action || null;
    if (!appToastAction) {
      button.hidden = true;
      button.textContent = "";
      return;
    }
    button.textContent = appToastAction.label;
    button.hidden = false;
  }

  function showAppToast(message, action) {
    var toast = getAppToast();
    toast.querySelector(".app-toast-message").textContent = message;
    setAppToastAction(action);
    toast.removeAttribute("aria-hidden");
    toast.classList.remove("is-visible");
    void toast.offsetWidth;
    toast.classList.add("is-visible");
    clearTimeout(appToastTimer);
    appToastTimer = setTimeout(hideAppToast, 7000);
  }

  function retryLastFailedRequest() {
    var elt = lastFailedElt;
    hideAppToast();
    if (!elt || !document.body.contains(elt)) {
      window.location.reload();
      return;
    }
    if (elt.tagName === "FORM" && elt.requestSubmit) {
      elt.requestSubmit();
      return;
    }
    if (elt.click) elt.click();
  }

  function reportHtmxFailure(evt, message, action) {
    if (isBackgroundHtmx(evt)) return;
    if (action && action.retry) {
      lastFailedElt = evt.detail && evt.detail.elt;
      action = { label: "Coba lagi", handler: retryLastFailedRequest };
    }
    showAppToast(message, action);
  }

  document.addEventListener("htmx:sendError", function (e) {
    reportHtmxFailure(e, "Koneksi terputus. Perubahan belum tersimpan.", { retry: true });
  });
  document.addEventListener("htmx:timeout", function (e) {
    reportHtmxFailure(e, "Server terlalu lama merespons. Coba lagi.", { retry: true });
  });
  document.addEventListener("htmx:responseError", function (e) {
    var status = e.detail && e.detail.xhr ? e.detail.xhr.status : 0;
    if (status === 403) {
      reportHtmxFailure(e, "Sesi keamanan tidak valid. Muat ulang halaman lalu coba lagi.", {
        label: "Muat ulang",
        handler: function () {
          window.location.reload();
        },
      });
      return;
    }
    if (status === 429) {
      reportHtmxFailure(e, "Terlalu banyak percobaan. Tunggu sebentar lalu coba lagi.");
      return;
    }
    reportHtmxFailure(e, "Aksi gagal diproses. Coba lagi.", { retry: true });
  });

  // 9. Navigation feedback: a slim progress bar for full-page moves and htmx.
  var navProgress = null;
  var navProgressTimer = null;

  function getNavProgress() {
    if (!navProgress) {
      navProgress = document.createElement("div");
      navProgress.className = "nav-progress";
      navProgress.setAttribute("aria-hidden", "true");
      (document.body || document.documentElement).appendChild(navProgress);
    }
    return navProgress;
  }

  function startNavProgress() {
    var bar = getNavProgress();
    clearTimeout(navProgressTimer);
    bar.classList.remove("is-done");
    bar.classList.remove("is-active");
    void bar.offsetWidth;
    bar.classList.add("is-active");
    navProgressTimer = setTimeout(finishNavProgress, 10000);
  }

  function finishNavProgress() {
    clearTimeout(navProgressTimer);
    if (!navProgress) return;
    navProgress.classList.remove("is-active");
    navProgress.classList.add("is-done");
    setTimeout(function () {
      if (navProgress) navProgress.classList.remove("is-done");
    }, 240);
  }

  function isPlainNavigation(e) {
    return !e.defaultPrevented && e.button === 0 && !e.metaKey && !e.ctrlKey && !e.shiftKey && !e.altKey;
  }

  document.addEventListener("click", function (e) {
    if (!isPlainNavigation(e)) return;
    if (!e.target || !e.target.closest) return;
    var link = e.target.closest("a[href]");
    if (!link || link.hasAttribute("data-no-progress")) return;
    if (link.target && link.target !== "_self") return;
    if (link.hasAttribute("download")) return;
    var href = link.getAttribute("href");
    if (!href || href.charAt(0) === "#" || /^(mailto:|tel:|javascript:)/i.test(href)) return;
    var url;
    try {
      url = new URL(link.href, window.location.href);
    } catch (err) {
      return;
    }
    if (url.origin !== window.location.origin) return;
    if (url.pathname === window.location.pathname && url.search === window.location.search && url.hash) return;
    startNavProgress();
  });

  document.addEventListener("submit", function (e) {
    var form = e.target;
    if (!form || !form.getAttribute || e.defaultPrevented) return;
    if (form.hasAttribute("data-no-progress")) return;
    if (form.target && form.target !== "_self") return;
    startNavProgress();
  });

  function isBackgroundHtmx(evt) {
    var elt = evt.detail && evt.detail.elt;
    if (!elt || !elt.getAttribute) return false;
    if (elt.hasAttribute("data-no-progress")) return true;
    var trigger = (elt.getAttribute("hx-trigger") || "").trim();
    return trigger === "load" || trigger === "revealed" || /\bevery\b/.test(trigger);
  }

  function setHtmxTargetBusy(evt, busy) {
    var target = evt.detail && evt.detail.target;
    if (!target || !target.setAttribute) return;
    if (busy) {
      target.setAttribute("aria-busy", "true");
      target.classList.add("is-htmx-busy");
    } else {
      target.removeAttribute("aria-busy");
      target.classList.remove("is-htmx-busy");
    }
  }

  document.addEventListener("htmx:beforeRequest", function (e) {
    if (isBackgroundHtmx(e)) return;
    startNavProgress();
    setHtmxTargetBusy(e, true);
  });
  ["htmx:afterRequest", "htmx:responseError", "htmx:sendError", "htmx:timeout", "htmx:abort"].forEach(function (name) {
    document.addEventListener(name, function (e) {
      finishNavProgress();
      setHtmxTargetBusy(e, false);
    });
  });
  document.addEventListener("htmx:historyRestore", finishNavProgress);
  window.addEventListener("pageshow", finishNavProgress);

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

    // Close open <details class="user-menu"> or <details class="info-bubble"> when clicking outside
    document.querySelectorAll("details.user-menu[open], details.info-bubble[open]").forEach(function (menu) {
      if (!menu.contains(e.target)) {
        menu.removeAttribute("open");
      }
    });
  });

  // Ensure info bubble popovers never clip outside the viewport
  document.addEventListener("toggle", function (e) {
    if (e.target && e.target.matches && e.target.matches("details.info-bubble") && e.target.open) {
      var content = e.target.querySelector(".info-bubble-content");
      if (content) {
        var rect = content.getBoundingClientRect();
        var margin = 12;
        if (rect.right > window.innerWidth - margin) {
          var overflow = rect.right - (window.innerWidth - margin);
          content.style.transform = "translateX(-" + overflow + "px)";
        } else if (rect.left < margin) {
          var underflow = margin - rect.left;
          content.style.transform = "translateX(" + underflow + "px)";
        } else {
          content.style.transform = "";
        }
      }
    }
  }, true);

  document.addEventListener("keydown", function (e) {
    if (e.key === "Escape") {
      setDrawerOpen(false);
      document.querySelectorAll("details.user-menu[open], details.info-bubble[open]").forEach(function (m) {
        m.removeAttribute("open");
      });
    }
  });

  // Clicks on the ::backdrop report the dialog itself as the target, outside
  // its own box. Native Esc handling stays with the browser.
  document.addEventListener("click", function (e) {
    var dialog = e.target;
    if (!dialog || dialog.tagName !== "DIALOG" || !dialog.classList.contains("modal")) return;
    var rect = dialog.getBoundingClientRect();
    var outside = e.clientX < rect.left || e.clientX > rect.right || e.clientY < rect.top || e.clientY > rect.bottom;
    if (outside) closeModal(dialog);
  });

  document.addEventListener("DOMContentLoaded", function () {
    document.querySelectorAll("dialog.modal").forEach(function (dialog) {
      dialog.addEventListener("close", function () { stopModalTimers(dialog); });
    });
    focusFormError();
    initCbtKiosk();
    initRewindSlides();
    initMysteryOrb();
    initMobileCards();
  });

  // After a rejected submit the message is announced by role="alert"; moving
  // focus there also tells keyboard users why the form did not go through.
  function focusFormError() {
    var error = document.querySelector(".alert-danger[role='alert']");
    if (!error || error.hasAttribute("data-error-focused")) return;
    error.setAttribute("data-error-focused", "");
    error.setAttribute("tabindex", "-1");
    error.focus({ preventScroll: false });
  }

  document.addEventListener("htmx:afterSwap", function () {
    focusFormError();
    initCbtKiosk();
    initRewindSlides();
    initMysteryOrb();
    initMobileCards();
  });
})();
