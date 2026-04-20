import { view } from 'cc';

export type RuntimeSafeArea = {
    left: number;
    right: number;
    top: number;
    bottom: number;
    width: number;
    height: number;
};

export type RuntimeScreenMetrics = {
    windowWidth: number;
    windowHeight: number;
    safeArea: RuntimeSafeArea;
    statusBarHeight: number;
    safeAreaFallbackApplied: boolean;
    source:
        | 'wechat-window-info'
        | 'wechat-system-info'
        | 'bytedance-window-info'
        | 'bytedance-system-info'
        | 'browser-window'
        | 'cocos-view';
};

type PlatformWindowInfo = {
    windowWidth?: unknown;
    windowHeight?: unknown;
    safeArea?: Partial<RuntimeSafeArea> | null;
    statusBarHeight?: unknown;
};

type MiniGameRuntimeApi = {
    getWindowInfo?: () => PlatformWindowInfo;
    getSystemInfoSync?: () => PlatformWindowInfo;
};

type RuntimeGlobal = typeof globalThis & {
    document?: {
        documentElement?: {
            clientWidth?: number;
            clientHeight?: number;
        };
    };
    innerWidth?: number;
    innerHeight?: number;
    tt?: MiniGameRuntimeApi;
    wx?: MiniGameRuntimeApi;
};

type ScreenMetricsOptions = {
    forceSafeAreaFallback?: boolean;
    windowInfoOverride?: PlatformWindowInfo | null;
};

function asPositiveNumber(value: unknown): number | null {
    if (typeof value !== 'number' || !Number.isFinite(value) || value <= 0) {
        return null;
    }

    return value;
}

function asNonNegativeNumber(value: unknown): number | null {
    if (typeof value !== 'number' || !Number.isFinite(value) || value < 0) {
        return null;
    }

    return value;
}

function readPlatformWindowInfo(): {
    info: PlatformWindowInfo | null;
    source: RuntimeScreenMetrics['source'] | null;
} {
    const runtimeGlobal = globalThis as RuntimeGlobal;
    const candidates: Array<{
        api: MiniGameRuntimeApi | undefined;
        windowInfoSource: RuntimeScreenMetrics['source'];
        systemInfoSource: RuntimeScreenMetrics['source'];
    }> = [
        {
            api: runtimeGlobal.wx,
            windowInfoSource: 'wechat-window-info',
            systemInfoSource: 'wechat-system-info',
        },
        {
            api: runtimeGlobal.tt,
            windowInfoSource: 'bytedance-window-info',
            systemInfoSource: 'bytedance-system-info',
        },
    ];

    for (const candidate of candidates) {
        const api = candidate.api;
        if (!api) {
            continue;
        }

        if (typeof api.getWindowInfo === 'function') {
            try {
                return {
                    info: api.getWindowInfo(),
                    source: candidate.windowInfoSource,
                };
            } catch {
                // Fall through to system info if the runtime blocks getWindowInfo.
            }
        }

        if (typeof api.getSystemInfoSync === 'function') {
            try {
                return {
                    info: api.getSystemInfoSync(),
                    source: candidate.systemInfoSource,
                };
            } catch {
                // Ignore and continue into the generic browser fallback.
            }
        }
    }

    return {
        info: null,
        source: null,
    };
}

function readFallbackWindowSize(): {
    width: number;
    height: number;
    source: RuntimeScreenMetrics['source'];
} {
    const runtimeGlobal = globalThis as RuntimeGlobal;
    const browserWidth =
        asPositiveNumber(runtimeGlobal.innerWidth) ??
        asPositiveNumber(runtimeGlobal.document?.documentElement?.clientWidth);
    const browserHeight =
        asPositiveNumber(runtimeGlobal.innerHeight) ??
        asPositiveNumber(runtimeGlobal.document?.documentElement?.clientHeight);

    if (browserWidth && browserHeight) {
        return {
            width: browserWidth,
            height: browserHeight,
            source: 'browser-window',
        };
    }

    const visibleSize = view.getVisibleSize();
    return {
        width: Math.max(1, visibleSize.width),
        height: Math.max(1, visibleSize.height),
        source: 'cocos-view',
    };
}

function normalizeSafeArea(
    safeArea: Partial<RuntimeSafeArea> | null | undefined,
    windowWidth: number,
    windowHeight: number,
): RuntimeSafeArea | null {
    if (!safeArea) {
        return null;
    }

    const left = asNonNegativeNumber(safeArea.left);
    const right = asPositiveNumber(safeArea.right);
    const top = asNonNegativeNumber(safeArea.top);
    const bottom = asPositiveNumber(safeArea.bottom);
    const width = asPositiveNumber(safeArea.width);
    const height = asPositiveNumber(safeArea.height);

    if (
        left === null ||
        right === null ||
        top === null ||
        bottom === null ||
        width === null ||
        height === null
    ) {
        return null;
    }

    const normalizedLeft = Math.min(Math.max(0, left), windowWidth);
    const normalizedRight = Math.min(Math.max(normalizedLeft, right), windowWidth);
    const normalizedTop = Math.min(Math.max(0, top), windowHeight);
    const normalizedBottom = Math.min(Math.max(normalizedTop, bottom), windowHeight);
    const normalizedWidth = Math.min(width, Math.max(0, normalizedRight - normalizedLeft));
    const normalizedHeight = Math.min(height, Math.max(0, normalizedBottom - normalizedTop));

    return {
        left: normalizedLeft,
        right: normalizedRight,
        top: normalizedTop,
        bottom: normalizedBottom,
        width: normalizedWidth,
        height: normalizedHeight,
    };
}

function buildFallbackSafeArea(
    windowWidth: number,
    windowHeight: number,
    statusBarHeight: number,
): RuntimeSafeArea {
    const top = Math.min(Math.max(0, statusBarHeight), windowHeight);
    return {
        left: 0,
        right: windowWidth,
        top,
        bottom: windowHeight,
        width: windowWidth,
        height: Math.max(0, windowHeight - top),
    };
}

export function getRuntimeScreenMetrics(options: ScreenMetricsOptions = {}): RuntimeScreenMetrics {
    const platformResult = options.windowInfoOverride
        ? {
              info: options.windowInfoOverride,
              source: null,
          }
        : readPlatformWindowInfo();
    const fallbackWindowSize = readFallbackWindowSize();

    const windowWidth = asPositiveNumber(platformResult.info?.windowWidth) ?? fallbackWindowSize.width;
    const windowHeight = asPositiveNumber(platformResult.info?.windowHeight) ?? fallbackWindowSize.height;
    const statusBarHeight = asNonNegativeNumber(platformResult.info?.statusBarHeight) ?? 0;
    const nativeSafeArea = options.forceSafeAreaFallback
        ? null
        : normalizeSafeArea(platformResult.info?.safeArea, windowWidth, windowHeight);

    return {
        windowWidth,
        windowHeight,
        safeArea: nativeSafeArea ?? buildFallbackSafeArea(windowWidth, windowHeight, statusBarHeight),
        statusBarHeight,
        safeAreaFallbackApplied: !nativeSafeArea,
        source: platformResult.source ?? fallbackWindowSize.source,
    };
}
