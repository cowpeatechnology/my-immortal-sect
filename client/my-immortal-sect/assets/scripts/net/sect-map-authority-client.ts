import type { AuthorityCommandEnvelope, AuthoritySessionResponse } from './sect-map-authority-contract';

const DEFAULT_AUTHORITY_BASE_URL = 'http://127.0.0.1:8787';
const DEFAULT_TIMEOUT_MS = 2500;

export class SectMapAuthorityClient {
    constructor(
        private readonly baseUrl: string = DEFAULT_AUTHORITY_BASE_URL,
        private readonly timeoutMs: number = DEFAULT_TIMEOUT_MS,
    ) {}

    getBaseUrl(): string {
        return this.baseUrl;
    }

    async bootstrapSession(sessionId: string): Promise<AuthoritySessionResponse> {
        return this.request('/v1/authority/m1/session/bootstrap', {
            method: 'POST',
            body: JSON.stringify({ sessionId }),
        });
    }

    async getSnapshot(sessionId: string): Promise<AuthoritySessionResponse> {
        const encoded = encodeURIComponent(sessionId);
        return this.request(`/v1/authority/m1/session/snapshot?sessionId=${encoded}`, {
            method: 'GET',
        });
    }

    async executeCommand<TPayload>(sessionId: string, command: AuthorityCommandEnvelope<TPayload>): Promise<AuthoritySessionResponse> {
        return this.request('/v1/authority/m1/session/command', {
            method: 'POST',
            body: JSON.stringify({ sessionId, command }),
        });
    }

    private async request(path: string, init: RequestInit): Promise<AuthoritySessionResponse> {
        const controller = new AbortController();
        const timeoutId = globalThis.setTimeout(() => controller.abort(), this.timeoutMs);

        try {
            const response = await fetch(`${this.baseUrl}${path}`, {
                ...init,
                headers: {
                    'Content-Type': 'application/json',
                    ...(init.headers ?? {}),
                },
                signal: controller.signal,
            });

            const payload = (await response.json()) as AuthoritySessionResponse | { error?: string };
            if (!response.ok) {
                const message = 'error' in payload && payload.error ? payload.error : `authority_request_failed:${response.status}`;
                throw new Error(message);
            }
            return payload as AuthoritySessionResponse;
        } catch (error) {
            if (error instanceof Error && error.name === 'AbortError') {
                throw new Error('authority_request_timeout');
            }
            throw error;
        } finally {
            globalThis.clearTimeout(timeoutId);
        }
    }
}
