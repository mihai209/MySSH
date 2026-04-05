(function () {
  const state = {
    profiles: [],
    filtered: [],
    selectedId: "",
    editingId: "",
    terminalVisible: false,
    terminal: null,
    fitAddon: null,
    webglAddon: null,
    webglEnabled: false,
    terminalBuffer: "",
    pendingTerminalWrite: "",
    terminalFlushScheduled: false,
  };

  const els = {};

  document.addEventListener("DOMContentLoaded", init);

  async function init() {
    cacheElements();
    bindEvents();
    await bootstrap();
  }

  function cacheElements() {
    els.pingState = document.getElementById("ping-state");
    els.search = document.getElementById("search");
    els.profileList = document.getElementById("profile-list");
    els.profileCount = document.getElementById("profile-count");
    els.recommendedAuth = document.getElementById("recommended-auth");
    els.securityCopy = document.getElementById("security-copy");
    els.dataDir = document.getElementById("data-dir");
    els.authMix = document.getElementById("auth-mix");
    els.statusText = document.getElementById("status-text");
    els.newProfile = document.getElementById("new-profile");
    els.emptyAddButton = document.getElementById("empty-add-button");
    els.detailTitle = document.getElementById("detail-title");
    els.heroTitle = document.getElementById("hero-title");
    els.heroCopy = document.getElementById("hero-copy");
    els.machineMeta = document.getElementById("machine-meta");
    els.machineName = document.getElementById("machine-name");
    els.machineUsername = document.getElementById("machine-username");
    els.machineHost = document.getElementById("machine-host");
    els.machinePort = document.getElementById("machine-port");
    els.machineAuth = document.getElementById("machine-auth");
    els.machineSecretState = document.getElementById("machine-secret-state");
    els.emptyState = document.getElementById("empty-state");
    els.machineDetail = document.getElementById("machine-detail");
    els.modalBackdrop = document.getElementById("modal-backdrop");
    els.closeModal = document.getElementById("close-modal");
    els.cancelModal = document.getElementById("cancel-modal");
    els.saveProfile = document.getElementById("save-profile");
    els.editProfile = document.getElementById("edit-profile");
    els.connectProfile = document.getElementById("connect-profile");
    els.deleteProfile = document.getElementById("delete-profile");
    els.modalTitle = document.getElementById("modal-title");
    els.modalNote = document.getElementById("modal-note");
    els.name = document.getElementById("name");
    els.username = document.getElementById("username");
    els.host = document.getElementById("host");
    els.port = document.getElementById("port");
    els.authKind = document.getElementById("auth-kind");
    els.keySource = document.getElementById("key-source");
    els.keySourceWrap = document.getElementById("key-source-wrap");
    els.keyPath = document.getElementById("key-path");
    els.keyPathWrap = document.getElementById("key-path-wrap");
    els.secretWrap = document.getElementById("secret-wrap");
    els.keyContentWrap = document.getElementById("key-content-wrap");
    els.keyContent = document.getElementById("key-content");
    els.modalSecurityCopy = document.getElementById("modal-security-copy");
    els.secret = document.getElementById("secret");
    els.terminalScreen = document.getElementById("terminal-screen");
    els.terminalTitle = document.getElementById("terminal-title");
    els.terminalSubtitle = document.getElementById("terminal-subtitle");
    els.terminalStatusChip = document.getElementById("terminal-status-chip");
    els.terminalLoader = document.getElementById("terminal-loader");
    els.terminalLoaderText = document.getElementById("terminal-loader-text");
    els.terminalTrust = document.getElementById("terminal-trust");
    els.terminalTrustCopy = document.getElementById("terminal-trust-copy");
    els.terminalTrustButton = document.getElementById("terminal-trust-button");
    els.terminalContainer = document.getElementById("terminal-container");
    els.terminalGpuToggle = document.getElementById("terminal-gpu-toggle");
    els.terminalCopy = document.getElementById("terminal-copy");
    els.terminalPaste = document.getElementById("terminal-paste");
    els.terminalBack = document.getElementById("terminal-back");
    els.connectSecret = document.getElementById("connect-secret");
  }

  function bindEvents() {
    els.search.addEventListener("input", applyFilter);
    els.newProfile.addEventListener("click", openCreateModal);
    els.emptyAddButton.addEventListener("click", openCreateModal);
    els.closeModal.addEventListener("click", closeModal);
    els.cancelModal.addEventListener("click", closeModal);
    els.saveProfile.addEventListener("click", saveProfile);
    els.editProfile.addEventListener("click", openEditModal);
    els.connectProfile.addEventListener("click", connectProfile);
    els.deleteProfile.addEventListener("click", deleteProfile);
    els.authKind.addEventListener("change", updateSecurityCopy);
    els.keySource.addEventListener("change", updateSecurityCopy);
    els.terminalBack.addEventListener("click", closeTerminal);
    els.terminalTrustButton.addEventListener("click", trustPendingHost);
    els.terminalGpuToggle.addEventListener("click", toggleTerminalGpu);
    els.terminalCopy.addEventListener("click", copyTerminalSelection);
    els.terminalPaste.addEventListener("click", pasteIntoTerminal);
    els.modalBackdrop.addEventListener("click", (event) => {
      if (event.target === els.modalBackdrop) {
        closeModal();
      }
    });

    if (window.runtime?.EventsOn) {
      window.runtime.EventsOn("ssh:output", handleTerminalOutput);
      window.runtime.EventsOn("ssh:status", handleTerminalStatus);
      window.runtime.EventsOn("ssh:hostkey", handleUnknownHostKey);
    }

    window.addEventListener("resize", debounceResizeTerminal);
    updateGpuButton();
  }

  async function bootstrap() {
    try {
      if (window.go?.main?.App?.Ping) {
        els.pingState.textContent = await window.go.main.App.Ping();
      } else {
        els.pingState.textContent = "Wails bindings pending";
      }

      await refreshDashboard();
      renderSelection();
    } catch (error) {
      setStatus(String(error), true);
    }
  }

  async function refreshDashboard() {
    const dashboard = await window.go.main.App.Dashboard();
    state.profiles = dashboard.profiles || [];

    if (state.selectedId && !state.profiles.some((profile) => profile.id === state.selectedId)) {
      state.selectedId = "";
    }

    els.profileCount.textContent = String(state.profiles.length);
    els.recommendedAuth.textContent = dashboard.recommendedAuth || "agent";
    els.dataDir.textContent = dashboard.dataDir || "";
    els.authMix.textContent = `agent ${dashboard.agentCount} | key ${dashboard.keyCount} | password ${dashboard.passwordCount}`;

    applyFilter();
    updateSecurityCopy();
  }

  function applyFilter() {
    const query = els.search.value.trim().toLowerCase();
    state.filtered = state.profiles.filter((profile) =>
      [profile.name, profile.username, profile.host].some((value) =>
        String(value || "").toLowerCase().includes(query),
      ),
    );
    renderProfiles();
  }

  function renderProfiles() {
    els.profileList.innerHTML = "";

    if (!state.filtered.length) {
      const empty = document.createElement("div");
      empty.className = "empty-state";
      empty.innerHTML = `<p>${state.profiles.length ? "No SSH machines match this search." : "No SSH added yet."}</p>`;
      els.profileList.appendChild(empty);
      return;
    }

    state.filtered.forEach((profile) => {
      const card = document.createElement("button");
      card.type = "button";
      card.className = "machine-card";
      if (profile.id === state.selectedId) {
        card.classList.add("active");
      }

      card.innerHTML = `
        <div class="machine-card-title">${escapeHtml(profile.name || "Unnamed host")}</div>
        <div class="machine-card-meta">${escapeHtml(profile.username)}@${escapeHtml(profile.host)}:${profile.port}</div>
        <div class="machine-card-auth">${escapeHtml(profile.authKind)}${profile.keySource ? ` • ${escapeHtml(profile.keySource)}` : ""}</div>
      `;

      card.addEventListener("click", () => {
        state.selectedId = profile.id || "";
        renderSelection();
        renderProfiles();
        setStatus("Machine selected.");
      });

      els.profileList.appendChild(card);
    });
  }

  function renderSelection() {
    const profile = state.profiles.find((item) => item.id === state.selectedId);
    const hasProfiles = state.profiles.length > 0;

    if (!profile) {
      els.machineDetail.classList.add("hidden");
      els.emptyState.classList.toggle("hidden", hasProfiles);
      els.heroTitle.textContent = hasProfiles ? "Pick an SSH machine" : "No SSH selected";
      els.heroCopy.textContent = hasProfiles
        ? "Select a machine from the left or add a new one."
        : "Start by adding an SSH machine. Passwords and pasted private keys are persisted in your OS keyring.";
      els.editProfile.disabled = true;
      els.connectProfile.disabled = true;
      return;
    }

    els.emptyState.classList.add("hidden");
    els.machineDetail.classList.remove("hidden");
    els.detailTitle.textContent = profile.name || "Machine";
    els.heroTitle.textContent = profile.name || "SSH Machine";
    els.heroCopy.textContent = "Connect opens a dedicated terminal workspace with loader, live output and autoreconnect.";
    els.machineMeta.textContent = `${profile.username}@${profile.host}:${profile.port}`;
    els.machineName.textContent = profile.name || "-";
    els.machineUsername.textContent = profile.username || "-";
    els.machineHost.textContent = profile.host || "-";
    els.machinePort.textContent = String(profile.port || 22);
    els.machineAuth.textContent = profile.authKind || "agent";
    els.machineSecretState.textContent = profile.hasStoredSecret ? "stored in OS keyring" : profile.keyPath ? "key path reference" : "none";
    els.editProfile.disabled = false;
    els.connectProfile.disabled = !(profile.authKind === "agent" || profile.authKind === "password" || profile.authKind === "private_key");
  }

  function openCreateModal() {
    state.editingId = "";
    els.modalTitle.textContent = "New SSH";
    els.modalNote.textContent = "Create a new SSH machine profile.";
    els.name.value = "";
    els.username.value = "";
    els.host.value = "";
    els.port.value = 22;
    els.authKind.value = "agent";
    els.keySource.value = "path";
    els.keyPath.value = "";
    els.secret.value = "";
    els.connectSecret.value = "";
    els.connectSecret.placeholder = "Optional remote SECRET stored in OS keyring";
    els.keyContent.value = "";
    updateSecurityCopy();
    els.modalBackdrop.classList.remove("hidden");
  }

  function openEditModal() {
    const profile = state.profiles.find((item) => item.id === state.selectedId);
    if (!profile) {
      return;
    }

    state.editingId = profile.id || "";
    els.modalTitle.textContent = "Edit SSH";
    els.modalNote.textContent = profile.hasStoredSecret
      ? "Leave the secret field empty to keep the existing keyring value."
      : "Update the machine metadata or authentication mode.";
    els.name.value = profile.name || "";
    els.username.value = profile.username || "";
    els.host.value = profile.host || "";
    els.port.value = profile.port || 22;
    els.authKind.value = profile.authKind || "agent";
    els.keySource.value = profile.keySource || "path";
    els.keyPath.value = profile.keyPath || "";
    els.secret.value = "";
    els.connectSecret.value = "";
    els.connectSecret.placeholder = profile.hasConnectSecret
      ? "Stored in OS keyring"
      : "Optional remote SECRET stored in OS keyring";
    els.keyContent.value = "";
    updateSecurityCopy();
    els.modalBackdrop.classList.remove("hidden");
  }

  function closeModal() {
    els.modalBackdrop.classList.add("hidden");
  }

  async function saveProfile() {
    try {
      const payload = {
        id: state.editingId,
        name: els.name.value,
        username: els.username.value,
        host: els.host.value,
        port: Number(els.port.value || 22),
        authKind: els.authKind.value,
        keySource: els.authKind.value === "private_key" ? els.keySource.value : "",
        keyPath: els.authKind.value === "private_key" && els.keySource.value === "path" ? els.keyPath.value : "",
        secretValue: resolveSecretValue(),
        connectSecretValue: els.connectSecret.value,
      };

      const profile = await window.go.main.App.SaveProfile(payload);
      state.selectedId = profile.id;
      state.editingId = "";
      closeModal();
      await refreshDashboard();
      renderSelection();
      setStatus("SSH machine saved.");
    } catch (error) {
      setStatus(String(error), true);
    }
  }

  async function deleteProfile() {
    if (!state.selectedId) {
      return;
    }

    try {
      const profile = state.profiles.find((item) => item.id === state.selectedId);
      if (!profile) {
        return;
      }

      const confirmed = window.confirm(`Delete SSH "${profile.name}"?`);
      if (!confirmed) {
        return;
      }

      await window.go.main.App.DeleteProfile(state.selectedId);
      state.selectedId = "";
      await refreshDashboard();
      renderSelection();
      setStatus("SSH machine deleted.");
    } catch (error) {
      setStatus(String(error), true);
    }
  }

  async function connectProfile() {
    const profile = state.profiles.find((item) => item.id === state.selectedId);
    if (!profile || !(profile.authKind === "agent" || profile.authKind === "password" || profile.authKind === "private_key")) {
      setStatus("Connect is currently available only for password, agent, and private key profiles.");
      return;
    }

    openTerminal(profile);
    setTerminalLoader(true, `Connecting to ${profile.host}...`);
    setTerminalStatus("Connecting");
    appendTerminalOutput(`[MySSH] Connecting to ${profile.username}@${profile.host}:${profile.port}\r\n`);

    try {
      await window.go.main.App.ConnectProfile(profile.id);
    } catch (error) {
      setTerminalLoader(false);
      setTerminalStatus("Error");
      appendTerminalOutput(`[MySSH] Connection error: ${String(error)}\r\n`);
      setStatus(String(error), true);
    }
  }

  function updateSecurityCopy() {
    els.keySourceWrap.classList.add("hidden");
    els.keyPathWrap.classList.add("hidden");
    els.secretWrap.classList.add("hidden");
    els.keyContentWrap.classList.add("hidden");

    switch (els.authKind.value) {
      case "password":
        els.securityCopy.textContent = "Passwords are persisted in your OS keyring, not in profiles.json.";
        els.modalSecurityCopy.textContent = "Passwords are persisted in your OS keyring, not in profiles.json.";
        els.secret.placeholder = "Stored in OS keyring";
        els.secretWrap.classList.remove("hidden");
        break;
      case "private_key":
        els.securityCopy.textContent = "Private key mode supports either a key path reference or pasted key content persisted in your OS keyring.";
        els.keySourceWrap.classList.remove("hidden");
        if (els.keySource.value === "content") {
          els.keyContentWrap.classList.remove("hidden");
          els.modalSecurityCopy.textContent = "Pasted private key content is persisted in your OS keyring.";
        } else {
          els.keyPathWrap.classList.remove("hidden");
          els.modalSecurityCopy.textContent = "Key path mode stores only the filesystem path in the profile metadata.";
        }
        break;
      default:
        els.securityCopy.textContent = "Agent mode is the safest default for the MVP and avoids local secret persistence entirely.";
        els.modalSecurityCopy.textContent = "Agent mode avoids local secret persistence entirely.";
        break;
    }
  }

  function needsSecretValue() {
    return els.authKind.value === "password" || (els.authKind.value === "private_key" && els.keySource.value === "content");
  }

  function resolveSecretValue() {
    if (!needsSecretValue()) {
      return "";
    }

    if (els.authKind.value === "private_key" && els.keySource.value === "content") {
      return els.keyContent.value;
    }

    return els.secret.value;
  }

  function setStatus(message, isError) {
    els.statusText.textContent = message;
    els.statusText.style.color = isError ? "#ff8da0" : "";
  }

  function openTerminal(profile) {
    state.terminalVisible = true;
    state.terminalBuffer = "";
    els.terminalScreen.classList.remove("hidden");
    els.terminalTitle.textContent = profile.name || "SSH Session";
    els.terminalSubtitle.textContent = `${profile.username}@${profile.host}:${profile.port}`;
    els.terminalTrust.classList.add("hidden");
    destroyTerminal();
    ensureTerminal();
    state.terminal.reset();
    state.terminal.focus();
  }

  async function closeTerminal() {
    state.terminalVisible = false;
    els.terminalScreen.classList.add("hidden");
    setTerminalLoader(false);
    els.terminalTrust.classList.add("hidden");
    setTerminalStatus("Idle");
    destroyTerminal();
    try {
      await window.go.main.App.DisconnectTerminal();
    } catch (_) {
      // best effort disconnect
    }
  }

  function handleTerminalOutput(payload) {
    if (!state.terminalVisible) {
      return;
    }
    appendTerminalOutput(payload?.chunk || "");
  }

  function handleTerminalStatus(payload) {
    const message = payload?.message || "SSH status update";
    const status = payload?.state || "Idle";
    const profile = payload?.profile || {};

    setTerminalStatus(status);
    els.terminalTitle.textContent = profile.name || "SSH Session";
    els.terminalSubtitle.textContent = profile.host ? `${profile.username}@${profile.host}:${profile.port}` : "Waiting for connection...";

    if (status === "connecting" || status === "reconnecting") {
      setTerminalLoader(true, message);
    } else {
      setTerminalLoader(false);
    }

    if (state.terminalVisible) {
      appendTerminalOutput(`[MySSH] ${message}\r\n`);
    }
  }

  function handleUnknownHostKey(payload) {
    if (!state.terminalVisible) {
      return;
    }

    els.terminalTrust.classList.remove("hidden");
    els.terminalTrustCopy.textContent = `${payload?.message || "Unknown host key"}\n${payload?.fingerprint || ""}`;
    setTerminalLoader(false);
    setTerminalStatus("Trust Required");
    appendTerminalOutput(`[MySSH] Unknown host key: ${payload?.fingerprint || "unknown"}\r\n`);
  }

  async function trustPendingHost() {
    try {
      await window.go.main.App.TrustPendingHost();
      els.terminalTrust.classList.add("hidden");
      setTerminalStatus("Trusted");
      appendTerminalOutput("[MySSH] Host key trusted. Reconnect now.\r\n");
      await connectProfile();
    } catch (error) {
      appendTerminalOutput(`[MySSH] Trust error: ${String(error)}\r\n`);
    }
  }

  function appendTerminalOutput(chunk) {
    const sanitized = sanitizeTerminalChunk(chunk);
    if (!sanitized) {
      return;
    }
    state.terminalBuffer += sanitized;
    if (state.terminalBuffer.length > 1200000) {
      state.terminalBuffer = state.terminalBuffer.slice(-900000);
    }
    if (!state.terminal) {
      return;
    }
    queueTerminalWrite(sanitized);
  }

  function setTerminalLoader(visible, message) {
    els.terminalLoader.classList.toggle("hidden", !visible);
    if (message) {
      els.terminalLoaderText.textContent = message;
    }
  }

  function setTerminalStatus(status) {
    els.terminalStatusChip.textContent = status;
  }

  function ensureTerminal() {
    if (state.terminal) {
      fitTerminal();
      return;
    }

    state.terminal = new window.Terminal({
      cursorBlink: true,
      fontFamily: "DejaVu Sans Mono, Cascadia Mono, Fira Code, monospace",
      fontSize: 13,
      lineHeight: 1.1,
      letterSpacing: 0,
      customGlyphs: false,
      allowTransparency: false,
      theme: {
        background: "#050b12",
        foreground: "#d7e8f7",
        cursor: "#4fb3ff",
        selectionBackground: "rgba(79, 179, 255, 0.25)",
      },
      scrollback: 1200,
      convertEol: true,
      windowsMode: false,
      allowProposedApi: false,
      smoothScrollDuration: 0,
      fastScrollModifier: "alt",
      fastScrollSensitivity: 1,
    });

    state.fitAddon = new window.FitAddon.FitAddon();
    state.terminal.loadAddon(state.fitAddon);
    state.terminal.open(els.terminalContainer);
    tryEnableWebgl();
    fitTerminal();
    window.setTimeout(() => fitTerminal(), 50);
    if (state.terminalBuffer) {
      queueTerminalWrite(state.terminalBuffer);
    }

    state.terminal.attachCustomKeyEventHandler((event) => {
      if (event.type !== "keydown") {
        return true;
      }

      const key = String(event.key || "").toLowerCase();
      if (event.ctrlKey && event.shiftKey && key === "c") {
        copyTerminalSelection();
        return false;
      }
      if (event.ctrlKey && event.shiftKey && key === "v") {
        pasteIntoTerminal();
        return false;
      }
      if (event.shiftKey && key === "insert") {
        pasteIntoTerminal();
        return false;
      }

      return true;
    });

    state.terminal.onData((data) => {
      window.go.main.App.SendTerminalInput(data).catch((error) => {
        appendTerminalOutput(`\r\n[MySSH] Input error: ${String(error)}\r\n`);
      });
    });

    state.terminal.onResize((size) => {
      window.go.main.App.ResizeTerminal(size.cols, size.rows).catch(() => {});
    });
  }

  function fitTerminal() {
    if (!state.fitAddon || !state.terminalVisible) {
      return;
    }
    state.fitAddon.fit();
    const cols = state.terminal.cols;
    const rows = state.terminal.rows;
    window.go.main.App.ResizeTerminal(cols, rows).catch(() => {});
  }

  function destroyTerminal() {
    state.pendingTerminalWrite = "";
    state.terminalFlushScheduled = false;
    if (state.webglAddon) {
      try {
        state.webglAddon.dispose();
      } catch (_) {
        // ignore dispose errors
      }
      state.webglAddon = null;
    }

    if (state.terminal) {
      try {
        state.terminal.dispose();
      } catch (_) {
        // ignore dispose errors
      }
      state.terminal = null;
    }

    state.fitAddon = null;
    els.terminalContainer.innerHTML = "";
  }

  function tryEnableWebgl() {
    if (!state.webglEnabled) {
      return;
    }
    if (!window.WebglAddon?.WebglAddon) {
      return;
    }
    if (state.webglAddon) {
      return;
    }

    try {
      state.webglAddon = new window.WebglAddon.WebglAddon();
      state.terminal.loadAddon(state.webglAddon);
    } catch (_) {
      state.webglAddon = null;
      state.webglEnabled = false;
    }
    updateGpuButton();
  }

  function toggleTerminalGpu() {
    state.webglEnabled = !state.webglEnabled;
    recreateTerminal();
    updateGpuButton();
  }

  function updateGpuButton() {
    els.terminalGpuToggle.textContent = state.webglEnabled ? "GPU On" : "GPU Off";
  }

  function queueTerminalWrite(chunk) {
    state.pendingTerminalWrite += chunk;
    if (state.terminalFlushScheduled) {
      return;
    }
    state.terminalFlushScheduled = true;
    window.requestAnimationFrame(flushTerminalWrite);
  }

  function flushTerminalWrite() {
    state.terminalFlushScheduled = false;
    if (!state.terminal || !state.pendingTerminalWrite) {
      return;
    }
    const chunk = state.pendingTerminalWrite;
    state.pendingTerminalWrite = "";
    state.terminal.write(chunk);
  }

  function recreateTerminal() {
    if (!state.terminalVisible) {
      return;
    }
    destroyTerminal();
    ensureTerminal();
    fitTerminal();
    state.terminal.focus();
  }

  function debounceResizeTerminal() {
    window.clearTimeout(debounceResizeTerminal._timer);
    debounceResizeTerminal._timer = window.setTimeout(() => {
      fitTerminal();
    }, 120);
  }

  async function copyTerminalSelection() {
    const selection = state.terminal?.getSelection?.() || "";
    if (!selection) {
      setStatus("Nothing selected in terminal.");
      return;
    }
    await window.go.main.App.CopyToClipboard(selection);
    setStatus("Terminal selection copied.");
  }

  async function pasteIntoTerminal() {
    if (!state.terminalVisible) {
      return;
    }
    const text = await window.go.main.App.PasteFromClipboard();
    if (!text) {
      return;
    }
    await window.go.main.App.SendTerminalInput(text);
  }

  function sanitizeTerminalChunk(chunk) {
    return String(chunk || "")
      .replace(/\u001b]1337;File=[\s\S]*?(?:\u0007|\u001b\\)/g, "")
      .replace(/\u001b\]52;[\s\S]*?(?:\u0007|\u001b\\)/g, "")
      .replace(/\u001b_[\s\S]*?(?:\u0007|\u001b\\)/g, "")
      .replace(/\u001bP(?:q|0;1;q)[\s\S]*?\u001b\\/g, "")
      .replace(/\u001bP[\s\S]*?\u001b\\/g, "");
  }

  function escapeHtml(value) {
    return String(value ?? "")
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#39;");
  }
})();
