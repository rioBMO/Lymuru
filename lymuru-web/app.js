(function () {
  "use strict";

  // ── API base URL ──────────────────────────────────────────────────
  // When served via the backend (port 8000) or nginx (port 3000),
  // API calls are relative (same origin or proxied).
  var API_BASE = "";

  // ── Auth helpers ──────────────────────────────────────────────────
  var AUTH_STORAGE_KEY = "lymuru_api_token";

  function getAuthToken() {
    try {
      return localStorage.getItem(AUTH_STORAGE_KEY) || "";
    } catch (_) {
      return "";
    }
  }

  function setAuthToken(token) {
    try {
      localStorage.setItem(AUTH_STORAGE_KEY, token);
    } catch (_) {}
  }

  function clearAuthToken() {
    try {
      localStorage.removeItem(AUTH_STORAGE_KEY);
    } catch (_) {}
  }

  function getAuthHeaders() {
    var token = getAuthToken();
    if (token) {
      return { "Authorization": "Bearer " + token };
    }
    return {};
  }

  // ── Loading animation frames (1.png – 8.png looping) ─────────────
  var FRAME_PATHS = [1, 2, 3, 4, 5, 6, 7, 8].map(function (n) {
    return "assets/image/" + n + ".png";
  });
  var _frameIndex = 0;
  var _frameTimers = [];

  function startFrameAnimation(imgElement) {
    _frameIndex = 0;
    imgElement.src = FRAME_PATHS[0];
    var timer = setInterval(function () {
      _frameIndex = (_frameIndex + 1) % FRAME_PATHS.length;
      imgElement.src = FRAME_PATHS[_frameIndex];
    }, 250);
    _frameTimers.push(timer);
    return timer;
  }

  function stopAllFrameAnimations() {
    _frameTimers.forEach(function (t) { clearInterval(t); });
    _frameTimers = [];
  }

  // ── Anime.js wrapper ──────────────────────────────────────────────
  function animate(opts) {
    if (window.anime) {
      window.anime(opts);
    }
  }

  // ── Toast notifications ───────────────────────────────────────────
  var toast = document.getElementById("toast");

  function showToast(message, variant) {
    if (!toast) return;
    toast.textContent = message;
    toast.hidden = false;
    toast.className = "toast" + (variant ? " " + variant : "");

    animate({
      targets: toast,
      translateY: [-18, 0],
      opacity: [0, 1],
      duration: 360,
      easing: "easeOutCubic",
    });

    clearTimeout(showToast._timer);
    showToast._timer = setTimeout(function () {
      animate({
        targets: toast,
        translateY: [0, -18],
        opacity: [1, 0],
        duration: 280,
        easing: "easeInCubic",
        complete: function () { toast.hidden = true; },
      });
    }, 3500);
  }

  // ── Link status message ─────────────────────────────────────────
  var linkStatus = document.getElementById("link-status");

  function setLinkStatus(message, variant) {
    if (!linkStatus) return;
    if (!message) {
      linkStatus.hidden = true;
      linkStatus.textContent = "";
      linkStatus.className = "link-status";
      return;
    }
    linkStatus.hidden = false;
    linkStatus.textContent = message;
    linkStatus.className = "link-status" + (variant ? " " + variant : "");
  }

  // ── API fetch helper ──────────────────────────────────────────────
  async function apiFetch(path, opts) {
    var options = opts || {};
    options.headers = Object.assign({}, options.headers || {}, getAuthHeaders());
    var resp = await fetch(API_BASE + path, options);
    if (resp.status === 401) {
      clearAuthToken();
      showLoginOverlay();
      throw new Error("Authentication required — enter your API token");
    }
    if (!resp.ok) {
      var errBody;
      try { errBody = await resp.json(); } catch (_) { errBody = {}; }
      throw new Error(errBody.detail || "Request failed (" + resp.status + ")");
    }
    return resp.json();
  }

  // ── Tab navigation ────────────────────────────────────────────────
  var navItems = document.querySelectorAll("[data-tab]");
  var panels = document.querySelectorAll("[data-panel]");

  function switchTab(tabName) {
    navItems.forEach(function (btn) {
      var isActive = btn.getAttribute("data-tab") === tabName;
      btn.classList.toggle("active", isActive);
      if (btn.hasAttribute("aria-selected")) {
        btn.setAttribute("aria-selected", isActive ? "true" : "false");
      }
    });

    panels.forEach(function (panel) {
      panel.classList.toggle("active", panel.getAttribute("data-panel") === tabName);
    });
  }

  navItems.forEach(function (btn) {
    btn.addEventListener("click", function () {
      switchTab(btn.getAttribute("data-tab"));

      // Scale click animation
      animate({
        targets: btn,
        scale: [1, 0.95, 1],
        duration: 200,
        easing: "easeOutCubic",
      });
    });
  });

  // ── Telegram status ───────────────────────────────────────────────
  async function loadTelegramStatus() {
    try {
      var resp = await fetch(API_BASE + "/api/telegram/status", { headers: getAuthHeaders() });
      if (resp.status === 401) {
        clearAuthToken();
        updateTelegramUI({ authorized: false, message: "Auth required" });
        return;
      }
      if (!resp.ok) throw new Error("HTTP " + resp.status);
      var status = await resp.json();
      updateTelegramUI(status);
    } catch (err) {
      updateTelegramUI({ authorized: false, message: "Backend offline" });
    }
  }

  function updateTelegramUI(status) {
    var authorized = status && status.authorized;
    var message = (status && status.message) || "Backend offline";

    // Truncate long error messages for the sidebar
    if (message.length > 40) {
      if (message.indexOf("No Telegram session") !== -1) {
        message = "No session found";
      } else if (message.indexOf("expired") !== -1) {
        message = "Session expired";
      } else {
        message = message.substring(0, 37) + "…";
      }
    }

    var dot = document.querySelector("[data-sidebar-dot]");
    var statusEl = document.querySelector("[data-telegram-status]");
    var phoneEl = document.querySelector("[data-telegram-phone]");

    if (dot) dot.classList.toggle("connected", authorized);
    if (statusEl) statusEl.textContent = authorized ? "Connected" : message;
    if (phoneEl) phoneEl.textContent = authorized ? "Session active" : "Run deezload.py first";
  }

  loadTelegramStatus();

  // ── Drop zone handling ────────────────────────────────────────────
  function setupDropZone(zoneId, fileInputId, filenameId, onFile) {
    var zone = document.getElementById(zoneId);
    var input = document.getElementById(fileInputId);
    var nameEl = document.getElementById(filenameId);
    if (!zone || !input) return;

    zone.addEventListener("click", function () { input.click(); });

    zone.addEventListener("dragover", function (e) {
      e.preventDefault();
      zone.classList.add("drag-over");
    });

    zone.addEventListener("dragleave", function () {
      zone.classList.remove("drag-over");
    });

    zone.addEventListener("drop", function (e) {
      e.preventDefault();
      zone.classList.remove("drag-over");
      if (e.dataTransfer.files.length) {
        input.files = e.dataTransfer.files;
        handleFile(e.dataTransfer.files[0]);
      }
    });

    input.addEventListener("change", function () {
      if (input.files.length) handleFile(input.files[0]);
    });

    function handleFile(file) {
      zone.classList.add("has-file");
      if (nameEl) nameEl.textContent = file.name;
      if (onFile) onFile(file);
    }
  }

  // ── Loading animation helpers ─────────────────────────────────────
  function showLoading(id) {
    var el = document.getElementById(id);
    if (el) {
      el.hidden = false;
      var img = el.querySelector(".loading-mascot");
      if (img) startFrameAnimation(img);
    }
  }

  function hideLoading(id) {
    var el = document.getElementById(id);
    if (el) el.hidden = true;
    stopAllFrameAnimations();
  }

  // ── Task progress helpers ──────────────────────────────────────────
  function startTaskProgress(regionId) {
    var region = document.getElementById(regionId);
    if (!region) return { setPhase: function(){}, hide: function(){} };

    region.hidden = false;

    var mascot = region.querySelector("[data-task-mascot]");
    var stageEl = region.querySelector("[data-task-stage]");
    var dlBar = region.querySelector("[data-download-bar]");
    var dlFill = region.querySelector("[data-dl-fill]");
    var dlPercent = region.querySelector("[data-dl-percent]");
    var dlSize = region.querySelector("[data-dl-size]");

    if (mascot) startFrameAnimation(mascot);
    if (stageEl) stageEl.textContent = "Preparing…";
    if (dlBar) dlBar.hidden = true;
    if (dlFill) dlFill.style.width = "0%";

    return {
      setPhase: function (status) {
        // Update stage text
        if (stageEl) stageEl.textContent = status.stage || "Processing…";

        if (status.phase === "downloading") {
          // Show the download bar
          if (dlBar) dlBar.hidden = false;
          var pct = status.download_percent || 0;
          if (dlFill) dlFill.style.width = pct + "%";
          if (dlPercent) dlPercent.textContent = Math.round(pct) + "%";
          if (dlSize) {
            var recv = status.download_received || 0;
            var total = status.download_total || 0;
            dlSize.textContent = formatSize(recv) + " / " + formatSize(total);
          }
        } else {
          // Hide the download bar for preparing / finalizing phases
          if (dlBar) dlBar.hidden = true;
        }
      },
      hide: function () {
        region.hidden = true;
        stopAllFrameAnimations();
      },
    };
  }

  function formatSize(bytes) {
    if (bytes < 1024) return bytes + " B";
    if (bytes < 1048576) return (bytes / 1024).toFixed(1) + " KB";
    return (bytes / 1048576).toFixed(1) + " MB";
  }

  // ── Task progress polling ─────────────────────────────────────────
  async function pollTaskProgress(taskId, ctrl, onDone, onChoiceNeeded, onError) {
    var maxPolls = 300; // 5 minutes max
    var interval = 600;

    for (var i = 0; i < maxPolls; i++) {
      try {
        var status = await apiFetch("/api/downloads/progress/" + taskId);
        ctrl.setPhase(status);

        // Backend is waiting for the user to choose lyrics format
        if (status.waiting_for_choice && status.has_romanized) {
          ctrl.hide();
          if (onChoiceNeeded) onChoiceNeeded(taskId);
          return;
        }

        if (status.done) {
          if (status.error) {
            showToast(status.error, "error");
            if (onError) onError(status.error);
            ctrl.hide();
            return;
          }
          ctrl.hide();
          if (onDone) {
            var result = await apiFetch("/api/task-files/" + taskId);
            onDone(result.files);
          }
          return;
        }
      } catch (err) {
        // ignore transient errors
      }

      await new Promise(function (r) { setTimeout(r, interval); });
    }

    showToast("Task timed out", "error");
    if (onError) onError("Task timed out");
    ctrl.hide();
  }

  // ── Show download files ───────────────────────────────────────────
  function showDownloadFiles(containerId, files) {
    var container = document.getElementById(containerId);
    if (!container) return;

    var parentCard = container.closest(".download-complete");
    if (parentCard) parentCard.hidden = false;

    container.innerHTML = "";
    files.forEach(function (f) {
      var a = document.createElement("a");
      a.className = "download-file-btn";
      a.href = API_BASE + f.url;
      a.download = f.filename;
      a.innerHTML =
        '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>' +
        f.filename + " (" + formatSize(f.size) + ")";
      container.appendChild(a);
    });

    // Animate in
    animate({
      targets: parentCard,
      opacity: [0, 1],
      translateY: [16, 0],
      duration: 400,
      easing: "easeOutCubic",
    });
  }

  function formatSize(bytes) {
    if (bytes < 1024) return bytes + " B";
    if (bytes < 1048576) return (bytes / 1024).toFixed(1) + " KB";
    return (bytes / 1048576).toFixed(1) + " MB";
  }

  // ══════════════════════════════════════════════════════════════════
  // Tab 1: Search & Download
  // ══════════════════════════════════════════════════════════════════
  var formSearch = document.getElementById("form-search");
  var resultsDiv = document.getElementById("search-results");
  var resultsList = document.getElementById("results-list");
  var _currentSearchKey = "";

  if (formSearch) {
    formSearch.addEventListener("submit", async function (e) {
      e.preventDefault();
      var artist = formSearch.elements.artist.value.trim();
      var title = formSearch.elements.title.value.trim();
      if (!artist || !title) {
        formSearch.classList.add("shake");
        setTimeout(function () { formSearch.classList.remove("shake"); }, 500);
        return;
      }

      // Reset states
      resultsDiv.hidden = true;
      document.getElementById("progress-search").hidden = true;
      document.getElementById("download-complete-search").hidden = true;

      showLoading("loading-search");
      document.getElementById("loading-text-search").textContent = "Searching…";

      try {
        var fd = new FormData();
        fd.append("artist", artist);
        fd.append("title", title);

        var data = await apiFetch("/api/search", { method: "POST", body: fd });
        hideLoading("loading-search");

        if (!data.results || data.results.length === 0) {
          showToast("No results found", "error");
          return;
        }

        _currentSearchKey = data.search_key;
        renderResults(data.results, artist, title);
      } catch (err) {
        hideLoading("loading-search");
        showToast(err.message, "error");
      }
    });
  }

  function renderResults(results, artist, title) {
    resultsList.innerHTML = "";
    results.forEach(function (r) {
      var card = document.createElement("div");
      card.className = "result-card";
      card.innerHTML =
        '<div class="result-index">' + (r.index + 1) + "</div>" +
        '<div class="result-info">' +
          '<div class="result-title">' + escapeHtml(r.title) + "</div>" +
          '<div class="result-desc">' + escapeHtml(r.description) + "</div>" +
        "</div>" +
        '<button class="result-action" type="button">Download</button>';

      card.querySelector(".result-action").addEventListener("click", function () {
        downloadChoice(r.index, artist, title);
      });

      resultsList.appendChild(card);
    });

    resultsDiv.hidden = false;

    // Animate results in
    animate({
      targets: ".result-card",
      opacity: [0, 1],
      translateY: [12, 0],
      delay: window.anime ? window.anime.stagger(60) : 0,
      duration: 300,
      easing: "easeOutCubic",
    });
  }

  async function downloadChoice(index, artist, title) {
    resultsDiv.hidden = true;
    document.getElementById("lyrics-choice-search").hidden = true;
    document.getElementById("download-complete-search").hidden = true;
    var ctrl = startTaskProgress("progress-search");

    try {
      var fd = new FormData();
      fd.append("search_key", _currentSearchKey);
      fd.append("choice", index);
      fd.append("artist", artist);
      fd.append("title", title);

      var resp = await apiFetch("/api/downloads/choose", { method: "POST", body: fd });
      await pollTaskProgress(resp.task_id, ctrl, function (files) {
        document.getElementById("progress-search").hidden = true;
        showDownloadFiles("download-files-search", files);
      }, function (taskId) {
        // Romanized lyrics available — show choice UI
        document.getElementById("progress-search").hidden = true;
        showLyricsChoice("search", taskId);
      });
    } catch (err) {
      showToast(err.message, "error");
      document.getElementById("progress-search").hidden = true;
    }
  }

  // ══════════════════════════════════════════════════════════════════
  // Tab 2: Add Lyrics
  // ══════════════════════════════════════════════════════════════════
  var formAddLyrics = document.getElementById("form-addlyrics");

  setupDropZone("drop-addlyrics", "file-addlyrics", "filename-addlyrics", function () {
    document.getElementById("btn-addlyrics").disabled = false;
    document.getElementById("meta-override-addlyrics").hidden = false;
  });

  if (formAddLyrics) {
    formAddLyrics.addEventListener("submit", async function (e) {
      e.preventDefault();
      var fileInput = document.getElementById("file-addlyrics");
      if (!fileInput.files.length) return;

      document.getElementById("download-complete-addlyrics").hidden = true;
      document.getElementById("lyrics-choice-addlyrics").hidden = true;

      var fd = new FormData();
      fd.append("file", fileInput.files[0]);
      fd.append("artist", formAddLyrics.elements.artist.value.trim());
      fd.append("title", formAddLyrics.elements.title.value.trim());

      var ctrl = startTaskProgress("progress-addlyrics");

      try {
        var resp = await apiFetch("/api/lyrics/add", { method: "POST", body: fd });
        await pollTaskProgress(resp.task_id, ctrl, function (files) {
          document.getElementById("progress-addlyrics").hidden = true;
          showDownloadFiles("download-files-addlyrics", files);
        }, function (taskId) {
          document.getElementById("progress-addlyrics").hidden = true;
          showLyricsChoice("addlyrics", taskId);
        });
      } catch (err) {
        showToast(err.message, "error");
        document.getElementById("progress-addlyrics").hidden = true;
      }
    });
  }

  // ══════════════════════════════════════════════════════════════════
  // Tab 3: Embed LRC
  // ══════════════════════════════════════════════════════════════════
  var formEmbed = document.getElementById("form-embed");
  var embedFlacReady = false;
  var embedLrcReady = false;

  setupDropZone("drop-embed-flac", "file-embed-flac", "filename-embed-flac", function () {
    embedFlacReady = true;
    if (embedFlacReady && embedLrcReady) document.getElementById("btn-embed").disabled = false;
  });

  setupDropZone("drop-embed-lrc", "file-embed-lrc", "filename-embed-lrc", function () {
    embedLrcReady = true;
    if (embedFlacReady && embedLrcReady) document.getElementById("btn-embed").disabled = false;
  });

  if (formEmbed) {
    formEmbed.addEventListener("submit", async function (e) {
      e.preventDefault();
      var flacInput = document.getElementById("file-embed-flac");
      var lrcInput = document.getElementById("file-embed-lrc");
      if (!flacInput.files.length || !lrcInput.files.length) return;

      showLoading("loading-embed");

      try {
        var fd = new FormData();
        fd.append("flac_file", flacInput.files[0]);
        fd.append("lrc_file", lrcInput.files[0]);

        var resp = await fetch(API_BASE + "/api/lyrics/embed", { method: "POST", body: fd, headers: getAuthHeaders() });
        hideLoading("loading-embed");

        if (resp.status === 401) {
          clearAuthToken();
          showLoginOverlay();
          showToast("Authentication required — enter your API token", "error");
          return;
        }
        if (!resp.ok) {
          var err = await resp.json();
          showToast(err.detail || "Embed failed", "error");
          return;
        }

        // Download the result
        var blob = await resp.blob();
        var url = URL.createObjectURL(blob);
        var a = document.createElement("a");
        a.href = url;
        a.download = flacInput.files[0].name;
        a.click();
        URL.revokeObjectURL(url);

        showToast("Lyrics embedded successfully!", "success");
      } catch (err) {
        hideLoading("loading-embed");
        showToast(err.message, "error");
      }
    });
  }

  // ══════════════════════════════════════════════════════════════════
  // Tab 4: Romanize LRC
  // ══════════════════════════════════════════════════════════════════
  var formRomanize = document.getElementById("form-romanize");

  setupDropZone("drop-romanize", "file-romanize", "filename-romanize", function () {
    document.getElementById("btn-romanize").disabled = false;
  });

  if (formRomanize) {
    formRomanize.addEventListener("submit", async function (e) {
      e.preventDefault();
      var fileInput = document.getElementById("file-romanize");
      if (!fileInput.files.length) return;

      document.getElementById("preview-romanize").hidden = true;
      showLoading("loading-romanize");

      try {
        var fd = new FormData();
        fd.append("lrc_file", fileInput.files[0]);

        var data = await apiFetch("/api/lyrics/romanize", { method: "POST", body: fd });
        hideLoading("loading-romanize");

        if (!data.romanized) {
          showToast(data.message || "No CJK characters detected", "error");
          return;
        }

        document.getElementById("romanize-content").textContent = data.romanized;
        var dlBtn = document.getElementById("btn-download-romanize");
        dlBtn.href = API_BASE + data.download_url;
        document.getElementById("preview-romanize").hidden = false;

        showToast("Romanization complete!", "success");
      } catch (err) {
        hideLoading("loading-romanize");
        showToast(err.message, "error");
      }
    });
  }

  // ══════════════════════════════════════════════════════════════════
  // Tab 5: Extract LRC
  // ══════════════════════════════════════════════════════════════════
  var formExtract = document.getElementById("form-extract");

  setupDropZone("drop-extract", "file-extract", "filename-extract", function () {
    document.getElementById("btn-extract").disabled = false;
  });

  if (formExtract) {
    formExtract.addEventListener("submit", async function (e) {
      e.preventDefault();
      var fileInput = document.getElementById("file-extract");
      if (!fileInput.files.length) return;

      document.getElementById("preview-extract").hidden = true;
      showLoading("loading-extract");

      try {
        var fd = new FormData();
        fd.append("file", fileInput.files[0]);

        var data = await apiFetch("/api/lyrics/extract", { method: "POST", body: fd });
        hideLoading("loading-extract");

        document.getElementById("extract-content").textContent = data.lyrics;
        var dlBtn = document.getElementById("btn-download-extract");
        dlBtn.href = API_BASE + data.download_url;
        dlBtn.textContent = "Download " + (data.is_synced ? ".lrc" : ".txt");
        document.getElementById("preview-extract").hidden = false;

        showToast("Lyrics extracted!", "success");
      } catch (err) {
        hideLoading("loading-extract");
        showToast(err.message, "error");
      }
    });
  }

  // ══════════════════════════════════════════════════════════════════
  // Tab 6: Download via Link
  // ══════════════════════════════════════════════════════════════════
  var formLink = document.getElementById("form-link");

  if (formLink) {
    formLink.addEventListener("submit", async function (e) {
      e.preventDefault();
      var link = formLink.elements.link.value.trim();
      if (!link) return;

      setLinkStatus("");
      document.getElementById("download-complete-link").hidden = true;
      document.getElementById("lyrics-choice-link").hidden = true;
      var ctrl = startTaskProgress("progress-link");

      try {
        var fd = new FormData();
        fd.append("link", link);

        var resp = await apiFetch("/api/downloads/link", { method: "POST", body: fd });
        await pollTaskProgress(resp.task_id, ctrl, function (files) {
          document.getElementById("progress-link").hidden = true;
          showDownloadFiles("download-files-link", files);
          setLinkStatus("Track retrieved. Download ready.", "success");
        }, function (taskId) {
          document.getElementById("progress-link").hidden = true;
          showLyricsChoice("link", taskId);
        }, function (errMsg) {
          var msg = errMsg || "No track found for this link";
          if (msg.toLowerCase().indexOf("no audio") !== -1 || msg.toLowerCase().indexOf("timed out") !== -1) {
            msg = "No track found for this link";
          }
          setLinkStatus(msg, "error");
        });
      } catch (err) {
        showToast(err.message, "error");
        document.getElementById("progress-link").hidden = true;
        setLinkStatus(err.message, "error");
      }
    });
  }

  // ══════════════════════════════════════════════════════════════════
  // Tab 7: Bulk Download
  // ══════════════════════════════════════════════════════════════════
  var bulkAddBtn = document.getElementById("bulk-add");
  var bulkStartBtn = document.getElementById("bulk-start");
  var bulkList = document.getElementById("bulk-list");
  var bulkCounter = document.getElementById("bulk-counter");
  var bulkModePicker = document.getElementById("bulk-mode-picker");
  var bulkSetup = document.getElementById("bulk-setup");
  var bulkRunner = document.getElementById("bulk-runner");
  var bulkRunnerIndex = document.getElementById("bulk-runner-index");
  var bulkRunnerFound = document.getElementById("bulk-runner-found");
  var bulkRunnerSelected = document.getElementById("bulk-runner-selected");
  var bulkRunnerStage = bulkRunner ? bulkRunner.querySelector("[data-bulk-runner-stage]") : null;
  var bulkRunnerPercent = bulkRunner ? bulkRunner.querySelector("[data-bulk-runner-percent]") : null;
  var bulkRunnerFill = bulkRunner ? bulkRunner.querySelector("[data-bulk-runner-fill]") : null;
  var bulkItems = [];
  var bulkRunning = false;

  function updateBulkCounter() {
    if (bulkCounter) {
      bulkCounter.textContent = bulkItems.length + " / 30";
    }
    if (bulkStartBtn) {
      bulkStartBtn.disabled = bulkItems.length === 0 || bulkRunning;
    }
  }

  function setBulkControlsDisabled(disabled) {
    if (bulkAddBtn) bulkAddBtn.disabled = disabled;
    if (bulkStartBtn) bulkStartBtn.disabled = disabled || bulkItems.length === 0;
    if (bulkModePicker) {
      bulkModePicker.querySelectorAll("button").forEach(function (btn) {
        btn.disabled = disabled;
      });
    }
    if (bulkList) {
      bulkList.querySelectorAll("input, button").forEach(function (el) {
        if (el.classList.contains("download-file-btn")) return;
        el.disabled = disabled;
      });
    }
  }

  function setBulkRunnerVisible(visible) {
    if (bulkSetup) bulkSetup.hidden = visible;
    if (bulkRunner) bulkRunner.hidden = !visible;
  }

  function setBulkRunnerLine(el, value) {
    if (!el) return;
    el.textContent = value || "";
  }

  function updateBulkRunnerFoundFromStage(stageText) {
    if (!stageText || !bulkRunnerFound) return;
    var text = "Finding track…";
    if (stageText.indexOf("Downloading audio") !== -1 || stageText.indexOf("Extracting metadata") !== -1) {
      text = "Track found";
    } else if (stageText.indexOf("Searching lyrics") !== -1 || stageText.indexOf("Embedding") !== -1) {
      text = "Track found";
    } else if (stageText.indexOf("Choose lyrics") !== -1 || stageText.indexOf("Complete") !== -1) {
      text = "Track found";
    } else if (stageText.indexOf("Error") !== -1) {
      text = "Not found";
    }
    bulkRunnerFound.textContent = text;
  }

  function createBulkRunnerProgressCtrl() {
    return {
      reset: function () {
        if (bulkRunnerStage) bulkRunnerStage.textContent = "Preparing…";
        if (bulkRunnerPercent) bulkRunnerPercent.textContent = "0%";
        if (bulkRunnerFill) bulkRunnerFill.style.width = "0%";
      },
      setPhase: function (status) {
        if (bulkRunnerStage) bulkRunnerStage.textContent = status.stage || "Processing…";
        updateBulkRunnerFoundFromStage(status.stage || "");
        if (status.phase === "downloading") {
          var pct = status.download_percent || 0;
          if (bulkRunnerFill) bulkRunnerFill.style.width = pct + "%";
          if (bulkRunnerPercent) bulkRunnerPercent.textContent = Math.round(pct) + "%";
        }
      },
    };
  }

  function createBulkProgressCtrl(itemEl) {
    var progress = itemEl.querySelector("[data-bulk-progress]");
    var stageEl = itemEl.querySelector("[data-bulk-stage]");
    var pctEl = itemEl.querySelector("[data-bulk-percent]");
    var fill = itemEl.querySelector("[data-bulk-fill]");
    if (progress) progress.hidden = false;
    if (fill) fill.style.width = "0%";
    if (pctEl) pctEl.textContent = "0%";
    return {
      setPhase: function (status) {
        if (stageEl) stageEl.textContent = status.stage || "Processing…";
        if (status.phase === "downloading") {
          var pct = status.download_percent || 0;
          if (fill) fill.style.width = pct + "%";
          if (pctEl) pctEl.textContent = Math.round(pct) + "%";
        }
      },
      hide: function () {
        if (progress) progress.hidden = true;
      },
    };
  }

  function showDownloadFilesInto(container, files) {
    if (!container) return;
    container.innerHTML = "";
    files.forEach(function (f) {
      var a = document.createElement("a");
      a.className = "download-file-btn";
      a.href = API_BASE + f.url;
      a.download = f.filename;
      a.innerHTML =
        '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>' +
        f.filename + " (" + formatSize(f.size) + ")";
      container.appendChild(a);
    });
  }

  function createBulkItem(mode) {
    var itemId = "bulk-" + Date.now() + "-" + Math.floor(Math.random() * 10000);
    var el = document.createElement("div");
    el.className = "card bulk-item";
    el.setAttribute("data-bulk-id", itemId);

    var titleLabel = mode === "search" ? "Search" : "Link";
    var headerHtml =
      '<div class="bulk-item-header">' +
        '<div class="bulk-item-title">' + titleLabel + '</div>' +
        '<button type="button" class="bulk-remove">Remove</button>' +
      '</div>';

    var bodyHtml = "";
    if (mode === "search") {
      bodyHtml =
        '<div class="form-row">' +
          '<div class="form-group">' +
            '<label>Artist</label>' +
            '<input type="text" class="bulk-artist" placeholder="e.g. Kenshi Yonezu" autocomplete="off">' +
          '</div>' +
          '<div class="form-group">' +
            '<label>Track Title</label>' +
            '<input type="text" class="bulk-title" placeholder="e.g. KICK BACK" autocomplete="off">' +
          '</div>' +
        '</div>' +
        '<button type="button" class="btn btn-secondary bulk-search-btn">Search</button>' +
        '<div class="bulk-selected" hidden></div>' +
        '<div class="bulk-results" hidden>' +
          '<h4>Results</h4>' +
          '<div class="results-list"></div>' +
        '</div>';
    } else {
      bodyHtml =
        '<div class="form-group">' +
          '<label>Spotify / Deezer URL</label>' +
          '<input type="url" class="bulk-link" placeholder="https://open.spotify.com/track/..." autocomplete="off">' +
        '</div>';
    }

    bodyHtml +=
      '<div class="bulk-status" data-bulk-status>Idle</div>' +
      '<div class="download-bar-section bulk-progress" data-bulk-progress hidden>' +
        '<div class="progress-info">' +
          '<span class="progress-stage" data-bulk-stage>Preparing…</span>' +
          '<span class="progress-percent" data-bulk-percent>0%</span>' +
        '</div>' +
        '<div class="progress-bar">' +
          '<div class="progress-fill" data-bulk-fill></div>' +
        '</div>' +
      '</div>' +
      '<div class="download-files" data-bulk-files></div>';

    el.innerHTML = headerHtml + bodyHtml;
    return el;
  }

  function setBulkStatus(itemEl, message, variant) {
    var statusEl = itemEl.querySelector("[data-bulk-status]");
    if (!statusEl) return;
    statusEl.textContent = message || "";
    statusEl.className = "bulk-status" + (variant ? " " + variant : "");
  }

  if (bulkAddBtn) {
    bulkAddBtn.addEventListener("click", function () {
      if (bulkItems.length >= 30) {
        showToast("Bulk list is limited to 30 items", "error");
        return;
      }
      if (bulkModePicker) {
        bulkModePicker.hidden = !bulkModePicker.hidden;
      }
    });
  }

  if (bulkModePicker) {
    bulkModePicker.addEventListener("click", function (e) {
      var btn = e.target.closest("[data-bulk-mode]");
      if (!btn || !bulkList) return;
      var mode = btn.getAttribute("data-bulk-mode");
      if (bulkItems.length >= 30) {
        showToast("Bulk list is limited to 30 items", "error");
        return;
      }

      var itemEl = createBulkItem(mode);
      bulkList.prepend(itemEl);

      var item = {
        id: itemEl.getAttribute("data-bulk-id"),
        mode: mode,
        element: itemEl,
        selected: null,
        link: "",
      };
      bulkItems.unshift(item);
      updateBulkCounter();

      itemEl.querySelector(".bulk-remove").addEventListener("click", function () {
        bulkItems = bulkItems.filter(function (i) { return i.id !== item.id; });
        itemEl.remove();
        updateBulkCounter();
      });

      if (mode === "search") {
        var searchBtn = itemEl.querySelector(".bulk-search-btn");
        var artistInput = itemEl.querySelector(".bulk-artist");
        var titleInput = itemEl.querySelector(".bulk-title");
        var resultsBox = itemEl.querySelector(".bulk-results");
        var resultsList = itemEl.querySelector(".results-list");
        var selectedBox = itemEl.querySelector(".bulk-selected");

        searchBtn.addEventListener("click", async function () {
          var artist = artistInput.value.trim();
          var title = titleInput.value.trim();
          if (!artist || !title) {
            itemEl.classList.add("shake");
            setTimeout(function () { itemEl.classList.remove("shake"); }, 500);
            return;
          }

          setBulkStatus(itemEl, "Searching…");
          if (resultsBox) resultsBox.hidden = true;
          if (selectedBox) selectedBox.hidden = true;
          item.selected = null;

          try {
            var fd = new FormData();
            fd.append("artist", artist);
            fd.append("title", title);

            var data = await apiFetch("/api/search", { method: "POST", body: fd });
            if (!data.results || data.results.length === 0) {
              setBulkStatus(itemEl, "No results found", "error");
              return;
            }

            resultsList.innerHTML = "";
            data.results.forEach(function (r) {
              var card = document.createElement("div");
              card.className = "result-card";
              card.innerHTML =
                '<div class="result-index">' + (r.index + 1) + "</div>" +
                '<div class="result-info">' +
                  '<div class="result-title">' + escapeHtml(r.title) + "</div>" +
                  '<div class="result-desc">' + escapeHtml(r.description) + "</div>" +
                "</div>" +
                '<button class="result-action" type="button">Select</button>';

              card.querySelector(".result-action").addEventListener("click", function () {
                item.selected = {
                  search_key: data.search_key,
                  choice: r.index,
                  artist: artist,
                  title: title,
                  label: r.title,
                };
                if (selectedBox) {
                  selectedBox.textContent = "Selected: " + r.title;
                  selectedBox.hidden = false;
                }
                setBulkStatus(itemEl, "Ready", "success");
              });

              resultsList.appendChild(card);
            });

            if (resultsBox) resultsBox.hidden = false;
            setBulkStatus(itemEl, "Select a result to queue", "");
          } catch (err) {
            setBulkStatus(itemEl, err.message, "error");
          }
        });
      } else {
        var linkInput = itemEl.querySelector(".bulk-link");
        linkInput.addEventListener("input", function () {
          item.link = linkInput.value.trim();
          if (item.link) {
            setBulkStatus(itemEl, "Ready", "success");
          } else {
            setBulkStatus(itemEl, "Idle");
          }
        });
      }

      if (bulkModePicker) bulkModePicker.hidden = true;
    });
  }

  async function runBulkQueue() {
    if (bulkRunning || bulkItems.length === 0) return;

    for (var i = 0; i < bulkItems.length; i++) {
      var item = bulkItems[i];
      if (item.mode === "search" && !item.selected) {
        setBulkStatus(item.element, "Pick a search result first", "error");
        showToast("Select results for all search items before starting", "error");
        return;
      }
      if (item.mode === "link" && !item.link) {
        setBulkStatus(item.element, "Add a link first", "error");
        showToast("Fill in all link items before starting", "error");
        return;
      }
    }

    bulkRunning = true;
    setBulkControlsDisabled(true);
    setBulkRunnerVisible(true);
    updateBulkCounter();

    var runnerCtrl = createBulkRunnerProgressCtrl();
    runnerCtrl.reset();

    for (var j = 0; j < bulkItems.length; j++) {
      var current = bulkItems[j];
      var itemEl = current.element;
      var filesContainer = itemEl.querySelector("[data-bulk-files]");
      filesContainer.innerHTML = "";

      setBulkRunnerLine(bulkRunnerIndex, (j + 1) + " / " + bulkItems.length);
      setBulkRunnerLine(bulkRunnerFound, "Finding track…");
      if (current.mode === "search") {
        setBulkRunnerLine(bulkRunnerSelected, current.selected ? current.selected.label : "—");
      } else {
        setBulkRunnerLine(bulkRunnerSelected, "Waiting for track info");
      }
      runnerCtrl.reset();

      setBulkStatus(itemEl, "Starting…");
      var itemCtrl = createBulkProgressCtrl(itemEl);
      var ctrl = {
        setPhase: function (status) {
          itemCtrl.setPhase(status);
          runnerCtrl.setPhase(status);
        },
        hide: function () {
          itemCtrl.hide();
        },
      };

      try {
        var fd = new FormData();
        var endpoint = "";
        if (current.mode === "search") {
          fd.append("search_key", current.selected.search_key);
          fd.append("choice", current.selected.choice);
          fd.append("artist", current.selected.artist);
          fd.append("title", current.selected.title);
          endpoint = "/api/downloads/choose";
        } else {
          fd.append("link", current.link);
          endpoint = "/api/downloads/link";
        }

        var resp = await apiFetch(endpoint, { method: "POST", body: fd });

        await pollTaskProgress(resp.task_id, ctrl, function (files) {
          ctrl.hide();
          showDownloadFilesInto(filesContainer, files);
          setBulkStatus(itemEl, "Complete", "success");
          if (files && files.length) {
            setBulkRunnerLine(bulkRunnerSelected, files[0].filename);
          }
          setBulkRunnerLine(bulkRunnerFound, "Track found");
        }, async function (taskId) {
          try {
            var chooseFd = new FormData();
            chooseFd.append("task_id", taskId);
            chooseFd.append("lyrics_choice", "original");
            await apiFetch("/api/downloads/choose-lyrics", { method: "POST", body: chooseFd });
            var result = await apiFetch("/api/task-files/" + taskId);
            ctrl.hide();
            showDownloadFilesInto(filesContainer, result.files);
            setBulkStatus(itemEl, "Complete (original lyrics)", "success");
            if (result.files && result.files.length) {
              setBulkRunnerLine(bulkRunnerSelected, result.files[0].filename);
            }
            setBulkRunnerLine(bulkRunnerFound, "Track found");
          } catch (err) {
            ctrl.hide();
            setBulkStatus(itemEl, err.message, "error");
            setBulkRunnerLine(bulkRunnerFound, "Not found");
          }
        }, function (errMsg) {
          ctrl.hide();
          var msg = errMsg || "Download failed";
          if (current.mode === "link") {
            if (msg.toLowerCase().indexOf("no audio") !== -1 || msg.toLowerCase().indexOf("timed out") !== -1) {
              msg = "No track found for this link";
            }
          }
          setBulkStatus(itemEl, msg, "error");
          setBulkRunnerLine(bulkRunnerFound, "Not found");
        });
      } catch (err) {
        ctrl.hide();
        setBulkStatus(itemEl, err.message, "error");
        setBulkRunnerLine(bulkRunnerFound, "Not found");
      }
    }

    bulkRunning = false;
    setBulkControlsDisabled(false);
    setBulkRunnerVisible(false);
    updateBulkCounter();
  }

  if (bulkStartBtn) {
    bulkStartBtn.addEventListener("click", function () {
      runBulkQueue();
    });
  }

  // ══════════════════════════════════════════════════════════════════
  // Lyrics Choice — shared logic for Tab 1, 2, and 6
  // ══════════════════════════════════════════════════════════════════

  function showLyricsChoice(suffix, taskId) {
    var container = document.getElementById("lyrics-choice-" + suffix);
    if (!container) return;

    // Reset selection to "original"
    container.querySelectorAll(".lyrics-option").forEach(function (btn) {
      btn.classList.toggle("selected", btn.getAttribute("data-choice") === "original");
    });

    container.hidden = false;
    container._taskId = taskId;

    // Animate in
    animate({
      targets: container,
      opacity: [0, 1],
      translateY: [16, 0],
      duration: 400,
      easing: "easeOutCubic",
    });
  }

  // Wire up option toggle buttons for all three lyrics-choice cards
  ["search", "addlyrics", "link"].forEach(function (suffix) {
    var container = document.getElementById("lyrics-choice-" + suffix);
    if (!container) return;

    // Option selection toggle
    container.querySelectorAll(".lyrics-option").forEach(function (btn) {
      btn.addEventListener("click", function () {
        container.querySelectorAll(".lyrics-option").forEach(function (b) {
          b.classList.remove("selected");
        });
        btn.classList.add("selected");

        animate({
          targets: btn,
          scale: [1, 0.95, 1],
          duration: 200,
          easing: "easeOutCubic",
        });
      });
    });

    // Confirm button
    var confirmBtn = document.getElementById("btn-confirm-lyrics-" + suffix);
    if (confirmBtn) {
      confirmBtn.addEventListener("click", async function () {
        var selected = container.querySelector(".lyrics-option.selected");
        var choice = selected ? selected.getAttribute("data-choice") : "original";
        var taskId = container._taskId;

        confirmBtn.disabled = true;
        confirmBtn.textContent = "Embedding…";

        try {
          var fd = new FormData();
          fd.append("task_id", taskId);
          fd.append("lyrics_choice", choice);

          await apiFetch("/api/downloads/choose-lyrics", { method: "POST", body: fd });
          container.hidden = true;

          // Fetch completed files
          var result = await apiFetch("/api/task-files/" + taskId);
          var filesContainerId = "download-files-" + suffix;
          showDownloadFiles(filesContainerId, result.files);

          showToast(
            choice === "romanized"
              ? "Romanized lyrics embedded!"
              : "Original lyrics embedded!",
            "success"
          );
        } catch (err) {
          showToast(err.message, "error");
        } finally {
          confirmBtn.disabled = false;
          confirmBtn.innerHTML =
            '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg> Confirm \u0026 Embed';
        }
      });
    }
  });

  // ── Auth ─────────────────────────────────────────────────────────
  var loginOverlay = document.getElementById("login-overlay");
  var loginForm = document.getElementById("login-form");
  var loginToken = document.getElementById("login-token");
  var loginError = document.getElementById("login-error");

  function showLoginOverlay() {
    if (loginOverlay) loginOverlay.hidden = false;
    if (loginToken) loginToken.value = "";
    if (loginError) loginError.hidden = true;
  }

  function hideLoginOverlay() {
    if (loginOverlay) loginOverlay.hidden = true;
  }

  async function checkAuth() {
    var token = getAuthToken();
    if (!token) {
      showLoginOverlay();
      return;
    }
    try {
      var resp = await fetch(API_BASE + "/api/auth/check", { headers: getAuthHeaders() });
      if (resp.ok) {
        hideLoginOverlay();
      } else {
        clearAuthToken();
        showLoginOverlay();
      }
    } catch (_) {
      hideLoginOverlay();
    }
  }

  if (loginForm) {
    loginForm.addEventListener("submit", async function (e) {
      e.preventDefault();
      var token = (loginToken.value || "").trim();
      if (!token) return;

      loginError.hidden = true;
      var submitBtn = loginForm.querySelector("button");

      try {
        submitBtn.disabled = true;
        submitBtn.textContent = "Verifying…";

        var headers = { "Authorization": "Bearer " + token };
        var resp = await fetch(API_BASE + "/api/auth/check", { headers: headers });

        if (resp.ok) {
          setAuthToken(token);
          hideLoginOverlay();
          showToast("Authenticated", "success");
          loadTelegramStatus();
        } else {
          loginError.textContent = "Invalid token — try again";
          loginError.hidden = false;
        }
      } catch (_) {
        loginError.textContent = "Could not reach server";
        loginError.hidden = false;
      } finally {
        submitBtn.disabled = false;
        submitBtn.textContent = "Authenticate";
      }
    });
  }

  checkAuth();

  // ── Utility ───────────────────────────────────────────────────────
  function escapeHtml(str) {
    var div = document.createElement("div");
    div.textContent = str;
    return div.innerHTML;
  }

})();
