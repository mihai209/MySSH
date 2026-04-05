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
    sessions: [],
    activeSessionId: "",
    sftp: {
      visible: false,
      sessionId: "",
      profileId: "",
      profileName: "",
      path: "",
      parent: "",
      entries: [],
    },
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
    els.openLocal = document.getElementById("open-local");
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
    els.openSFTP = document.getElementById("open-sftp");
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
    els.passphraseWrap = document.getElementById("passphrase-wrap");
    els.keyContentWrap = document.getElementById("key-content-wrap");
    els.keyContent = document.getElementById("key-content");
    els.modalSecurityCopy = document.getElementById("modal-security-copy");
    els.secret = document.getElementById("secret");
    els.passphrase = document.getElementById("passphrase");
    els.terminalScreen = document.getElementById("terminal-screen");
    els.terminalTitle = document.getElementById("terminal-title");
    els.terminalSubtitle = document.getElementById("terminal-subtitle");
    els.terminalStatusChip = document.getElementById("terminal-status-chip");
    els.terminalSFTPToggle = document.getElementById("terminal-sftp-toggle");
    els.terminalRename = document.getElementById("terminal-rename");
    els.terminalReconnect = document.getElementById("terminal-reconnect");
    els.terminalClose = document.getElementById("terminal-close");
    els.terminalTabs = document.getElementById("terminal-tabs");
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
    els.sftpScreen = document.getElementById("sftp-screen");
    els.sftpTitle = document.getElementById("sftp-title");
    els.sftpSubtitle = document.getElementById("sftp-subtitle");
    els.sftpStatusChip = document.getElementById("sftp-status-chip");
    els.sftpLoader = document.getElementById("sftp-loader");
    els.sftpLoaderText = document.getElementById("sftp-loader-text");
    els.sftpPath = document.getElementById("sftp-path");
    els.sftpList = document.getElementById("sftp-list");
    els.sftpRefresh = document.getElementById("sftp-refresh");
    els.sftpUp = document.getElementById("sftp-up");
    els.sftpUpload = document.getElementById("sftp-upload");
    els.sftpMkdir = document.getElementById("sftp-mkdir");
    els.sftpClose = document.getElementById("sftp-close");
    els.toastStack = document.getElementById("toast-stack");
    els.workspaceBody = document.querySelector(".workspace-body");
    els.connectSecret = document.getElementById("connect-secret");
  }

  function bindEvents() {
    els.search.addEventListener("input", applyFilter);
    els.newProfile.addEventListener("click", openCreateModal);
    els.openLocal.addEventListener("click", connectLocalShell);
    els.emptyAddButton.addEventListener("click", openCreateModal);
    els.closeModal.addEventListener("click", closeModal);
    els.cancelModal.addEventListener("click", closeModal);
    els.saveProfile.addEventListener("click", saveProfile);
    els.editProfile.addEventListener("click", openEditModal);
    els.connectProfile.addEventListener("click", connectProfile);
    els.openSFTP.addEventListener("click", openSFTP);
    els.deleteProfile.addEventListener("click", deleteProfile);
    els.authKind.addEventListener("change", updateSecurityCopy);
    els.keySource.addEventListener("change", updateSecurityCopy);
    els.terminalBack.addEventListener("click", closeTerminal);
    els.terminalSFTPToggle.addEventListener("click", toggleSFTP);
    els.terminalRename.addEventListener("click", renameActiveSession);
    els.terminalReconnect.addEventListener("click", reconnectActiveSession);
    els.terminalClose.addEventListener("click", closeActiveSession);
    els.terminalTrustButton.addEventListener("click", trustPendingHost);
    els.terminalGpuToggle.addEventListener("click", toggleTerminalGpu);
    els.terminalCopy.addEventListener("click", copyTerminalSelection);
    els.terminalPaste.addEventListener("click", pasteIntoTerminal);
    els.sftpRefresh.addEventListener("click", refreshSFTP);
    els.sftpUp.addEventListener("click", goUpSFTP);
    els.sftpUpload.addEventListener("click", uploadToSFTP);
    els.sftpMkdir.addEventListener("click", mkdirInSFTP);
    els.sftpClose.addEventListener("click", closeSFTP);
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
      els.openSFTP.disabled = true;
      return;
    }

    els.emptyState.classList.add("hidden");
    els.machineDetail.classList.remove("hidden");
    els.detailTitle.textContent = profile.name || "Machine";
    els.heroTitle.textContent = profile.name || "SSH Machine";
    els.heroCopy.textContent = "Connect opens a dedicated terminal workspace with loader, live output and autoreconnect.";
    els.machineMeta.textContent = formatSessionTarget(profile);
    els.machineName.textContent = profile.name || "-";
    els.machineUsername.textContent = profile.username || "-";
    els.machineHost.textContent = profile.host || "-";
    els.machinePort.textContent = String(profile.port || 22);
    els.machineAuth.textContent = profile.authKind || "agent";
    els.machineSecretState.textContent = describeSecretState(profile);
    els.editProfile.disabled = false;
    els.connectProfile.disabled = !canConnectProfile(profile);
    els.openSFTP.disabled = !canOpenSFTP(profile);
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
    els.passphrase.value = "";
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
    els.passphrase.value = "";
    els.passphrase.placeholder = profile.hasPassphrase
      ? "Stored in OS keyring"
      : "Optional passphrase stored in OS keyring";
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
        keySource: usesKeyAuthentication() ? els.keySource.value : "",
        keyPath: usesKeyAuthentication() && els.keySource.value === "path" ? els.keyPath.value : "",
        secretValue: resolveSecretValue(),
        passphraseValue: resolvePassphraseValue(),
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
    if (!profile || !canConnectProfile(profile)) {
      setStatus(profile?.authKind === "private_key" ? "Private key path is missing or authentication is not ready." : "Connect is currently available only for password, agent, and private key profiles.");
      return;
    }

    try {
      await ensureSSHSessionForProfile(profile);
    } catch (error) {
      setStatus(String(error), true);
    }
  }

  async function connectLocalShell() {
    const tab = openSessionTab({
      title: "Local Terminal",
      subtitle: "local@localhost",
      kind: "local",
      loaderVisible: true,
      loaderText: "Opening local shell...",
      status: "Connecting",
    });
    appendTerminalOutputToTab(tab.id, "[MySSH] Opening local shell\r\n");

    try {
      const backendSessionId = await window.go.main.App.ConnectLocalShell();
      attachBackendSessionToTab(tab.id, backendSessionId);
      updateSession(tab.id, {
        status: "Local",
        loaderVisible: false,
        loaderText: "",
      });
    } catch (error) {
      updateSession(tab.id, {
        loaderVisible: false,
        status: "Error",
      });
      appendTerminalOutputToTab(tab.id, `[MySSH] Local shell error: ${String(error)}\r\n`);
      setStatus(String(error), true);
    }
  }

  async function openSFTP() {
    const profile = state.profiles.find((item) => item.id === state.selectedId);
    if (!profile || !canOpenSFTP(profile)) {
      showToast("SFTP needs a valid SSH profile with working authentication.", true);
      return;
    }

    try {
      await ensureSSHSessionForProfile(profile);
    } catch (error) {
      showToast(`SFTP needs a live SSH session: ${String(error)}`, true);
      return;
    }

    state.sftp.visible = true;
    state.sftp.profileId = profile.id;
    state.sftp.profileName = profile.name || "SFTP Browser";
    state.sftp.sessionId = "";
    state.sftp.path = "";
    state.sftp.parent = "";
    state.sftp.entries = [];
    syncSFTPView();
    setSFTPLoader(true, `Opening SFTP for ${profile.host}...`);

    try {
      const directory = await window.go.main.App.OpenSFTP(profile.id);
      updateSFTPDirectory(directory);
      setSFTPLoader(false);
      els.sftpStatusChip.textContent = "Ready";
      showToast(`SFTP opened for ${profile.name || profile.host}.`);
    } catch (error) {
      setSFTPLoader(false);
      els.sftpStatusChip.textContent = "Error";
      showToast(String(error), true);
    }
  }

  async function refreshSFTP() {
    if (!state.sftp.sessionId || !state.sftp.path) {
      return;
    }
    setSFTPLoader(true, `Refreshing ${state.sftp.path}...`);
    try {
      const directory = await window.go.main.App.ListSFTP(state.sftp.sessionId, state.sftp.path);
      updateSFTPDirectory(directory);
      showToast("SFTP list refreshed.");
    } catch (error) {
      showToast(String(error), true);
    } finally {
      setSFTPLoader(false);
    }
  }

  async function goUpSFTP() {
    if (!state.sftp.parent) {
      return;
    }
    await navigateSFTP(state.sftp.parent);
  }

  async function navigateSFTP(path) {
    if (!state.sftp.sessionId) {
      return;
    }
    setSFTPLoader(true, `Loading ${path}...`);
    try {
      const directory = await window.go.main.App.ListSFTP(state.sftp.sessionId, path);
      updateSFTPDirectory(directory);
    } catch (error) {
      showToast(String(error), true);
    } finally {
      setSFTPLoader(false);
    }
  }

  async function downloadSFTPFile(path) {
    if (!state.sftp.sessionId) {
      return;
    }
    setSFTPLoader(true, `Downloading ${path}...`);
    try {
      const localPath = await window.go.main.App.DownloadSFTPFile(state.sftp.sessionId, path);
      showToast(`Downloaded to ${localPath}`);
    } catch (error) {
      showToast(String(error), true);
    } finally {
      setSFTPLoader(false);
    }
  }

  async function uploadToSFTP() {
    if (!state.sftp.sessionId || !state.sftp.path) {
      return;
    }
    setSFTPLoader(true, `Uploading into ${state.sftp.path}...`);
    try {
      const directory = await window.go.main.App.UploadSFTPFileToPath(state.sftp.sessionId, state.sftp.path);
      updateSFTPDirectory(directory);
      showToast("Upload completed.");
    } catch (error) {
      if (!String(error).includes("no local file selected")) {
        showToast(String(error), true);
      }
    } finally {
      setSFTPLoader(false);
    }
  }

  async function mkdirInSFTP() {
    if (!state.sftp.sessionId || !state.sftp.path) {
      return;
    }
    const name = window.prompt("New folder name");
    if (!name) {
      return;
    }
    setSFTPLoader(true, `Creating ${name}...`);
    try {
      const directory = await window.go.main.App.MkdirSFTP(state.sftp.sessionId, state.sftp.path, name);
      updateSFTPDirectory(directory);
      showToast("Folder created.");
    } catch (error) {
      showToast(String(error), true);
    } finally {
      setSFTPLoader(false);
    }
  }

  async function renameSFTPEntry(entry) {
    const nextName = window.prompt("Rename to", entry.name);
    if (!nextName || nextName === entry.name) {
      return;
    }
    setSFTPLoader(true, `Renaming ${entry.name}...`);
    try {
      const directory = await window.go.main.App.RenameSFTPPath(state.sftp.sessionId, entry.path, nextName);
      updateSFTPDirectory(directory);
      showToast("Remote path renamed.");
    } catch (error) {
      showToast(String(error), true);
    } finally {
      setSFTPLoader(false);
    }
  }

  async function deleteSFTPEntry(entry) {
    const confirmed = window.confirm(`Delete ${entry.isDir ? "folder" : "file"} "${entry.name}"?`);
    if (!confirmed) {
      return;
    }
    setSFTPLoader(true, `Deleting ${entry.name}...`);
    try {
      const directory = await window.go.main.App.DeleteSFTPPath(state.sftp.sessionId, entry.path, entry.isDir);
      updateSFTPDirectory(directory);
      showToast("Remote path deleted.");
    } catch (error) {
      showToast(String(error), true);
    } finally {
      setSFTPLoader(false);
    }
  }

  function updateSecurityCopy() {
    els.keySourceWrap.classList.add("hidden");
    els.keyPathWrap.classList.add("hidden");
    els.secretWrap.classList.add("hidden");
    els.passphraseWrap.classList.add("hidden");
    els.keyContentWrap.classList.add("hidden");

    switch (els.authKind.value) {
      case "password":
        els.securityCopy.textContent = "Passwords are persisted in your OS keyring, not in profiles.json.";
        els.modalSecurityCopy.textContent = "Passwords are persisted in your OS keyring, not in profiles.json.";
        els.secret.placeholder = "Stored in OS keyring";
        els.secretWrap.classList.remove("hidden");
        break;
      case "agent_fallback_key":
        els.securityCopy.textContent = "Agent + fallback key tries ssh-agent first, then uses the configured key if needed. Key content and passphrase are persisted in your OS keyring.";
        els.keySourceWrap.classList.remove("hidden");
        els.passphraseWrap.classList.remove("hidden");
        els.passphrase.placeholder = "Optional passphrase stored in OS keyring";
        if (els.keySource.value === "content") {
          els.keyContentWrap.classList.remove("hidden");
          els.modalSecurityCopy.textContent = "Agent mode is tried first. If it fails, pasted key content and optional passphrase are loaded from your OS keyring.";
        } else {
          els.keyPathWrap.classList.remove("hidden");
          els.modalSecurityCopy.textContent = "Agent mode is tried first. If it fails, the local key path and optional passphrase are used.";
        }
        break;
      case "private_key":
        els.securityCopy.textContent = "Private key mode supports either a key path reference or pasted key content persisted in your OS keyring.";
        els.keySourceWrap.classList.remove("hidden");
        els.passphraseWrap.classList.remove("hidden");
        els.passphrase.placeholder = "Optional passphrase stored in OS keyring";
        if (els.keySource.value === "content") {
          els.keyContentWrap.classList.remove("hidden");
          els.modalSecurityCopy.textContent = "Pasted private key content and optional passphrase are persisted in your OS keyring.";
        } else {
          els.keyPathWrap.classList.remove("hidden");
          els.modalSecurityCopy.textContent = "Key path mode stores only the filesystem path in the profile metadata. Optional passphrase is persisted in your OS keyring.";
        }
        break;
      default:
        els.securityCopy.textContent = "Agent mode is the safest default for the MVP and avoids local secret persistence entirely.";
        els.modalSecurityCopy.textContent = "Agent mode avoids local secret persistence entirely.";
        break;
    }
  }

  function needsSecretValue() {
    return els.authKind.value === "password" || (usesKeyAuthentication() && els.keySource.value === "content");
  }

  function resolveSecretValue() {
    if (!needsSecretValue()) {
      return "";
    }

    if (usesKeyAuthentication() && els.keySource.value === "content") {
      return els.keyContent.value;
    }

    return els.secret.value;
  }

  function resolvePassphraseValue() {
    if (!usesKeyAuthentication()) {
      return "";
    }
    return els.passphrase.value;
  }

  function usesKeyAuthentication() {
    return els.authKind.value === "private_key" || els.authKind.value === "agent_fallback_key";
  }

  function canConnectProfile(profile) {
    if (!profile) {
      return false;
    }
    if (profile.authKind === "agent" || profile.authKind === "password") {
      return true;
    }
    if (profile.authKind !== "private_key" && profile.authKind !== "agent_fallback_key") {
      return false;
    }
    if (profile.keySource === "content") {
      return Boolean(profile.hasStoredSecret);
    }
    if (profile.keySource === "path") {
      return Boolean(profile.keyPath && profile.keyPathExists);
    }
    return false;
  }

  function canOpenSFTP(profile) {
    return canConnectProfile(profile);
  }

  function describeSecretState(profile) {
    if (!profile) {
      return "none";
    }
    if ((profile.authKind === "private_key" || profile.authKind === "agent_fallback_key") && profile.keySource === "path") {
      return profile.keyPathExists ? "key path found" : "key path missing";
    }
    if (profile.hasStoredSecret) {
      return "stored in OS keyring";
    }
    if (profile.keyPath) {
      return "key path reference";
    }
    return "none";
  }

  function formatSessionTarget(profile) {
    if (!profile) {
      return "Waiting for connection...";
    }
    if (profile.port && Number(profile.port) > 0) {
      return `${profile.username}@${profile.host}:${profile.port}`;
    }
    return `${profile.username}@${profile.host}`;
  }

  function setStatus(message, isError) {
    els.statusText.textContent = message;
    els.statusText.style.color = isError ? "#ff8da0" : "";
  }

  function showToast(message, isError) {
    const toast = document.createElement("div");
    toast.className = `toast${isError ? " error" : ""}`;
    toast.textContent = message;
    els.toastStack.appendChild(toast);
    window.setTimeout(() => {
      toast.remove();
    }, 2600);
  }

  function syncSFTPView() {
    els.sftpScreen.classList.toggle("hidden", !state.sftp.visible);
    els.workspaceBody.classList.toggle("split", state.sftp.visible);
    if (!state.sftp.visible) {
      els.terminalSFTPToggle.textContent = "SFTP";
      return;
    }
    els.terminalSFTPToggle.textContent = "Hide SFTP";
    els.sftpTitle.textContent = `${state.sftp.profileName} SFTP`;
    els.sftpSubtitle.textContent = state.sftp.profileName || "SFTP Browser";
    els.sftpPath.textContent = state.sftp.path || "Waiting for path...";
    els.sftpUp.disabled = !state.sftp.parent;
    renderSFTPEntries();
  }

  function setSFTPLoader(visible, text) {
    els.sftpLoader.classList.toggle("hidden", !visible);
    if (text) {
      els.sftpLoaderText.textContent = text;
    }
  }

  function updateSFTPDirectory(directory) {
    state.sftp.sessionId = directory.sessionId;
    state.sftp.path = directory.path;
    state.sftp.parent = directory.parent;
    state.sftp.entries = directory.entries || [];
    state.sftp.visible = true;
    syncSFTPView();
  }

  function renderSFTPEntries() {
    els.sftpList.innerHTML = "";
    if (!state.sftp.entries.length) {
      const empty = document.createElement("div");
      empty.className = "empty-state";
      empty.textContent = "No files in this folder.";
      els.sftpList.appendChild(empty);
      return;
    }

    state.sftp.entries.forEach((entry) => {
      const row = document.createElement("div");
      row.className = `sftp-entry${entry.isDir ? " sftp-entry-dir" : ""}`;
      row.innerHTML = `
        <div class="sftp-entry-main">
          <div class="sftp-entry-name">${escapeHtml(entry.name)}</div>
          <div class="sftp-entry-meta">${escapeHtml(entry.mode)} • ${entry.isDir ? "directory" : `${entry.size} bytes`}</div>
        </div>
        <div class="sftp-entry-meta">${escapeHtml(formatTimestamp(entry.modified))}</div>
        <div class="sftp-entry-actions">
          <button class="ghost-button" data-sftp-open="${escapeHtml(entry.path)}">${entry.isDir ? "Open" : "Download"}</button>
          <button class="ghost-button" data-sftp-rename="${escapeHtml(entry.path)}">Rename</button>
          <button class="danger-button" data-sftp-delete="${escapeHtml(entry.path)}">Delete</button>
        </div>
      `;
      row.querySelector("[data-sftp-open]").addEventListener("click", () => {
        if (entry.isDir) {
          navigateSFTP(entry.path);
          return;
        }
        downloadSFTPFile(entry.path);
      });
      row.querySelector("[data-sftp-rename]").addEventListener("click", () => renameSFTPEntry(entry));
      row.querySelector("[data-sftp-delete]").addEventListener("click", () => deleteSFTPEntry(entry));
      els.sftpList.appendChild(row);
    });
  }

  async function closeSFTP() {
    if (state.sftp.sessionId) {
      try {
        await window.go.main.App.CloseSFTP(state.sftp.sessionId);
      } catch (_) {}
    }
    state.sftp = {
      visible: false,
      sessionId: "",
      profileId: "",
      profileName: "",
      path: "",
      parent: "",
      entries: [],
    };
    syncSFTPView();
    showToast("SFTP session closed.");
  }

  function toggleSFTP() {
    const session = activeSession();
    if (!session || session.kind !== "ssh" || !session.profileId) {
      showToast("SFTP is available only for SSH sessions.", true);
      return;
    }
    if (!state.sftp.visible) {
      const profile = state.profiles.find((item) => item.id === session.profileId);
      if (!profile) {
        showToast("Profile no longer exists for this session.", true);
        return;
      }
      state.selectedId = profile.id || state.selectedId;
      renderProfiles();
      renderSelection();
      openSFTP();
      return;
    }
    state.sftp.visible = false;
    syncSFTPView();
    showToast("SFTP panel hidden. Session stays open until Close.");
  }

  function openSessionTab(session) {
    const entry = {
      id: `client-${Date.now()}-${Math.random().toString(16).slice(2, 8)}`,
      backendSessionId: "",
      title: session.title || "Session",
      subtitle: session.subtitle || "Waiting for connection...",
      kind: session.kind || "ssh",
      connectTarget: session.connectTarget || null,
      profileId: session.profileId || "",
      pendingHostKeyId: session.pendingHostKeyId || "",
      buffer: "",
      status: session.status || "Idle",
      loaderVisible: Boolean(session.loaderVisible),
      loaderText: session.loaderText || "",
      trustVisible: false,
      trustText: "",
    };
    state.sessions.push(entry);
    activateSession(entry.id);
    return entry;
  }

  function updateSession(sessionId, patch) {
    const session = findSession(sessionId);
    if (!session) {
      return;
    }
    Object.assign(session, patch);
    if (state.activeSessionId === sessionId) {
      syncActiveSessionView();
    }
    renderSessionTabs();
  }

  function attachBackendSessionToTab(clientSessionId, backendSessionId) {
    const session = findSession(clientSessionId);
    if (!session) {
      return;
    }
    session.backendSessionId = backendSessionId;
    if (!session.pendingHostKeyId) {
      session.pendingHostKeyId = backendSessionId;
    }
    renderSessionTabs();
  }

  function activateSession(sessionId) {
    state.activeSessionId = sessionId;
    state.terminalVisible = true;
    els.terminalScreen.classList.remove("hidden");
    ensureTerminal();
    syncActiveSessionView();
    renderSessionTabs();
    state.terminal.focus();
  }

  function syncActiveSessionView() {
    const session = activeSession();
    if (!session) {
      els.terminalScreen.classList.add("hidden");
      els.terminalSFTPToggle.disabled = true;
      return;
    }

    state.terminalBuffer = session.buffer || "";
    els.terminalTitle.textContent = session.title;
    els.terminalSubtitle.textContent = session.subtitle;
    setTerminalStatus(session.status || "Idle");
    setTerminalLoader(Boolean(session.loaderVisible), session.loaderText || "");
    els.terminalTrust.classList.toggle("hidden", !session.trustVisible);
    els.terminalTrustCopy.textContent = session.trustText || "";
    els.terminalSFTPToggle.disabled = session.kind !== "ssh" || !session.profileId;
    if (state.sftp.visible && (session.kind !== "ssh" || state.sftp.profileId !== session.profileId)) {
      state.sftp.visible = false;
      syncSFTPView();
    }

    destroyTerminal();
    ensureTerminal();
    state.terminal.reset();
    if (session.buffer) {
      queueTerminalWrite(session.buffer);
    }
  }

  function renderSessionTabs() {
    els.terminalTabs.innerHTML = "";
    state.sessions.forEach((session) => {
      const button = document.createElement("div");
      button.className = "terminal-tab";
      button.setAttribute("role", "button");
      button.setAttribute("tabindex", "0");
      if (session.id === state.activeSessionId) {
        button.classList.add("active");
      }
      const label = document.createElement("span");
      label.className = "terminal-tab-label";
      label.textContent = session.title;
      const closeButton = document.createElement("span");
      closeButton.className = "terminal-tab-close";
      closeButton.setAttribute("data-session-close", session.id);
      closeButton.setAttribute("role", "button");
      closeButton.setAttribute("tabindex", "0");
      closeButton.textContent = "x";
      button.appendChild(label);
      button.appendChild(closeButton);
      button.addEventListener("click", () => activateSession(session.id));
      button.addEventListener("keydown", (event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          activateSession(session.id);
        }
      });
      els.terminalTabs.appendChild(button);
    });
    els.terminalReconnect.disabled = !activeSession();
    els.terminalRename.disabled = !activeSession();
    els.terminalClose.disabled = !activeSession();
    els.terminalCopy.disabled = !activeSession();
    els.terminalPaste.disabled = !activeSession();
    els.terminalGpuToggle.disabled = !activeSession();
    els.terminalSFTPToggle.disabled = !activeSession() || activeSession()?.kind !== "ssh";
    els.terminalReconnect.textContent = activeSession()?.kind === "local" ? "Reopen" : "Reconnect";
    Array.from(els.terminalTabs.querySelectorAll("[data-session-close]")).forEach((closeButton) => {
      closeButton.addEventListener("click", (event) => {
        event.stopPropagation();
        closeSession(closeButton.getAttribute("data-session-close"));
      });
    });
  }

  async function closeTerminal() {
    state.terminalVisible = false;
    els.terminalScreen.classList.add("hidden");
    if (state.sessions.length) {
      showToast("Session remains active in background until you press Close.");
    }
  }

  async function closeActiveSession() {
    if (!state.activeSessionId) {
      return;
    }
    await closeSession(state.activeSessionId);
  }

  async function closeSession(sessionId) {
    const session = findSession(sessionId);
    if (!session) {
      return;
    }
    if (state.sftp.sessionId && state.sftp.profileId && session.profileId === state.sftp.profileId) {
      try {
        await window.go.main.App.CloseSFTP(state.sftp.sessionId);
      } catch (_) {}
      state.sftp = {
        visible: false,
        sessionId: "",
        profileId: "",
        profileName: "",
        path: "",
        parent: "",
        entries: [],
      };
      syncSFTPView();
    }
    if (session.backendSessionId) {
      try {
        await window.go.main.App.DisconnectTerminal(session.backendSessionId);
      } catch (_) {}
    }
    state.sessions = state.sessions.filter((item) => item.id !== sessionId);
    if (state.activeSessionId === sessionId) {
      state.activeSessionId = state.sessions[0]?.id || "";
    }
    if (state.activeSessionId) {
      activateSession(state.activeSessionId);
    } else {
      state.terminalVisible = false;
      els.terminalScreen.classList.add("hidden");
      destroyTerminal();
    }
    renderSessionTabs();
  }

  async function reconnectActiveSession() {
    const session = activeSession();
    if (!session) {
      return;
    }
    if (session.backendSessionId) {
      try {
        await window.go.main.App.DisconnectTerminal(session.backendSessionId);
      } catch (_) {}
      session.backendSessionId = "";
    }
    session.buffer = "";
    session.loaderVisible = true;
    session.status = session.kind === "local" ? "Opening" : "Connecting";
    session.trustVisible = false;
    syncActiveSessionView();
    if (session.kind === "local") {
      const backendSessionId = await window.go.main.App.ConnectLocalShell();
      attachBackendSessionToTab(session.id, backendSessionId);
      updateSession(session.id, {
        status: "Local",
        loaderVisible: false,
        loaderText: "",
      });
      showToast("Local terminal reopened.");
      return;
    }
    if (session.profileId) {
      const backendSessionId = await window.go.main.App.ConnectProfile(session.profileId);
      attachBackendSessionToTab(session.id, backendSessionId);
      updateSession(session.id, {
        status: "Connected",
        loaderVisible: false,
        loaderText: "",
      });
      showToast("SSH session reconnected.");
    }
  }

  function renameActiveSession() {
    const session = activeSession();
    if (!session) {
      return;
    }
    const nextTitle = window.prompt("Session name", session.title);
    if (!nextTitle) {
      return;
    }
    session.title = nextTitle.trim() || session.title;
    syncActiveSessionView();
    renderSessionTabs();
  }

  function activeSession() {
    return state.sessions.find((item) => item.id === state.activeSessionId) || null;
  }

  function findSessionByProfile(profileId) {
    return state.sessions.find((item) => item.kind === "ssh" && item.profileId === profileId) || null;
  }

  async function ensureSSHSessionForProfile(profile) {
    const existing = findSessionByProfile(profile.id);
    if (existing) {
      activateSession(existing.id);
      if (existing.backendSessionId) {
        return existing;
      }
    }

    const tab = existing || openSessionTab({
      title: profile.name || "SSH Session",
      subtitle: formatSessionTarget(profile),
      connectTarget: profile,
      profileId: profile.id,
      kind: "ssh",
      loaderVisible: true,
      loaderText: `Connecting to ${profile.host}...`,
      status: "Connecting",
      pendingHostKeyId: profile.id,
    });
    if (!existing) {
      appendTerminalOutputToTab(tab.id, `[MySSH] Connecting to ${profile.username}@${profile.host}:${profile.port}\r\n`);
    } else {
      updateSession(tab.id, {
        loaderVisible: true,
        loaderText: `Connecting to ${profile.host}...`,
        status: "Connecting",
      });
    }

    try {
      const backendSessionId = await window.go.main.App.ConnectProfile(profile.id);
      attachBackendSessionToTab(tab.id, backendSessionId);
      updateSession(tab.id, {
        status: "Connected",
        loaderVisible: false,
        loaderText: "",
      });
      return tab;
    } catch (error) {
      updateSession(tab.id, {
        loaderVisible: false,
        status: "Error",
      });
      appendTerminalOutputToTab(tab.id, `[MySSH] Connection error: ${String(error)}\r\n`);
      throw error;
    }
  }

  function findSession(sessionId) {
    return state.sessions.find((item) => item.id === sessionId) || null;
  }

  function handleTerminalOutput(payload) {
    if (!payload?.sessionId) {
      return;
    }
    appendTerminalOutputToTab(resolveSessionTabId(payload.sessionId), payload?.chunk || "");
  }

  function handleTerminalStatus(payload) {
    const sessionId = resolveSessionTabId(payload?.sessionId);
    const message = payload?.message || "SSH status update";
    const status = payload?.state || "Idle";
    const profile = payload?.profile || {};
    const session = findSession(sessionId);
    if (!session) {
      return;
    }
    session.title = profile.name || session.title || "SSH Session";
    session.subtitle = profile.host ? formatSessionTarget(profile) : session.subtitle;
    session.status = status;
    session.loaderVisible = status === "connecting" || status === "reconnecting";
    session.loaderText = message;
    session.trustVisible = false;
    appendTerminalOutputToTab(session.id, `[MySSH] ${message}\r\n`);
    if (state.activeSessionId === session.id) {
      syncActiveSessionView();
    }
    renderSessionTabs();
  }

  function handleUnknownHostKey(payload) {
    const session = findSession(resolveSessionTabId(payload?.sessionId));
    if (!session) {
      return;
    }
    session.trustVisible = true;
    session.trustText = `${payload?.message || "Unknown host key"}\n${payload?.fingerprint || ""}`;
    session.status = "Trust Required";
    session.loaderVisible = false;
    appendTerminalOutputToTab(session.id, `[MySSH] Unknown host key: ${payload?.fingerprint || "unknown"}\r\n`);
    if (state.activeSessionId === session.id) {
      syncActiveSessionView();
    }
  }

  async function trustPendingHost() {
    const session = activeSession();
    if (!session) {
      return;
    }
    try {
      await window.go.main.App.TrustPendingHost(session.pendingHostKeyId || session.profileId || "");
      session.trustVisible = false;
      session.status = "Trusted";
      appendTerminalOutputToTab(session.id, "[MySSH] Host key trusted. Reconnect now.\r\n");
      await reconnectActiveSession();
    } catch (error) {
      appendTerminalOutputToTab(session.id, `[MySSH] Trust error: ${String(error)}\r\n`);
    }
  }

  function appendTerminalOutput(chunk) {
    const session = activeSession();
    if (!session) {
      return;
    }
    appendTerminalOutputToTab(session.id, chunk);
  }

  function appendTerminalOutputToTab(sessionId, chunk) {
    const session = findSession(sessionId);
    if (!session) {
      return;
    }
    const sanitized = sanitizeTerminalChunk(chunk);
    if (!sanitized) {
      return;
    }
    session.buffer = (session.buffer || "") + sanitized;
    if (session.buffer.length > 1200000) {
      session.buffer = session.buffer.slice(-900000);
    }
    if (!state.terminal || state.activeSessionId !== sessionId) {
      return;
    }
    state.terminalBuffer = session.buffer;
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
      const session = activeSession();
      if (!session?.backendSessionId) {
        return;
      }
      window.go.main.App.SendTerminalInput(session.backendSessionId, data).catch((error) => {
        appendTerminalOutput(`\r\n[MySSH] Input error: ${String(error)}\r\n`);
      });
    });

    state.terminal.onResize((size) => {
      const session = activeSession();
      if (!session?.backendSessionId) {
        return;
      }
      window.go.main.App.ResizeTerminal(session.backendSessionId, size.cols, size.rows).catch(() => {});
    });
  }

  function fitTerminal() {
    if (!state.fitAddon || !state.terminalVisible) {
      return;
    }
    state.fitAddon.fit();
    const cols = state.terminal.cols;
    const rows = state.terminal.rows;
    const session = activeSession();
    if (!session?.backendSessionId) {
      return;
    }
    window.go.main.App.ResizeTerminal(session.backendSessionId, cols, rows).catch(() => {});
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
      showToast("Nothing selected in terminal.", true);
      return;
    }
    await window.go.main.App.CopyToClipboard(selection);
    showToast("Copied terminal selection.");
  }

  async function pasteIntoTerminal() {
    if (!state.terminalVisible) {
      return;
    }
    const text = await window.go.main.App.PasteFromClipboard();
    if (!text) {
      return;
    }
    const session = activeSession();
    if (!session?.backendSessionId) {
      return;
    }
    await window.go.main.App.SendTerminalInput(session.backendSessionId, text);
    showToast("Pasted from clipboard.");
  }

  function resolveSessionTabId(eventSessionId) {
    return state.sessions.find((item) => item.backendSessionId === eventSessionId || item.pendingHostKeyId === eventSessionId)?.id || eventSessionId;
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

  function formatTimestamp(value) {
    if (!value) {
      return "-";
    }
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return value;
    }
    return date.toLocaleString();
  }
})();
