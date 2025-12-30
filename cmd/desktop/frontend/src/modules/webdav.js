// Data Sync / Backup management
import { t } from "../i18n/index.js";

function translateError(error) {
  const errorStr = error.toString();

  // Try new backup error namespace first
  const backupKey = `backup.errors.${errorStr}`;
  const backupTranslated = t(backupKey);
  if (backupTranslated !== backupKey) {
    return backupTranslated;
  }

  // Fallback to legacy webdav errors
  const webdavKey = `webdav.errors.${errorStr}`;
  const webdavTranslated = t(webdavKey);
  return webdavTranslated !== webdavKey ? webdavTranslated : errorStr;
}

// Global variables to store WebDAV config
let currentWebDAVConfig = {
  url: "",
  username: "",
  password: "",
};

let currentBackupConfig = {
  provider: "webdav",
  local: { dir: "" },
  s3: {
    endpoint: "",
    region: "",
    bucket: "",
    prefix: "",
    accessKey: "",
    secretKey: "",
    sessionToken: "",
    useSSL: true,
    forcePathStyle: false,
  },
};

let selectedTab = "webdav";

// Track if connection test passed
let connectionTestPassed = false;

// Show a generic modal
function showModal(title, content) {
  // Remove existing generic modal if any
  const existingModal = document.getElementById("genericModal");
  if (existingModal) {
    existingModal.remove();
  }

  // Create modal element
  const modal = document.createElement("div");
  modal.id = "genericModal";
  modal.className = "modal active";
  modal.innerHTML = `
        <div class="modal-content">
            <div class="modal-header">
                <h2>${title}</h2>
                <button class="modal-close" onclick="window.closeDataSyncDialog()">&times;</button>
            </div>
            <div class="modal-body">
                ${content}
            </div>
        </div>
    `;

  document.body.appendChild(modal);

  // Do NOT close modal when clicking outside (like history modal)
}

// Show a sub-modal on top of existing modal
function showSubModal(title, content) {
  // Remove existing sub modal if any
  const existingModal = document.getElementById("subModal");
  if (existingModal) {
    existingModal.remove();
  }

  // Create modal element
  const modal = document.createElement("div");
  modal.id = "subModal";
  modal.className = "modal active";
  modal.style.zIndex = "1001";
  modal.innerHTML = `
        <div class="modal-content">
            <div class="modal-header">
                <h2>${title}</h2>
                <button class="modal-close" onclick="window.closeSubModal()">&times;</button>
            </div>
            <div class="modal-body">
                ${content}
            </div>
        </div>
    `;

  document.body.appendChild(modal);

  // Do NOT close modal when clicking outside (like history modal)
}

// Show a confirm modal on top of sub modal
function showConfirmModal(title, content, allowClickOutsideClose = true) {
  const existingModal = document.getElementById("confirmModal");
  if (existingModal) {
    existingModal.remove();
  }

  const modal = document.createElement("div");
  modal.id = "confirmModal";
  modal.className = "modal active";
  modal.style.zIndex = "1002";
  modal.innerHTML = content;

  document.body.appendChild(modal);

  if (allowClickOutsideClose) {
    modal.addEventListener("click", (e) => {
      if (e.target === modal) {
        hideConfirmModal();
      }
    });
  }
}

// Hide the confirm modal
function hideConfirmModal() {
  const modal = document.getElementById("confirmModal");
  if (modal) {
    modal.classList.remove("active");
    setTimeout(() => modal.remove(), 300);
  }
}

// Hide the sub modal
function hideSubModal() {
  const modal = document.getElementById("subModal");
  if (modal) {
    modal.classList.remove("active");
    setTimeout(() => modal.remove(), 300);
  }
}

// Global function to close sub modal
window.closeSubModal = function () {
  hideSubModal();
};

// Hide the generic modal
function hideModal() {
  const modal = document.getElementById("genericModal");
  if (modal) {
    modal.classList.remove("active");
    setTimeout(() => modal.remove(), 300);
  }
}

// Load config from backend
async function loadDataSyncConfig() {
  try {
    const configStr = await window.go.main.App.GetConfig();
    const cfg = JSON.parse(configStr);

    if (cfg.webdav) {
      currentWebDAVConfig = {
        url: cfg.webdav.url || "",
        username: cfg.webdav.username || "",
        password: cfg.webdav.password || "",
      };
    }

    const backupCfg = cfg.backup || {};
    currentBackupConfig = {
      provider: backupCfg.provider || "webdav",
      local: {
        dir: backupCfg.local && backupCfg.local.dir ? backupCfg.local.dir : "",
      },
      s3: {
        endpoint:
          backupCfg.s3 && backupCfg.s3.endpoint ? backupCfg.s3.endpoint : "",
        region: backupCfg.s3 && backupCfg.s3.region ? backupCfg.s3.region : "",
        bucket: backupCfg.s3 && backupCfg.s3.bucket ? backupCfg.s3.bucket : "",
        prefix: backupCfg.s3 && backupCfg.s3.prefix ? backupCfg.s3.prefix : "",
        accessKey:
          backupCfg.s3 && backupCfg.s3.accessKey ? backupCfg.s3.accessKey : "",
        secretKey:
          backupCfg.s3 && backupCfg.s3.secretKey ? backupCfg.s3.secretKey : "",
        sessionToken:
          backupCfg.s3 && backupCfg.s3.sessionToken
            ? backupCfg.s3.sessionToken
            : "",
        useSSL:
          backupCfg.s3 && typeof backupCfg.s3.useSSL === "boolean"
            ? backupCfg.s3.useSSL
            : true,
        forcePathStyle:
          backupCfg.s3 && typeof backupCfg.s3.forcePathStyle === "boolean"
            ? backupCfg.s3.forcePathStyle
            : false,
      },
    };
  } catch (error) {
    console.error("Failed to load backup config:", error);
  }
}

// Show Data Sync Dialog (main entry point)
export async function showDataSyncDialog(tab) {
  connectionTestPassed = false;
  await loadDataSyncConfig();
  selectedTab = tab || currentBackupConfig.provider || "webdav";

  const content = `
        <div class="data-sync-dialog">
            <div class="data-sync-tabs" style="display: flex; gap: 8px; margin-bottom: 12px;">
                <button class="btn btn-secondary btn-sm" onclick="window.switchDataSyncTab('webdav')" ${
                  selectedTab === "webdav" ? "disabled" : ""
                }>☁️ WebDAV</button>
                <button class="btn btn-secondary btn-sm" onclick="window.switchDataSyncTab('local')" ${
                  selectedTab === "local" ? "disabled" : ""
                }>💾 ${t("backup.local.title")}</button>
                <button class="btn btn-secondary btn-sm" onclick="window.switchDataSyncTab('s3')" ${
                  selectedTab === "s3" ? "disabled" : ""
                }>🪣 ${t("backup.s3.title")}</button>
            </div>

            ${renderActiveTabContent()}
        </div>
    `;

  showModal(`🔄 ${t("webdav.dataSync")}`, content);

  window.switchDataSyncTab = (nextTab) => {
    showDataSyncDialog(nextTab);
  };
}

function renderActiveTabContent() {
    if (selectedTab === 'local') return renderLocalTab();
    if (selectedTab === 's3') return renderS3Tab();
    return renderWebDAVTab();
}

function renderWebDAVTab() {
    return `
        <div class="webdav-settings">
            <div class="form-group">
                <label><span class="required-mark">*</span>${t('webdav.serverUrl')}</label>
                <input type="text" id="dataSyncUrl" class="form-input"
                       placeholder="https://dav.example.com/remote.php/dav/files/username/"
                       value="${currentWebDAVConfig.url}">
                <small style="color: #888; font-size: 12px; margin-top: 5px;">${t('webdav.serverUrlHelp')}</small>
            </div>
            <div class="form-row" style="gap: 10px;">
                <div class="form-group" style="flex: 1;">
                    <label><span class="required-mark">*</span>${t('webdav.username')}</label>
                    <input type="text" id="dataSyncUsername" class="form-input"
                           placeholder="${t('webdav.usernamePlaceholder')}"
                           value="${currentWebDAVConfig.username}">
                </div>
                <div class="form-group" style="flex: 1;">
                    <label><span class="required-mark">*</span>${t('webdav.password')}</label>
                    <input type="password" id="dataSyncPassword" class="form-input"
                           placeholder="${t('webdav.passwordPlaceholder')}"
                           value="${currentWebDAVConfig.password}">
                </div>
            </div>
        </div>
        <div class="data-sync-actions">
            <button class="btn btn-secondary" onclick="window.testDataSyncConnection()">🔍 ${t('webdav.testConnection')}</button>
            <button class="btn btn-secondary" onclick="window.saveDataSyncConfig()">💾 ${t('webdav.saveConfig')}</button>
            <button class="btn btn-primary" onclick="window.backupToProviderFromDialog('webdav')">📤 ${t('webdav.backup')}</button>
            <button class="btn btn-primary" onclick="window.openBackupManagerFromDialog('webdav')">📂 ${t('webdav.backupManager')}</button>
        </div>
    `;
}

function renderLocalTab() {
    return `
        <div class="local-settings">
            <div class="form-group">
                <label><span class="required-mark">*</span>${t('backup.local.dir')}</label>
                <div class="form-row" style="gap: 10px;">
                    <input type="text" id="backupLocalDir" class="form-input" value="${currentBackupConfig.local.dir}" placeholder="${t('backup.local.dirPlaceholder')}">
                    <button class="btn btn-secondary" onclick="window.selectBackupLocalDir()">📁 ${t('backup.local.chooseDir')}</button>
                </div>
            </div>
        </div>
        <div class="data-sync-actions">
            <button class="btn btn-secondary" onclick="window.saveLocalBackupConfig()">💾 ${t('backup.saveConfig')}</button>
            <button class="btn btn-primary" onclick="window.backupToProviderFromDialog('local')">📤 ${t('backup.backup')}</button>
            <button class="btn btn-primary" onclick="window.openBackupManagerFromDialog('local')">📂 ${t('backup.backupManager')}</button>
        </div>
    `;
}

function renderS3Tab() {
    const s3 = currentBackupConfig.s3;
    return `
        <div class="s3-settings">
            <div class="form-group">
                <label><span class="required-mark">*</span>${t('backup.s3.endpoint')}</label>
                <input type="text" id="backupS3Endpoint" class="form-input" value="${s3.endpoint}" placeholder="play.min.io">
            </div>
            <div class="form-row" style="gap: 10px;">
                <div class="form-group" style="flex: 1;">
                    <label>${t('backup.s3.region')}</label>
                    <input type="text" id="backupS3Region" class="form-input" value="${s3.region}" placeholder="us-east-1">
                </div>
                <div class="form-group" style="flex: 1;">
                    <label><span class="required-mark">*</span>${t('backup.s3.bucket')}</label>
                    <input type="text" id="backupS3Bucket" class="form-input" value="${s3.bucket}">
                </div>
            </div>
            <div class="form-group">
                <label>${t('backup.s3.prefix')}</label>
                <input type="text" id="backupS3Prefix" class="form-input" value="${s3.prefix}" placeholder="ccNexus/">
            </div>
            <div class="form-row" style="gap: 10px;">
                <div class="form-group" style="flex: 1;">
                    <label><span class="required-mark">*</span>${t('backup.s3.accessKey')}</label>
                    <input type="text" id="backupS3AccessKey" class="form-input" value="${s3.accessKey}">
                </div>
                <div class="form-group" style="flex: 1;">
                    <label><span class="required-mark">*</span>${t('backup.s3.secretKey')}</label>
                    <input type="password" id="backupS3SecretKey" class="form-input" value="${s3.secretKey}">
                </div>
            </div>
            <div class="form-group">
                <label>${t('backup.s3.sessionToken')}</label>
                <input type="text" id="backupS3SessionToken" class="form-input" value="${s3.sessionToken}">
            </div>
            <div class="form-row" style="gap: 10px;">
                <div class="form-group" style="flex: 1;">
                    <label class="toggle-label">${t('backup.s3.useSSL')}</label>
                    <input type="checkbox" id="backupS3UseSSL" ${s3.useSSL ? 'checked' : ''}>
                </div>
                <div class="form-group" style="flex: 1;">
                    <label class="toggle-label">${t('backup.s3.forcePathStyle')}</label>
                    <input type="checkbox" id="backupS3ForcePathStyle" ${s3.forcePathStyle ? 'checked' : ''}>
                </div>
            </div>
        </div>
        <div class="data-sync-actions">
            <button class="btn btn-secondary" onclick="window.testS3ConnectionFromDialog()">🔍 ${t('backup.s3.testConnection')}</button>
            <button class="btn btn-secondary" onclick="window.saveS3BackupConfig()">💾 ${t('backup.saveConfig')}</button>
            <button class="btn btn-primary" onclick="window.backupToProviderFromDialog('s3')">📤 ${t('backup.backup')}</button>
            <button class="btn btn-primary" onclick="window.openBackupManagerFromDialog('s3')">📂 ${t('backup.backupManager')}</button>
        </div>
    `;
}

// Save WebDAV config from dialog
window.saveDataSyncConfig = async function () {
  const url = document.getElementById("dataSyncUrl")?.value.trim() || "";
  const username =
    document.getElementById("dataSyncUsername")?.value.trim() || "";
  const password =
    document.getElementById("dataSyncPassword")?.value.trim() || "";

  // Validate required fields
  if (!url) {
    showNotification(t("webdav.urlRequired"), "error");
    return;
  }
  if (!username) {
    showNotification(t("webdav.usernameRequired"), "error");
    return;
  }
  if (!password) {
    showNotification(t("webdav.passwordRequired"), "error");
    return;
  }

  // Check if connection test passed
  if (!connectionTestPassed) {
    showNotification(t("webdav.testRequired"), "error");
    return;
  }

  try {
    await updateWebDAVConfig(url, username, password);
    await window.go.main.App.UpdateBackupProvider("webdav");
    currentWebDAVConfig = { url, username, password };
    connectionTestPassed = false; // Reset after save
    showNotification(t("webdav.configSaved"), "success");
  } catch (error) {
    showNotification(
      t("webdav.configSaveFailed") + ": " + translateError(error),
      "error"
    );
  }
};

// Test connection from dialog
window.testDataSyncConnection = async function () {
  const url = document.getElementById("dataSyncUrl")?.value.trim() || "";
  const username =
    document.getElementById("dataSyncUsername")?.value.trim() || "";
  const password =
    document.getElementById("dataSyncPassword")?.value.trim() || "";

  // Validate required fields
  if (!url) {
    showNotification(t("webdav.urlRequired"), "error");
    return;
  }
  if (!username) {
    showNotification(t("webdav.usernameRequired"), "error");
    return;
  }
  if (!password) {
    showNotification(t("webdav.passwordRequired"), "error");
    return;
  }

  try {
    // Test connection without saving
    const resultStr = await window.go.main.App.TestWebDAVConnection(
      url,
      username,
      password
    );
    const result = JSON.parse(resultStr);
    if (result.success) {
      connectionTestPassed = true;
      showNotification(t("webdav.connectionSuccess"), "success");
    } else {
      connectionTestPassed = false;
      showNotification(t("webdav.connectionFailedWithRecommend"), "error");
    }
  } catch (error) {
    connectionTestPassed = false;
    showNotification(t("webdav.connectionFailedWithRecommend"), "error");
  }
};

function readS3ConfigFromDialog() {
  return {
    endpoint: document.getElementById("backupS3Endpoint")?.value.trim() || "",
    region: document.getElementById("backupS3Region")?.value.trim() || "",
    bucket: document.getElementById("backupS3Bucket")?.value.trim() || "",
    prefix: document.getElementById("backupS3Prefix")?.value.trim() || "",
    accessKey: document.getElementById("backupS3AccessKey")?.value.trim() || "",
    secretKey: document.getElementById("backupS3SecretKey")?.value || "",
    sessionToken:
      document.getElementById("backupS3SessionToken")?.value.trim() || "",
    useSSL: !!document.getElementById("backupS3UseSSL")?.checked,
    forcePathStyle: !!document.getElementById("backupS3ForcePathStyle")?.checked,
  };
}

window.selectBackupLocalDir = async function () {
  try {
    const dir = await window.go.main.App.SelectDirectory();
    if (!dir) return;
    const input = document.getElementById("backupLocalDir");
    if (input) input.value = dir;
  } catch (error) {
    showNotification(translateError(error), "error");
  }
};

window.saveLocalBackupConfig = async function () {
  const dir = document.getElementById("backupLocalDir")?.value.trim() || "";
  if (!dir) {
    showNotification(t("backup.local.dirRequired"), "error");
    return;
  }
  try {
    await window.go.main.App.UpdateLocalBackupDir(dir);
    showNotification(t("backup.configSaved"), "success");
  } catch (error) {
    showNotification(translateError(error), "error");
  }
};

window.saveS3BackupConfig = async function () {
  const s3 = readS3ConfigFromDialog();
  if (!s3.endpoint || !s3.bucket || !s3.accessKey || !s3.secretKey) {
    showNotification(t("backup.s3.requiredFields"), "error");
    return;
  }
  try {
    await window.go.main.App.UpdateS3BackupConfig(
      s3.endpoint,
      s3.region,
      s3.bucket,
      s3.prefix,
      s3.accessKey,
      s3.secretKey,
      s3.sessionToken,
      s3.useSSL,
      s3.forcePathStyle
    );
    showNotification(t("backup.configSaved"), "success");
  } catch (error) {
    showNotification(translateError(error), "error");
  }
};

window.testS3ConnectionFromDialog = async function () {
  const s3 = readS3ConfigFromDialog();
  if (!s3.endpoint || !s3.bucket || !s3.accessKey || !s3.secretKey) {
    showNotification(t("backup.s3.requiredFields"), "error");
    return;
  }
  try {
    const resultStr = await window.go.main.App.TestS3Connection(
      s3.endpoint,
      s3.region,
      s3.bucket,
      s3.prefix,
      s3.accessKey,
      s3.secretKey,
      s3.sessionToken,
      s3.useSSL,
      s3.forcePathStyle
    );
    const result = JSON.parse(resultStr);
    showNotification(result.message || "", result.success ? "success" : "error");
  } catch (error) {
    showNotification(translateError(error), "error");
  }
};

// Backup from dialog
window.backupToWebDAVFromDialog = async function () {
  await backupToProvider("webdav");
};

// Open backup manager from dialog
window.backupToProviderFromDialog = async function (provider = "webdav") {
  await backupToProvider(provider);
};

// Open backup manager from dialog
window.openBackupManagerFromDialog = async function (provider = "webdav") {
  await openBackupManager(provider);
};

// Close dialog
window.closeDataSyncDialog = function () {
  hideModal();
};

// Update WebDAV configuration
export async function updateWebDAVConfig(url, username, password) {
  await window.go.main.App.UpdateWebDAVConfig(url, username, password);
}

// Test WebDAV connection (deprecated - use direct call with parameters)
export async function testWebDAVConnection(url, username, password) {
  const resultStr = await window.go.main.App.TestWebDAVConnection(
    url,
    username,
    password
  );
  return JSON.parse(resultStr);
}

// Generate default backup filename
function generateBackupFilename() {
  const now = new Date();
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, "0");
  const day = String(now.getDate()).padStart(2, "0");
  const hours = String(now.getHours()).padStart(2, "0");
  const minutes = String(now.getMinutes()).padStart(2, "0");
  const seconds = String(now.getSeconds()).padStart(2, "0");

  return `ccNexus-${year}${month}${day}${hours}${minutes}${seconds}.db`;
}

function tBackup(provider, key) {
  if (provider === "webdav") return t(`webdav.${key}`);
  return t(`backup.${key}`);
}

async function listBackups(provider) {
  const resultStr = await window.go.main.App.ListBackups(provider);
  return JSON.parse(resultStr);
}

async function deleteBackups(provider, filenames) {
  try {
    await window.go.main.App.DeleteBackups(provider, filenames);
    showNotification(tBackup(provider, "deleteSuccess"), "success");
  } catch (error) {
    showNotification(translateError(error), "error");
  }
}

async function backupToProvider(provider) {
  const filename = await promptFilename(
    tBackup(provider, "enterBackupName"),
    generateBackupFilename()
  );
  if (!filename) return;
  try {
    await window.go.main.App.BackupToProvider(provider, filename);
    showNotification(tBackup(provider, "backupSuccess"), "success");
  } catch (error) {
    showNotification(translateError(error), "error");
  }
}

async function restoreFromProvider(provider, filename) {
  const conflictStr = await window.go.main.App.DetectBackupConflict(
    provider,
    filename
  );
  const conflictResult = JSON.parse(conflictStr);

  if (!conflictResult.success) {
    showNotification(
      t("webdav.conflictDetectionFailed") + ": " + (conflictResult.message || ""),
      "error"
    );
    return;
  }

  const conflicts = conflictResult.conflicts || [];
  let choice = "keep_local";
  if (conflicts.length > 0) {
    const selected = await showConflictDialog(conflicts);
    if (!selected) return;
    choice = selected;
  }

  try {
    await window.go.main.App.RestoreFromProvider(provider, filename, choice);
    showNotification(tBackup(provider, "restoreSuccess"), "success");
    window.location.reload();
  } catch (error) {
    showNotification(translateError(error), "error");
  }
}

// Backup to WebDAV
export async function backupToWebDAV() {
  await backupToProvider("webdav");
}

// Restore from WebDAV
export async function restoreFromWebDAV(filename) {
  await restoreFromProvider("webdav", filename);
}

// List WebDAV backups
export async function listWebDAVBackups() {
  return await listBackups("webdav");
}

// Delete WebDAV backups
export async function deleteWebDAVBackups(filenames) {
  await deleteBackups("webdav", filenames);
}

// Show backup manager
export async function openBackupManager(provider = "webdav") {
  const result = await listBackups(provider);

  if (!result.success) {
    showNotification(result.message, "error");
    return;
  }

  const backups = result.backups || [];

  const content = `
        <div class="backup-manager">
	            <div class="backup-manager-header">
	                <div class="backup-manager-actions">
	                    <button class="btn btn-secondary btn-sm" onclick="window.refreshBackupList()">🔄 ${tBackup(
                        provider,
                        "refresh"
                      )}</button>
	                    <button class="btn btn-danger btn-sm" onclick="window.deleteSelectedBackups()" ${
	                      backups.length === 0 ? "disabled" : ""
	                    }>🗑️ ${tBackup(provider, "deleteSelected")}</button>
	                </div>
	            </div>
	            <div class="backup-list-container">
	                ${
	                  backups.length === 0
	                    ? `<div class="empty-state">${tBackup(
                        provider,
                        "noBackups"
                      )}</div>`
	                    : renderBackupList(provider, backups)
	                }
	            </div>
	        </div>
	    `;

  showSubModal("📂 " + tBackup(provider, "backupManager"), content);

  // Set up global functions for backup manager
  window.refreshBackupList = async () => {
    try {
      const result = await listBackups(provider);
      if (result.success) {
        showNotification(tBackup(provider, "refreshSuccess"), "success");
        openBackupManager(provider);
      } else {
        showNotification(
          result.message || tBackup(provider, "refreshFailed"),
          "error"
        );
      }
    } catch (error) {
      showNotification(translateError(error), "error");
    }
  };

  window.deleteSelectedBackups = async () => {
    const checkboxes = document.querySelectorAll(".backup-checkbox:checked");
    const selectedFiles = Array.from(checkboxes).map(
      (cb) => cb.dataset.filename
    );

    if (selectedFiles.length === 0) {
      showNotification(tBackup(provider, "selectBackupsToDelete"), "warning");
      return;
    }

    const confirmed = await confirmAction(
      tBackup(provider, "confirmDelete").replace("{count}", selectedFiles.length)
    );

    if (!confirmed) {
      return;
    }

    await deleteBackups(provider, selectedFiles);
    openBackupManager(provider);
  };

  window.restoreBackup = async (filename) => {
    const confirmed = await confirmAction(
      tBackup(provider, "confirmRestore").replace("{filename}", filename)
    );

    if (!confirmed) {
      return;
    }

    hideSubModal();
    await restoreFromProvider(provider, filename);
  };

  window.deleteSingleBackup = async (filename) => {
    const confirmed = await confirmAction(
      tBackup(provider, "confirmDelete").replace("{count}", "1")
    );

    if (!confirmed) {
      return;
    }

    await deleteBackups(provider, [filename]);
    openBackupManager(provider);
  };

  window.closeBackupManager = () => {
    hideSubModal();
  };
}

// Render backup list
function renderBackupList(provider, backups) {
  return `
        <table class="backup-table">
            <thead>
                <tr>
                    <th width="35"><input type="checkbox" id="selectAllBackups" onchange="window.toggleAllBackups(this)"></th>
                    <th>${tBackup(provider, "filename")}</th>
                    <th width="110">${tBackup(provider, "actions")}</th>
                </tr>
            </thead>
            <tbody>
                ${backups
                  .map(
                    (backup) => `
                    <tr>
                        <td><input type="checkbox" class="backup-checkbox" data-filename="${
                          backup.filename
                        }"></td>
                        <td>
                            <div style="font-weight: 500; margin-bottom: 4px; word-break: break-all;">${
                              backup.filename
                            }</div>
                            <div style="font-size: 11px; color: #888;">${formatFileSize(
                              backup.size
                            )}</div>
                            <div style="font-size: 11px; color: #888;">${formatDateTime(
                              backup.modTime
                            )}</div>
                        </td>
                        <td>
	                            <div style="display: flex; flex-direction: column; gap: 4px;">
	                                <button class="btn btn-primary btn-sm" onclick="window.restoreBackup('${
	                                  backup.filename
	                                }')">↩️ ${tBackup(provider, "restore")}</button>
	                                <button class="btn btn-danger btn-sm" onclick="window.deleteSingleBackup('${
	                                  backup.filename
	                                }')">🗑️ ${tBackup(provider, "delete")}</button>
	                            </div>
	                        </td>
	                    </tr>
	                `
                  )
                  .join("")}
            </tbody>
        </table>
    `;
}

// Toggle all backups
window.toggleAllBackups = function (checkbox) {
  const checkboxes = document.querySelectorAll(".backup-checkbox");
  checkboxes.forEach((cb) => (cb.checked = checkbox.checked));
};

// Format file size
function formatFileSize(bytes) {
  if (bytes < 1024) return bytes + " B";
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(2) + " KB";
  return (bytes / (1024 * 1024)).toFixed(2) + " MB";
}

// Format date time
function formatDateTime(dateStr) {
  const date = new Date(dateStr);
  return date.toLocaleString();
}

// Show conflict dialog
async function showConflictDialog(conflicts) {
  return new Promise((resolve) => {
    // Build conflict details HTML
    const conflictDetailsHTML = conflicts
      .map((conflict) => {
        const fields = conflict.conflictFields || [];
        const fieldLabels = {
          apiUrl: t("webdav.apiUrl"),
          apiKey: t("webdav.apiKey"),
          enabled: t("webdav.enabled"),
          transformer: t("webdav.transformer"),
          model: t("webdav.model"),
          remark: t("webdav.remark"),
        };

        return `
                <div class="conflict-endpoint">
                    <div class="conflict-endpoint-header">
                        <strong>📍 ${conflict.endpointName}</strong>
                        <span class="conflict-badge">${fields.length} ${
          fields.length === 1 ? t("webdav.conflict") : t("webdav.conflicts")
        }</span>
                    </div>
                    <div class="conflict-endpoint-body">
                        <div class="conflict-fields">
                            ${fields
                              .map(
                                (field) => `
                                <div class="conflict-field-item">
                                    <span class="conflict-field-name">${
                                      fieldLabels[field] || field
                                    }:</span>
                                    <div class="conflict-field-values">
                                        <div class="conflict-value-local">
                                            <span class="conflict-value-label">${t(
                                              "webdav.local"
                                            )}:</span>
                                            <code>${formatFieldValue(
                                              conflict.localEndpoint[field]
                                            )}</code>
                                        </div>
                                        <div class="conflict-value-remote">
                                            <span class="conflict-value-label">${t(
                                              "webdav.remote"
                                            )}:</span>
                                            <code>${formatFieldValue(
                                              conflict.remoteEndpoint[field]
                                            )}</code>
                                        </div>
                                    </div>
                                </div>
                            `
                              )
                              .join("")}
                        </div>
                    </div>
                </div>
            `;
      })
      .join("");

    const content = `
            <div class="conflict-dialog-content">
                <button class="conflict-close-btn" onclick="window.resolveConflict(null)">×</button>
                <div class="conflict-header">
                    <span class="conflict-icon">⚠️</span>
                    <span class="conflict-title">${t(
                      "webdav.conflictTitle"
                    )}</span>
                </div>
                <div class="conflict-divider"></div>
                <div class="conflict-body">
                    <p class="conflict-message">
                        ${t("webdav.conflictDetected")}
                        <strong>${conflicts.length}</strong> ${
      conflicts.length > 1 ? t("webdav.endpointsHave") : t("webdav.endpointHas")
    }
                    </p>
                    <div class="conflict-details-container">
                        ${conflictDetailsHTML}
                    </div>
                    <div class="conflict-strategy-info">
                        <p><strong>${t("webdav.useRemote")}:</strong> ${t(
      "webdav.useRemoteDesc"
    )}</p>
                        <p><strong>${t("webdav.keepLocal")}:</strong> ${t(
      "webdav.keepLocalDesc"
    )}</p>
                    </div>
                </div>
                <div class="conflict-footer">
                    <button class="btn btn-primary" onclick="window.resolveConflict('remote')">${t(
                      "webdav.useRemote"
                    )}</button>
                    <button class="btn btn-secondary" onclick="window.resolveConflict('keep_local')">${t(
                      "webdav.keepLocal"
                    )}</button>
                </div>
            </div>
        `;

    showConfirmModal("", content, false);

    window.resolveConflict = (choice) => {
      hideConfirmModal();
      delete window.resolveConflict;
      resolve(choice);
    };
  });
}

// Format field value for display
function formatFieldValue(value) {
  if (value === null || value === undefined || value === "") {
    return "<em>empty</em>";
  }
  if (typeof value === "boolean") {
    return value ? "✓ Enabled" : "✗ Disabled";
  }
  // Handle numeric boolean (0/1) for enabled field
  if (typeof value === "number" && (value === 0 || value === 1)) {
    return value === 1 ? "✓ Enabled" : "✗ Disabled";
  }
  if (typeof value === "string" && value.length > 50) {
    return value.substring(0, 47) + "...";
  }
  return String(value);
}

// Prompt for filename
async function promptFilename(message, defaultValue) {
  return new Promise((resolve) => {
    const content = `
            <div class="prompt-dialog">
                <p><span class="required">*</span>${message}</p>
                <div class="prompt-body">
                    <input type="text" id="promptInput" class="form-input" value="${
                      defaultValue || ""
                    }" />
                </div>
                <div class="prompt-actions">
                    <button class="btn btn-primary" onclick="window.submitPrompt()">${t(
                      "common.ok"
                    )}</button>
                    <button class="btn btn-secondary" onclick="window.cancelPrompt()">${t(
                      "common.cancel"
                    )}</button>
                </div>
            </div>
        `;

    showSubModal("📝 " + t("webdav.filename"), content);

    // Focus input
    setTimeout(() => {
      const input = document.getElementById("promptInput");
      if (input) {
        input.focus();
        input.select();
      }
    }, 100);

    window.submitPrompt = () => {
      const input = document.getElementById("promptInput");
      const value = input?.value.trim();
      if (!value) {
        showNotification(t("webdav.filenameRequired"), "warning");
        input?.focus();
        return;
      }
      hideSubModal();
      delete window.submitPrompt;
      delete window.cancelPrompt;
      resolve(value);
    };

    window.cancelPrompt = () => {
      hideSubModal();
      delete window.submitPrompt;
      delete window.cancelPrompt;
      resolve(null);
    };
  });
}

// Confirm action
async function confirmAction(message) {
  return new Promise((resolve) => {
    const content = `
            <div class="confirm-dialog-content">
                <div class="confirm-body">
                    <div class="confirm-icon">
                        <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                            <path d="M12 9v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                        </svg>
                    </div>
                    <div class="confirm-content">
                        <h4 class="confirm-title">${t("common.confirm")}</h4>
                        <p class="confirm-message">${message}</p>
                    </div>
                </div>
                <div class="confirm-divider"></div>
                <div class="confirm-footer">
                    <button class="btn-confirm-delete" onclick="window.confirmYes()">${t(
                      "common.yes"
                    )}</button>
                    <button class="btn-confirm-cancel" onclick="window.confirmNo()">${t(
                      "common.no"
                    )}</button>
                </div>
            </div>
        `;

    showConfirmModal("", content);

    window.confirmYes = () => {
      hideConfirmModal();
      delete window.confirmYes;
      delete window.confirmNo;
      resolve(true);
    };

    window.confirmNo = () => {
      hideConfirmModal();
      delete window.confirmYes;
      delete window.confirmNo;
      resolve(false);
    };
  });
}

// Show notification
function showNotification(message, type = "info") {
  // Create notification element
  const notification = document.createElement("div");
  notification.className = `notification notification-${type}`;
  notification.textContent = message;

  // Add to body
  document.body.appendChild(notification);

  // Show notification
  setTimeout(() => notification.classList.add("show"), 10);

  // Hide and remove after 3 seconds
  setTimeout(() => {
    notification.classList.remove("show");
    setTimeout(() => notification.remove(), 300);
  }, 3000);
}
