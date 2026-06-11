import { BrowserWindow, app } from 'electrobun';

// NanoVMS Desktop Application — Electrobun Implementation
// Provides a native webview UI for managing NanoVMS VMs and sandboxes.

const WINDOW_WIDTH = 1280;
const WINDOW_HEIGHT = 800;
const DEV_URL = 'http://localhost:5173';
const PROD_URL = 'https://app.nanovms.dev';

function createMainWindow() {
  const win = new BrowserWindow({
    title: 'NanoVMS Desktop',
    width: WINDOW_WIDTH,
    height: WINDOW_HEIGHT,
    minWidth: 900,
    minHeight: 600,
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true,
    },
  });

  // Load the appropriate URL based on environment
  const url = process.env.NODE_ENV === 'development' ? DEV_URL : PROD_URL;
  win.loadURL(url);

  // Handle window events
  win.on('closed', () => {
    console.log('[nanovms-desktop] main window closed');
  });

  return win;
}

// IPC handlers for NanoVMS backend communication
function registerIpcHandlers() {
  app.on('ipc:deploy', async (event, payload) => {
    try {
      const { tier, config } = payload;
      console.log(`[nanovms-desktop] deploy request: tier=${tier}`);
      // TODO: invoke native nvms CLI or REST API
      event.reply('ipc:deploy:reply', { ok: true, tier, config });
    } catch (err) {
      event.reply('ipc:deploy:reply', { ok: false, error: err.message });
    }
  });

  app.on('ipc:list-vms', async (event) => {
    try {
      // TODO: fetch from nvms API
      event.reply('ipc:list-vms:reply', { ok: true, vms: [] });
    } catch (err) {
      event.reply('ipc:list-vms:reply', { ok: false, error: err.message });
    }
  });

  app.on('ipc:status', async (event) => {
    event.reply('ipc:status:reply', {
      ok: true,
      version: app.getVersion(),
      platform: process.platform,
    });
  });
}

// Application lifecycle
app.on('ready', () => {
  console.log('[nanovms-desktop] app ready');
  const mainWindow = createMainWindow();
  registerIpcHandlers();

  // Expose window reference for testing
  global.mainWindow = mainWindow;
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('activate', () => {
  if (BrowserWindow.getAllWindows().length === 0) {
    createMainWindow();
  }
});

// Graceful shutdown
process.on('SIGTERM', () => {
  console.log('[nanovms-desktop] SIGTERM received, shutting down');
  app.quit();
});

process.on('SIGINT', () => {
  console.log('[nanovms-desktop] SIGINT received, shutting down');
  app.quit();
});

export { createMainWindow, registerIpcHandlers };
