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

  document.addEventListener("keydown", function (e) {
    if (e.key === "Escape") {
      setDrawerOpen(false);
      document.querySelectorAll(".modal:not([hidden])").forEach(closeModal);
      document.querySelectorAll("details.user-menu[open], details.info-bubble[open]").forEach(function (m) {
        m.removeAttribute("open");
      });
    }
  });

  document.addEventListener("DOMContentLoaded", function () {
    initCbtKiosk();
    initRewindSlides();
    initMysteryOrb();
  });

  document.addEventListener("htmx:afterSwap", function () {
    initCbtKiosk();
    initRewindSlides();
    initMysteryOrb();
  });
})();
