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
    els.detailTitle = document.getElementById("detail-title");
    els.name = document.getElementById("name");
    els.username = document.getElementById("username");
    els.host = document.getElementById("host");
    els.port = document.getElementById("port");
    els.authKind = document.getElementById("auth-kind");
    els.secret = document.getElementById("secret");
    els.securityCopy = document.getElementById("security-copy");
    els.dataDir = document.getElementById("data-dir");
    els.authMix = document.getElementById("auth-mix");
    els.statusText = document.getElementById("status-text");
    els.newProfile = document.getElementById("new-profile");
    els.saveProfile = document.getElementById("save-profile");
  }

  function bindEvents() {
    els.search.addEventListener("input", applyFilter);
    els.newProfile.addEventListener("click", resetForm);
    els.saveProfile.addEventListener("click", saveProfile);
    els.authKind.addEventListener("change", updateSecurityCopy);
  }

  async function bootstrap() {
    try {
      if (window.go?.main?.App?.Ping) {
        els.pingState.textContent = await window.go.main.App.Ping();
      } else {
        els.pingState.textContent = "Wails bindings pending";
      }

      await refreshDashboard();
      resetForm();
    } catch (error) {
      setStatus(String(error), true);
    }
  }

  async function refreshDashboard() {
    const dashboard = await window.go.main.App.Dashboard();
    state.profiles = dashboard.profiles || [];
    state.filtered = [...state.profiles];

    els.profileCount.textContent = String(state.profiles.length);
    els.recommendedAuth.textContent = dashboard.recommendedAuth || "agent";
    els.dataDir.textContent = dashboard.dataDir || "";
    els.authMix.textContent = `agent ${dashboard.agentCount} | key ${dashboard.keyCount} | password ${dashboard.passwordCount}`;
    els.securityCopy.textContent = dashboard.securityHeadline || "Agent mode is the safest default for the MVP.";

    renderProfiles();
  }

  function applyFilter() {
    const query = els.search.value.trim().toLowerCase();
    state.filtered = state.profiles.filter((profile) => {
      return [profile.name, profile.username, profile.host].some((value) =>
        String(value || "").toLowerCase().includes(query),
      );
    });
    renderProfiles();
  }

  function renderProfiles() {
    els.profileList.innerHTML = "";

    if (!state.filtered.length) {
      const empty = document.createElement("div");
      empty.className = "empty-state";
      empty.textContent = "No profiles match this search yet.";
      els.profileList.appendChild(empty);
      return;
    }

    state.filtered.forEach((profile) => {
      const card = document.createElement("button");
      card.type = "button";
      card.className = "connection-card";
      if (profile.id === state.selectedId) {
        card.classList.add("active");
      }

      card.innerHTML = `
        <div class="connection-title">${escapeHtml(profile.name || "Unnamed host")}</div>
        <div class="connection-meta">${escapeHtml(profile.username)}@${escapeHtml(profile.host)}:${profile.port}</div>
        <div class="connection-auth">${escapeHtml(profile.authKind)}</div>
      `;

      card.addEventListener("click", () => loadProfile(profile));
      els.profileList.appendChild(card);
    });
  }

  function loadProfile(profile) {
    state.selectedId = profile.id || "";
    els.detailTitle.textContent = profile.name || "Connection";
    els.name.value = profile.name || "";
    els.username.value = profile.username || "";
    els.host.value = profile.host || "";
    els.port.value = profile.port || 22;
    els.authKind.value = profile.authKind || "agent";
    els.secret.value = "";
    updateSecurityCopy();
    renderProfiles();
    setStatus("Profile loaded.");
  }

  function resetForm() {
    state.selectedId = "";
    els.detailTitle.textContent = "New Connection";
    els.name.value = "";
    els.username.value = "";
    els.host.value = "";
    els.port.value = 22;
    els.authKind.value = "agent";
    els.secret.value = "";
    updateSecurityCopy();
    renderProfiles();
    setStatus("Ready.");
  }

  async function saveProfile() {
    try {
      const payload = {
        id: state.selectedId,
        name: els.name.value,
        username: els.username.value,
        host: els.host.value,
        port: Number(els.port.value || 22),
        authKind: els.authKind.value,
      };

      const profile = await window.go.main.App.SaveProfile(payload);
      await refreshDashboard();
      const fresh = state.profiles.find((item) => item.id === profile.id);
      if (fresh) {
        loadProfile(fresh);
      }
      setStatus("Profile saved.");
    } catch (error) {
      setStatus(String(error), true);
    }
  }

  function updateSecurityCopy() {
    switch (els.authKind.value) {
      case "password":
        els.securityCopy.textContent = "Password mode is planned, but persistence stays disabled until keyring integration lands.";
        break;
      case "private_key":
        els.securityCopy.textContent = "Private key mode will use safe file references or keyring-backed storage, not plaintext blobs.";
        break;
      default:
        els.securityCopy.textContent = "Agent mode is the safest default for the MVP and avoids local secret persistence entirely.";
        break;
    }
  }

  function setStatus(message, isError) {
    els.statusText.textContent = message;
    els.statusText.style.color = isError ? "#b23a2f" : "";
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
