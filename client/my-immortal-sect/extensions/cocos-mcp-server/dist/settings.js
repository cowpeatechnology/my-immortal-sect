"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || (function () {
    var ownKeys = function(o) {
        ownKeys = Object.getOwnPropertyNames || function (o) {
            var ar = [];
            for (var k in o) if (Object.prototype.hasOwnProperty.call(o, k)) ar[ar.length] = k;
            return ar;
        };
        return ownKeys(o);
    };
    return function (mod) {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
Object.defineProperty(exports, "__esModule", { value: true });
exports.DEFAULT_TOOL_MANAGER_SETTINGS = exports.DEFAULT_SETTINGS = void 0;
exports.normalizeSettings = normalizeSettings;
exports.readSettings = readSettings;
exports.saveSettings = saveSettings;
exports.readToolManagerSettings = readToolManagerSettings;
exports.saveToolManagerSettings = saveToolManagerSettings;
exports.exportToolConfiguration = exportToolConfiguration;
exports.importToolConfiguration = importToolConfiguration;
const fs = __importStar(require("fs"));
const path = __importStar(require("path"));
const DEFAULT_SETTINGS = {
    port: 9527,
    autoStart: false,
    enableDebugLog: false,
    debugLog: false,
    allowedOrigins: ['*'],
    maxConnections: 10
};
exports.DEFAULT_SETTINGS = DEFAULT_SETTINGS;
const DEFAULT_TOOL_MANAGER_SETTINGS = {
    configurations: [],
    currentConfigId: '',
    maxConfigSlots: 5
};
exports.DEFAULT_TOOL_MANAGER_SETTINGS = DEFAULT_TOOL_MANAGER_SETTINGS;
function getSettingsPath() {
    return path.join(Editor.Project.path, 'settings', 'mcp-server.json');
}
function getToolManagerSettingsPath() {
    return path.join(Editor.Project.path, 'settings', 'tool-manager.json');
}
function ensureSettingsDir() {
    const settingsDir = path.dirname(getSettingsPath());
    if (!fs.existsSync(settingsDir)) {
        fs.mkdirSync(settingsDir, { recursive: true });
    }
}
function normalizeSettings(rawSettings = {}) {
    var _a, _b;
    const enableDebugLog = (_b = (_a = rawSettings.enableDebugLog) !== null && _a !== void 0 ? _a : rawSettings.debugLog) !== null && _b !== void 0 ? _b : DEFAULT_SETTINGS.enableDebugLog;
    const normalizedPort = typeof rawSettings.port === 'number' && Number.isInteger(rawSettings.port) && rawSettings.port > 0 && rawSettings.port <= 65535
        ? rawSettings.port
        : DEFAULT_SETTINGS.port;
    const normalizedMaxConnections = typeof rawSettings.maxConnections === 'number' && Number.isInteger(rawSettings.maxConnections) && rawSettings.maxConnections > 0
        ? rawSettings.maxConnections
        : DEFAULT_SETTINGS.maxConnections;
    return Object.assign(Object.assign(Object.assign({}, DEFAULT_SETTINGS), rawSettings), { port: normalizedPort, autoStart: typeof rawSettings.autoStart === 'boolean' ? rawSettings.autoStart : DEFAULT_SETTINGS.autoStart, enableDebugLog, debugLog: enableDebugLog, allowedOrigins: Array.isArray(rawSettings.allowedOrigins) && rawSettings.allowedOrigins.length > 0
            ? rawSettings.allowedOrigins
            : DEFAULT_SETTINGS.allowedOrigins, maxConnections: normalizedMaxConnections });
}
function readSettings() {
    try {
        ensureSettingsDir();
        const settingsFile = getSettingsPath();
        if (fs.existsSync(settingsFile)) {
            const content = fs.readFileSync(settingsFile, 'utf8');
            return normalizeSettings(JSON.parse(content));
        }
    }
    catch (e) {
        console.error('Failed to read settings:', e);
    }
    return normalizeSettings();
}
function saveSettings(settings) {
    try {
        ensureSettingsDir();
        const settingsFile = getSettingsPath();
        fs.writeFileSync(settingsFile, JSON.stringify(normalizeSettings(settings), null, 2));
    }
    catch (e) {
        console.error('Failed to save settings:', e);
        throw e;
    }
}
// 工具管理器设置相关函数
function readToolManagerSettings() {
    try {
        ensureSettingsDir();
        const settingsFile = getToolManagerSettingsPath();
        if (fs.existsSync(settingsFile)) {
            const content = fs.readFileSync(settingsFile, 'utf8');
            return Object.assign(Object.assign({}, DEFAULT_TOOL_MANAGER_SETTINGS), JSON.parse(content));
        }
    }
    catch (e) {
        console.error('Failed to read tool manager settings:', e);
    }
    return DEFAULT_TOOL_MANAGER_SETTINGS;
}
function saveToolManagerSettings(settings) {
    try {
        ensureSettingsDir();
        const settingsFile = getToolManagerSettingsPath();
        fs.writeFileSync(settingsFile, JSON.stringify(settings, null, 2));
    }
    catch (e) {
        console.error('Failed to save tool manager settings:', e);
        throw e;
    }
}
function exportToolConfiguration(config) {
    return JSON.stringify(config, null, 2);
}
function importToolConfiguration(configJson) {
    try {
        const config = JSON.parse(configJson);
        // 验证配置格式
        if (!config.id || !config.name || !Array.isArray(config.tools)) {
            throw new Error('Invalid configuration format');
        }
        return config;
    }
    catch (e) {
        console.error('Failed to parse tool configuration:', e);
        throw new Error('Invalid JSON format or configuration structure');
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoic2V0dGluZ3MuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyIuLi9zb3VyY2Uvc2V0dGluZ3MudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6Ijs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7O0FBa0NBLDhDQXFCQztBQUVELG9DQVlDO0FBRUQsb0NBU0M7QUFHRCwwREFZQztBQUVELDBEQVNDO0FBRUQsMERBRUM7QUFFRCwwREFZQztBQTVIRCx1Q0FBeUI7QUFDekIsMkNBQTZCO0FBRzdCLE1BQU0sZ0JBQWdCLEdBQXNCO0lBQ3hDLElBQUksRUFBRSxJQUFJO0lBQ1YsU0FBUyxFQUFFLEtBQUs7SUFDaEIsY0FBYyxFQUFFLEtBQUs7SUFDckIsUUFBUSxFQUFFLEtBQUs7SUFDZixjQUFjLEVBQUUsQ0FBQyxHQUFHLENBQUM7SUFDckIsY0FBYyxFQUFFLEVBQUU7Q0FDckIsQ0FBQztBQW1ITyw0Q0FBZ0I7QUFqSHpCLE1BQU0sNkJBQTZCLEdBQXdCO0lBQ3ZELGNBQWMsRUFBRSxFQUFFO0lBQ2xCLGVBQWUsRUFBRSxFQUFFO0lBQ25CLGNBQWMsRUFBRSxDQUFDO0NBQ3BCLENBQUM7QUE2R3lCLHNFQUE2QjtBQTNHeEQsU0FBUyxlQUFlO0lBQ3BCLE9BQU8sSUFBSSxDQUFDLElBQUksQ0FBQyxNQUFNLENBQUMsT0FBTyxDQUFDLElBQUksRUFBRSxVQUFVLEVBQUUsaUJBQWlCLENBQUMsQ0FBQztBQUN6RSxDQUFDO0FBRUQsU0FBUywwQkFBMEI7SUFDL0IsT0FBTyxJQUFJLENBQUMsSUFBSSxDQUFDLE1BQU0sQ0FBQyxPQUFPLENBQUMsSUFBSSxFQUFFLFVBQVUsRUFBRSxtQkFBbUIsQ0FBQyxDQUFDO0FBQzNFLENBQUM7QUFFRCxTQUFTLGlCQUFpQjtJQUN0QixNQUFNLFdBQVcsR0FBRyxJQUFJLENBQUMsT0FBTyxDQUFDLGVBQWUsRUFBRSxDQUFDLENBQUM7SUFDcEQsSUFBSSxDQUFDLEVBQUUsQ0FBQyxVQUFVLENBQUMsV0FBVyxDQUFD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LENBQUM7UUFDRCxNQUFNLE1BQU0sR0FBRyxJQUFJLENBQUMsS0FBSyxDQUFDLFVBQVUsQ0FBQyxDQUFDO1FBQ3RDLFNBQVM7UUFDVCxJQUFJLENBQUMsTUFBTSxDQUFDLEVBQUUsSUFBSSxDQUFDLE1BQU0sQ0FBQyxJQUFJLElBQUksQ0FBQyxLQUFLLENBQUMsT0FBTyxDQUFDLE1BQU0sQ0FBQyxLQUFLLENBQUMsRUFBRSxDQUFDO1lBQzdELE1BQU0sSUFBSSxLQUFLLENBQUMsOEJBQThCLENBQUMsQ0FBQztRQUNwRCxDQUFDO1FBQ0QsT0FBTyxNQUFNLENBQUM7SUFDbEIsQ0FBQztJQUFDLE9BQU8sQ0FBQyxFQUFFLENBQUM7UUFDVCxPQUFPLENBQUMsS0FBSyxDQUFDLHFDQUFxQyxFQUFFLENBQUMsQ0FBQyxDQUFDO1FBQ3hELE1BQU0sSUFBSSxLQUFLLENBQUMsZ0RBQWdELENBQUMsQ0FBQztJQUN0RSxDQUFDO0FBQ0wsQ0FBQyIsInNvdXJjZXNDb250ZW50IjpbImltcG9ydCAqIGFzIGZzIGZyb20gJ2ZzJztcbmltcG9ydCAqIGFzIHBhdGggZnJvbSAncGF0aCc7XG5pbXBvcnQgeyBNQ1BTZXJ2ZXJTZXR0aW5ncywgVG9vbE1hbmFnZXJTZXR0aW5ncywgVG9vbENvbmZpZ3VyYXRpb24sIFRvb2xDb25maWcgfSBmcm9tICcuL3R5cGVzJztcblxuY29uc3QgREVGQVVMVF9TRVRUSU5HUzogTUNQU2VydmVyU2V0dGluZ3MgPSB7XG4gICAgcG9ydDogOTUyNyxcbiAgICBhdXRvU3RhcnQ6IGZhbHNlLFxuICAgIGVuYWJsZURlYnVnTG9nOiBmYWxzZSxcbiAgICBkZWJ1Z0xvZzogZmFsc2UsXG4gICAgYWxsb3dlZE9yaWdpbnM6IFsnKiddLFxuICAgIG1heENvbm5lY3Rpb25zOiAxMFxufTtcblxuY29uc3QgREVGQVVMVF9UT09MX01BTkFHRVJfU0VUVElOR1M6IFRvb2xNYW5hZ2VyU2V0dGluZ3MgPSB7XG4gICAgY29uZmlndXJhdGlvbnM6IFtdLFxuICAgIGN1cnJlbnRDb25maWdJZDogJycsXG4gICAgbWF4Q29uZmlnU2xvdHM6IDVcbn07XG5cbmZ1bmN0aW9uIGdldFNldHRpbmdzUGF0aCgpOiBzdHJpbmcge1xuICAgIHJldHVybiBwYXRoLmpvaW4oRWRpdG9yLlByb2plY3QucGF0aCwgJ3NldHRpbmdzJywgJ21jcC1zZXJ2ZXIuanNvbicpO1xufVxuXG5mdW5jdGlvbiBnZXRUb29sTWFuYWdlclNldHRpbmdzUGF0aCgpOiBzdHJpbmcge1xuICAgIHJldHVybiBwYXRoLmpvaW4oRWRpdG9yLlByb2plY3QucGF0aCwgJ3NldHRpbmdzJywgJ3Rvb2wtbWFuYWdlci5qc29uJyk7XG59XG5cbmZ1bmN0aW9uIGVuc3VyZVNldHRpbmdzRGlyKCk6IHZvaWQge1xuICAgIGNvbnN0IHNldHRpbmdzRGlyID0gcGF0aC5kaXJuYW1lKGdldFNldHRpbmdzUGF0aCgpKTtcbiAgICBpZiAoIWZzLmV4aXN0c1N5bmMoc2V0dGluZ3NEaXIpKSB7XG4gICAgICAgIGZzLm1rZGlyU3luYyhzZXR0aW5nc0RpciwgeyByZWN1cnNpdmU6IHRydWUgfSk7XG4gICAgfVxufVxuXG5leHBvcnQgZnVuY3Rpb24gbm9ybWFsaXplU2V0dGluZ3MocmF3U2V0dGluZ3M6IFBhcnRpYWw8TUNQU2VydmVyU2V0dGluZ3M+ICYgeyBkZWJ1Z0xvZz86IGJvb2xlYW4gfSA9IHt9KTogTUNQU2VydmVyU2V0dGluZ3Mge1xuICAgIGNvbnN0IGVuYWJsZURlYnVnTG9nID0gcmF3U2V0dGluZ3MuZW5hYmxlRGVidWdMb2cgPz8gcmF3U2V0dGluZ3MuZGVidWdMb2cgPz8gREVGQVVMVF9TRVRUSU5HUy5lbmFibGVEZWJ1Z0xvZztcbiAgICBjb25zdCBub3JtYWxpemVkUG9ydCA9IHR5cGVvZiByYXdTZXR0aW5ncy5wb3J0ID09PSAnbnVtYmVyJyAmJiBOdW1iZXIuaXNJbnRlZ2VyKHJhd1NldHRpbmdzLnBvcnQpICYmIHJhd1NldHRpbmdzLnBvcnQgPiAwICYmIHJhd1NldHRpbmdzLnBvcnQgPD0gNjU1MzVcbiAgICAgICAgPyByYXdTZXR0aW5ncy5wb3J0XG4gICAgICAgIDogREVGQVVMVF9TRVRUSU5HUy5wb3J0O1xuICAgIGNvbnN0IG5vcm1hbGl6ZWRNYXhDb25uZWN0aW9ucyA9IHR5cGVvZiByYXdTZXR0aW5ncy5tYXhDb25uZWN0aW9ucyA9PT0gJ251bWJlcicgJiYgTnVtYmVyLmlzSW50ZWdlcihyYXdTZXR0aW5ncy5tYXhDb25uZWN0aW9ucykgJiYgcmF3U2V0dGluZ3MubWF4Q29ubmVjdGlvbnMgPiAwXG4gICAgICAgID8gcmF3U2V0dGluZ3MubWF4Q29ubmVjdGlvbnNcbiAgICAgICAgOiBERUZBVUxUX1NFVFRJTkdTLm1heENvbm5lY3Rpb25zO1xuXG4gICAgcmV0dXJuIHtcbiAgICAgICAgLi4uREVGQVVMVF9TRVRUSU5HUyxcbiAgICAgICAgLi4ucmF3U2V0dGluZ3MsXG4gICAgICAgIHBvcnQ6IG5vcm1hbGl6ZWRQb3J0LFxuICAgICAgICBhdXRvU3RhcnQ6IHR5cGVvZiByYXdTZXR0aW5ncy5hdXRvU3RhcnQgPT09ICdib29sZWFuJyA/IHJhd1NldHRpbmdzLmF1dG9TdGFydCA6IERFRkFVTFRfU0VUVElOR1MuYXV0b1N0YXJ0LFxuICAgICAgICBlbmFibGVEZWJ1Z0xvZyxcbiAgICAgICAgZGVidWdMb2c6IGVuYWJsZURlYnVnTG9nLFxuICAgICAgICBhbGxvd2VkT3JpZ2luczogQXJyYXkuaXNBcnJheShyYXdTZXR0aW5ncy5hbGxvd2VkT3JpZ2lucykgJiYgcmF3U2V0dGluZ3MuYWxsb3dlZE9yaWdpbnMubGVuZ3RoID4gMFxuICAgICAgICAgICAgPyByYXdTZXR0aW5ncy5hbGxvd2VkT3JpZ2luc1xuICAgICAgICAgICAgOiBERUZBVUxUX1NFVFRJTkdTLmFsbG93ZWRPcmlnaW5zLFxuICAgICAgICBtYXhDb25uZWN0aW9uczogbm9ybWFsaXplZE1heENvbm5lY3Rpb25zLFxuICAgIH07XG59XG5cbmV4cG9ydCBmdW5jdGlvbiByZWFkU2V0dGluZ3MoKTogTUNQU2VydmVyU2V0dGluZ3Mge1xuICAgIHRyeSB7XG4gICAgICAgIGVuc3VyZVNldHRpbmdzRGlyKCk7XG4gICAgICAgIGNvbnN0IHNldHRpbmdzRmlsZSA9IGdldFNldHRpbmdzUGF0aCgpO1xuICAgICAgICBpZiAoZnMuZXhpc3RzU3luYyhzZXR0aW5nc0ZpbGUpKSB7XG4gICAgICAgICAgICBjb25zdCBjb250ZW50ID0gZnMucmVhZEZpbGVTeW5jKHNldHRpbmdzRmlsZSwgJ3V0ZjgnKTtcbiAgICAgICAgICAgIHJldHVybiBub3JtYWxpemVTZXR0aW5ncyhKU09OLnBhcnNlKGNvbnRlbnQpKTtcbiAgICAgICAgfVxuICAgIH0gY2F0Y2ggKGUpIHtcbiAgICAgICAgY29uc29sZS5lcnJvcignRmFpbGVkIHRvIHJlYWQgc2V0dGluZ3M6JywgZSk7XG4gICAgfVxuICAgIHJldHVybiBub3JtYWxpemVTZXR0aW5ncygpO1xufVxuXG5leHBvcnQgZnVuY3Rpb24gc2F2ZVNldHRpbmdzKHNldHRpbmdzOiBNQ1BTZXJ2ZXJTZXR0aW5ncyk6IHZvaWQge1xuICAgIHRyeSB7XG4gICAgICAgIGVuc3VyZVNldHRpbmdzRGlyKCk7XG4gICAgICAgIGNvbnN0IHNldHRpbmdzRmlsZSA9IGdldFNldHRpbmdzUGF0aCgpO1xuICAgICAgICBmcy53cml0ZUZpbGVTeW5jKHNldHRpbmdzRmlsZSwgSlNPTi5zdHJpbmdpZnkobm9ybWFsaXplU2V0dGluZ3Moc2V0dGluZ3MpLCBudWxsLCAyKSk7XG4gICAgfSBjYXRjaCAoZSkge1xuICAgICAgICBjb25zb2xlLmVycm9yKCdGYWlsZWQgdG8gc2F2ZSBzZXR0aW5nczonLCBlKTtcbiAgICAgICAgdGhyb3cgZTtcbiAgICB9XG59XG5cbi8vIOW3peWFt+euoeeQhuWZqOiuvue9ruebuOWFs+WHveaVsFxuZXhwb3J0IGZ1bmN0aW9uIHJlYWRUb29sTWFuYWdlclNldHRpbmdzKCk6IFRvb2xNYW5hZ2VyU2V0dGluZ3Mge1xuICAgIHRyeSB7XG4gICAgICAgIGVuc3VyZVNldHRpbmdzRGlyKCk7XG4gICAgICAgIGNvbnN0IHNldHRpbmdzRmlsZSA9IGdldFRvb2xNYW5hZ2VyU2V0dGluZ3NQYXRoKCk7XG4gICAgICAgIGlmIChmcy5leGlzdHNTeW5jKHNldHRpbmdzRmlsZSkpIHtcbiAgICAgICAgICAgIGNvbnN0IGNvbnRlbnQgPSBmcy5yZWFkRmlsZVN5bmMoc2V0dGluZ3NGaWxlLCAndXRmOCcpO1xuICAgICAgICAgICAgcmV0dXJuIHsgLi4uREVGQVVMVF9UT09MX01BTkFHRVJfU0VUVElOR1MsIC4uLkpTT04ucGFyc2UoY29udGVudCkgfTtcbiAgICAgICAgfVxuICAgIH0gY2F0Y2ggKGUpIHtcbiAgICAgICAgY29uc29sZS5lcnJvcignRmFpbGVkIHRvIHJlYWQgdG9vbCBtYW5hZ2VyIHNldHRpbmdzOicsIGUpO1xuICAgIH1cbiAgICByZXR1cm4gREVGQVVMVF9UT09MX01BTkFHRVJfU0VUVElOR1M7XG59XG5cbmV4cG9ydCBmdW5jdGlvbiBzYXZlVG9vbE1hbmFnZXJTZXR0aW5ncyhzZXR0aW5nczogVG9vbE1hbmFnZXJTZXR0aW5ncyk6IHZvaWQge1xuICAgIHRyeSB7XG4gICAgICAgIGVuc3VyZVNldHRpbmdzRGlyKCk7XG4gICAgICAgIGNvbnN0IHNldHRpbmdzRmlsZSA9IGdldFRvb2xNYW5hZ2VyU2V0dGluZ3NQYXRoKCk7XG4gICAgICAgIGZzLndyaXRlRmlsZVN5bmMoc2V0dGluZ3NGaWxlLCBKU09OLnN0cmluZ2lmeShzZXR0aW5ncywgbnVsbCwgMikpO1xuICAgIH0gY2F0Y2ggKGUpIHtcbiAgICAgICAgY29uc29sZS5lcnJvcignRmFpbGVkIHRvIHNhdmUgdG9vbCBtYW5hZ2VyIHNldHRpbmdzOicsIGUpO1xuICAgICAgICB0aHJvdyBlO1xuICAgIH1cbn1cblxuZXhwb3J0IGZ1bmN0aW9uIGV4cG9ydFRvb2xDb25maWd1cmF0aW9uKGNvbmZpZzogVG9vbENvbmZpZ3VyYXRpb24pOiBzdHJpbmcge1xuICAgIHJldHVybiBKU09OLnN0cmluZ2lmeShjb25maWcsIG51bGwsIDIpO1xufVxuXG5leHBvcnQgZnVuY3Rpb24gaW1wb3J0VG9vbENvbmZpZ3VyYXRpb24oY29uZmlnSnNvbjogc3RyaW5nKTogVG9vbENvbmZpZ3VyYXRpb24ge1xuICAgIHRyeSB7XG4gICAgICAgIGNvbnN0IGNvbmZpZyA9IEpTT04ucGFyc2UoY29uZmlnSnNvbik7XG4gICAgICAgIC8vIOmqjOivgemFjee9ruagvOW8j1xuICAgICAgICBpZiAoIWNvbmZpZy5pZCB8fCAhY29uZmlnLm5hbWUgfHwgIUFycmF5LmlzQXJyYXkoY29uZmlnLnRvb2xzKSkge1xuICAgICAgICAgICAgdGhyb3cgbmV3IEVycm9yKCdJbnZhbGlkIGNvbmZpZ3VyYXRpb24gZm9ybWF0Jyk7XG4gICAgICAgIH1cbiAgICAgICAgcmV0dXJuIGNvbmZpZztcbiAgICB9IGNhdGNoIChlKSB7XG4gICAgICAgIGNvbnNvbGUuZXJyb3IoJ0ZhaWxlZCB0byBwYXJzZSB0b29sIGNvbmZpZ3VyYXRpb246JywgZSk7XG4gICAgICAgIHRocm93IG5ldyBFcnJvcignSW52YWxpZCBKU09OIGZvcm1hdCBvciBjb25maWd1cmF0aW9uIHN0cnVjdHVyZScpO1xuICAgIH1cbn1cblxuZXhwb3J0IHsgREVGQVVMVF9TRVRUSU5HUywgREVGQVVMVF9UT09MX01BTkFHRVJfU0VUVElOR1MgfTtcbiJdfQ==