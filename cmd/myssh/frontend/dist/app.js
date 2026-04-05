(function () {
  const state = {
    profiles: [],
    filtered: [],
    selectedId: "",
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
    els.connectProfile = document.getElementById("connect-profile");
    els.deleteProfile = document.getElementById("delete-profile");
    els.modalTitle = document.getElementById("modal-title");
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
  }

  function bindEvents() {
    els.search.addEventListener("input", applyFilter);
    els.newProfile.addEventListener("click", openCreateModal);
    els.emptyAddButton.addEventListener("click", openCreateModal);
    els.closeModal.addEventListener("click", closeModal);
    els.cancelModal.addEventListener("click", closeModal);
    els.saveProfile.addEventListener("click", saveProfile);
    els.connectProfile.addEventListener("click", connectPlaceholder);
    els.deleteProfile.addEventListener("click", deleteProfile);
    els.authKind.addEventListener("change", updateSecurityCopy);
    els.keySource.addEventListener("change", updateSecurityCopy);
    els.modalBackdrop.addEventListener("click", (event) => {
      if (event.target === els.modalBackdrop) {
        closeModal();
      }
    });
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
      return;
    }

    els.emptyState.classList.add("hidden");
    els.machineDetail.classList.remove("hidden");
    els.detailTitle.textContent = profile.name || "Machine";
    els.heroTitle.textContent = profile.name || "SSH Machine";
    els.heroCopy.textContent = "Connect is a placeholder for now. Delete already removes the profile from local metadata.";
    els.machineMeta.textContent = `${profile.username}@${profile.host}:${profile.port}`;
    els.machineName.textContent = profile.name || "-";
    els.machineUsername.textContent = profile.username || "-";
    els.machineHost.textContent = profile.host || "-";
    els.machinePort.textContent = String(profile.port || 22);
    els.machineAuth.textContent = profile.authKind || "agent";
    els.machineSecretState.textContent = profile.hasStoredSecret ? "stored in OS keyring" : profile.keyPath ? "key path reference" : "none";
  }

  function openCreateModal() {
    els.modalTitle.textContent = "New SSH";
    els.name.value = "";
    els.username.value = "";
    els.host.value = "";
    els.port.value = 22;
    els.authKind.value = "agent";
    els.keySource.value = "path";
    els.keyPath.value = "";
    els.secret.value = "";
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
        id: "",
        name: els.name.value,
        username: els.username.value,
        host: els.host.value,
        port: Number(els.port.value || 22),
        authKind: els.authKind.value,
        keySource: els.authKind.value === "private_key" ? els.keySource.value : "",
        keyPath: els.authKind.value === "private_key" && els.keySource.value === "path" ? els.keyPath.value : "",
        secretValue: resolveSecretValue(),
      };

      const profile = await window.go.main.App.SaveProfile(payload);
      state.selectedId = profile.id;
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

  function connectPlaceholder() {
    const profile = state.profiles.find((item) => item.id === state.selectedId);
    if (!profile) {
      return;
    }

    setStatus(`Connect placeholder for ${profile.username}@${profile.host}:${profile.port}`);
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

  function escapeHtml(value) {
    return String(value ?? "")
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#39;");
  }
})();
